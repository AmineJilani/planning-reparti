/*
main.go
Point d'entrée d'un participant NET.

Chaque instance représente un site du réseau réparti.
Ce programme gère :
  - l'initialisation de l'état local du site ;
  - l'enregistrement auprès du routeur ;
  - les goroutines de surveillance (heartbeat, détection de panne) ;
  - la tentative de rejoindre le groupe si un site cible est spécifié ;
  - la lecture continue des messages entrants et leur routage vers les handlers.

Architecture :
  NET reçoit les messages du routeur sur stdin,
  les traite via les handlers dédiés,
  et communique avec CTL via des FIFOs.
*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"planning-reparti/internal/comm"
	netstate "planning-reparti/internal/net"
	"planning-reparti/internal/utils"
)

func main() {
	// paramètres de démarrage du site
	name := flag.String("n", "A", "nom du site")
	join := flag.String("join", "", "site à contacter pour rejoindre le groupe")
	succ := flag.String("succ", "", "successeur initial dans l'anneau")
	pred := flag.String("pred", "", "prédécesseur initial dans l'anneau")
	members := flag.String("members", "", "membres initiaux du groupe, séparés par des virgules")
	ctlIn := flag.String("ctl-in", "", "FIFO des messages CTL vers NET")
	ctlOut := flag.String("ctl-out", "", "FIFO des messages NET vers CTL")

	flag.Parse()

	// création de l'état local du site
	st := netstate.NewState(*name)
	logger := utils.NewLogger("net-" + *name)

	// initialisation des membres si fournis au démarrage
	/*if *members != "" {
		for _, m := range strings.Split(*members, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				st.Members[m] = true
			}
		}
	}*/
	if *members != "" {
		list := []string{}

		for _, m := range strings.Split(*members, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				st.Members[m] = true
				st.LastSeen[m] = time.Now()
				list = append(list, m)
			}
		}

		for i, m := range list {
			pred := list[(i-1+len(list))%len(list)]
			succ := list[(i+1)%len(list)]

			st.Topology[m] = netstate.NodeLinks{
				Pred: pred,
				Succ: succ,
			}
		}
	}

	// initialisation de la topologie locale
	st.JoinTarget = *join
	st.Successor = *succ
	st.Predecessor = *pred
	st.CtlOut = *ctlOut

	if st.Successor != "" || st.Predecessor != "" {
		st.Topology[st.Name] = netstate.NodeLinks{
			Pred: st.Predecessor,
			Succ: st.Successor,
		}
	}

	// un site est considéré membre s'il possède déjà un prédécesseur et un successeur
	if st.Successor != "" && st.Predecessor != "" {
		st.Joined = true
	}

	// goroutine de surveillance périodique des membres (PING toutes les 5 secondes)
	go heartbeatLoop(st, logger)

	// goroutine de détection de panne (vérification LastSeen toutes les 3 secondes)
	go failureDetector(st, logger)

	// si un site cible est spécifié, on tente de rejoindre le groupe
	if *join != "" {
		go retryJoin(st, logger)
	}

	logger.Info("main", "NET démarré")

	// enregistrement auprès du routeur pour construire la table de routage dynamique
	reg := buildRegister(st.Name)
	fmt.Fprintln(os.Stdout, reg)
	logger.Info("main", "REGISTER envoyé : "+reg)

	// courte attente pour laisser le routeur traiter le REGISTER
	time.Sleep(500 * time.Millisecond)

	// lecture des messages entrants depuis CTL via FIFO
	if *ctlIn != "" {
		go readCtlInput(st, logger, *ctlIn)
	}

	// boucle principale : lecture et routage des messages entrants depuis le routeur
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		typ := comm.FindVal(line, "type")

		switch typ {
		case "JOIN_REQUEST":
			handleJoinRequest(st, logger, line)

		case "JOIN_ACCEPT":
			handleJoinAccept(st, logger, line)

		case "MEMBER_LIST":
			handleMemberList(st, logger, line)

		case "RECONFIGURE_ADD":
			handleReconfigureAdd(st, logger, line)

		case "LEAVE_REQUEST":
			handleLeaveRequest(st, logger, line)

		case "MEMBER_REMOVED":
			handleMemberRemoved(st, logger, line)

		case "ELECTION":
			handleElection(st, logger, line)

		case "SHOW_NET_STATE":
			handleShowNetState(st, logger, line)

		case "PING":
			handlePing(st, logger, line)

		case "PONG":
			handlePong(st, logger, line)

		case "CTL_MSG":
			handleCtlMsg(st, logger, line, *ctlOut)
		}
	}
}

// retryJoin est utilisée par un site qui souhaite rejoindre le groupe.
// Tant que le site n'est pas accepté, il envoie périodiquement
// un JOIN_REQUEST au site indiqué par JoinTarget.
// Cela permet de tolérer le cas où la première demande est ignorée
// parce qu'une autre admission est déjà en cours.
func retryJoin(st *netstate.State, logger *utils.Logger) {
	for !st.Joined {
		if st.Passive {
			// un site passif ne peut pas rejoindre le groupe
			return
		}

		req := buildJoinRequest(st.Name, st.JoinTarget)
		logger.Info("network", "envoi JOIN_REQUEST vers "+st.JoinTarget)
		fmt.Fprintln(os.Stdout, req)

		time.Sleep(5 * time.Second)
	}
}

// heartbeatLoop envoie périodiquement un message PING à chaque membre connu.
// Si un membre ne répond plus, le failureDetector déclenchera sa suppression.
// Fréquence : toutes les 5 secondes.
func heartbeatLoop(st *netstate.State, logger *utils.Logger) {
	for {
		time.Sleep(5 * time.Second)

		if st.Passive || !st.Joined {
			continue
		}

		for member := range st.Members {
			if member == st.Name {
				continue
			}

			if _, ok := st.Topology[member]; !ok {
				continue
			}

			ping := buildPing(st.Name, member)
			logger.Info("heartbeat", "PING envoyé à "+member)
			fmt.Fprintln(os.Stdout, ping)
		}
	}
}

// failureDetector surveille périodiquement les membres connus du réseau.
//
// Principe :
//   - chaque site conserve la date du dernier PONG reçu dans LastSeen ;
//   - si un membre ne répond plus pendant plus de 30 secondes,
//     il est considéré comme suspect ;
//   - pour éviter que plusieurs sites diffusent simultanément MEMBER_REMOVED,
//     seul le prédécesseur logique du membre suspecté a le droit
//     de confirmer la panne et de diffuser la suppression.
//
// Exemple :
// Anneau : A -> D -> B -> C -> A
// Si D ne répond plus :
//   - A est le prédécesseur de D ;
//   - seul A diffuse MEMBER_REMOVED(D, pred=A, succ=B) ;
//   - les autres sites peuvent suspecter D, mais ne déclenchent pas la reconfiguration.
func failureDetector(st *netstate.State, logger *utils.Logger) {
	for {
		time.Sleep(3 * time.Second)

		if st.Passive || !st.Joined {
			continue
		}

		now := time.Now()

		for member := range st.Members {
			if member == st.Name {
				continue
			}

			links, ok := st.Topology[member]
			if !ok {
				continue
			}

			if links.Pred == "" || links.Succ == "" {
				continue
			}

			last, ok := st.LastSeen[member]
			if !ok {
				st.LastSeen[member] = now
				continue
			}

			if now.Sub(last) <= 30*time.Second {
				continue
			}

			logger.Info("failure", "participant suspecté mort : "+member)

			if links.Pred != st.Name {
				logger.Info(
					"failure",
					"panne détectée sur "+member+
						" mais seul le prédécesseur "+links.Pred+
						" peut supprimer",
				)
				continue
			}

			removed := buildMemberRemoved(member, links.Pred, links.Succ)
			logger.Info("failure", "diffusion MEMBER_REMOVED : "+removed)

			fmt.Fprintln(os.Stdout, removed)

			handleMemberRemoved(st, logger, removed)
			delete(st.LastSeen, member)

			// On traite une seule panne par cycle pour éviter les suppressions en cascade.
			break
		}
	}
}
