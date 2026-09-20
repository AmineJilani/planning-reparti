/*
handlers.go
Traitement de tous les messages reçus par un participant NET.

Chaque handler correspond à un type de message du protocole réparti.
Les handlers modifient l'état local du site et émettent
les messages nécessaires vers le routeur ou vers CTL.

Messages traités :
  - JOIN_REQUEST    : demande d'admission d'un nouveau participant
  - JOIN_ACCEPT     : acceptation et intégration dans l'anneau
  - MEMBER_LIST     : mise à jour de la vue du groupe
  - RECONFIGURE_ADD : reconfiguration de l'anneau après admission
  - LEAVE_REQUEST   : départ volontaire d'un participant
  - MEMBER_REMOVED  : suppression d'un membre du groupe
  - ELECTION        : résolution des conflits d'admission simultanés
  - SHOW_NET_STATE  : affichage de l'état local NET
  - PING / PONG     : surveillance des participants pour la détection de panne
*/

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"planning-reparti/internal/comm"
	netstate "planning-reparti/internal/net"
	"planning-reparti/internal/utils"
)

// handleJoinRequest traite une demande d'admission dans le groupe.
// Si le site courant est le destinataire, il lance une élection
// pour garantir qu'aucune autre admission n'est en cours simultanément.
func handleJoinRequest(st *netstate.State, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	to := comm.FindVal(line, "to")

	if to != st.Name {
		fmt.Fprintln(os.Stdout, line)
		return
	}

	if st.Passive {
		return
	}

	logger.Info("network", "JOIN_REQUEST reçu de "+src)

	if st.Members[src] {
		logger.Info("network", src+" est déjà membre")
		return
	}

	if st.Admission.Running {
		logger.Info("admission", "admission déjà en cours, demande ignorée temporairement : "+src)
		return
	}

	// initialisation de l'état local de l'admission
	st.Admission.Running = true
	st.Admission.Host = st.Name
	st.Admission.Joining = src
	st.Admission.ID = "ADM-" + st.Name + "-" + src
	st.Admission.Winner = st.Name

	logger.Info("admission", "élection lancée pour admettre "+src)

	election := buildElection(
		st.Name,
		st.Successor,
		st.Admission.ID,
		src,
		st.Name,
	)

	logger.Info("admission", "envoi ELECTION vers "+st.Successor+" : "+election)
	fmt.Fprintln(os.Stdout, election)
}

// handleJoinAccept traite l'acceptation d'un nouveau participant.
// Le site met à jour sa position dans l'anneau (prédécesseur, successeur)
// et devient officiellement membre du groupe.
func handleJoinAccept(st *netstate.State, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	to := comm.FindVal(line, "to")
	succ := comm.FindVal(line, "succ")
	pred := comm.FindVal(line, "pred")

	if to != st.Name {
		fmt.Fprintln(os.Stdout, line)
		return
	}

	if st.Joined {
		logger.Info("network", "JOIN_ACCEPT ignoré : site déjà rejoint")
		return
	}

	st.Members[src] = true
	st.LastSeen[src] = time.Now()
	st.Successor = succ
	st.Predecessor = pred
	st.Joined = true

	logger.Info("network", "JOIN_ACCEPT reçu de "+src)
	logger.Info("network", "réseau rejoint")
	logger.Info("network", "mon prédécesseur = "+st.Predecessor)
	logger.Info("network", "mon successeur = "+st.Successor)
	logger.Info("network", "membres = "+st.MembersString())
}

// handleMemberList traite la réception d'une nouvelle vue du groupe.
// Elle met à jour la liste locale des membres et le numéro de vue.
// Un message VIEW_CHANGE est ensuite envoyé à CTL pour synchroniser
// la liste des membres côté logique répartie.
func handleMemberList(st *netstate.State, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	id := comm.FindVal(line, "id")
	members := comm.FindVal(line, "members")
	viewStr := comm.FindVal(line, "view")
	view, err := strconv.Atoi(viewStr)
	if err != nil {
		view = 0
	}

	// clé unique pour éviter de traiter plusieurs fois la même MEMBER_LIST
	key := src + "-" + id
	if st.SeenBroadcasts[key] {
		logger.Info("network", "MEMBER_LIST déjà traitée : "+key)
		return
	}
	st.SeenBroadcasts[key] = true

	// on ignore les vues plus anciennes que la vue courante
	if view < st.MembershipVerion {
		logger.Info("network", "MEMBER_LIST ignorée : vue ancienne")
		return
	}

	// mise à jour de la liste locale des membres
	newMembers := make(map[string]bool)
	for _, m := range strings.Split(members, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			newMembers[m] = true
			if !st.Members[m] {
				st.LastSeen[m] = time.Now()
			}
		}
	}

	st.Members = newMembers

	if view > st.MembershipVerion {
		st.MembershipVerion = view
	}

	logger.Info("network", "MEMBER_LIST reçue")
	logger.Info("network", "membres = "+st.MembersString())
	logger.Info("network", "vue membership = "+fmt.Sprintf("%d", st.MembershipVerion))

	// notification de changement de vue vers CTL
	viewMsg := buildViewChange(st)
	sendToCtl(st.CtlOut, viewMsg)
}

// handleLeaveRequest traite la demande volontaire de départ du groupe.
// Le site courant diffuse un message MEMBER_REMOVED contenant
// son prédécesseur et son successeur pour permettre la reconfiguration de l'anneau.
func handleLeaveRequest(st *netstate.State, logger *utils.Logger, line string) {
	site := comm.FindVal(line, "site")
	if site != st.Name {
		return
	}

	msg := buildMemberRemoved(st.Name, st.Predecessor, st.Successor)
	fmt.Fprintln(os.Stdout, msg)
}

// handleMemberRemoved applique la suppression d'un membre du groupe.
// Chaque site met à jour sa vue locale, reconfigure ses voisins dans l'anneau
// et notifie CTL du changement de vue.
// Ce handler est appelé aussi bien lors d'un départ volontaire
// que lors d'une panne détectée par le mécanisme de heartbeat.
func handleMemberRemoved(st *netstate.State, logger *utils.Logger, line string) {
	removed := comm.FindVal(line, "site")
	pred := comm.FindVal(line, "pred")
	succ := comm.FindVal(line, "succ")

	// clé unique pour éviter de traiter plusieurs fois le même MEMBER_REMOVED
	key := "MEMBER_REMOVED-" + removed + "-" + pred + "-" + succ
	if st.SeenBroadcasts[key] {
		logger.Info("leave", "MEMBER_REMOVED déjà traité : "+key)
		return
	}
	st.SeenBroadcasts[key] = true

	// suppression du membre dans la vue locale
	delete(st.Members, removed)
	delete(st.Topology, removed)
	st.MembershipVerion++

	// reconfiguration de l'anneau selon la position du site courant
	if st.Name == pred {
		st.Successor = succ
		logger.Info("leave", "mon successeur devient "+succ)
	}
	if st.Name == succ {
		st.Predecessor = pred
		logger.Info("leave", "mon prédécesseur devient "+pred)
	}

	// si le site courant est celui qui quitte, il devient passif
	if st.Name == removed {
		st.Passive = true
		st.Successor = ""
		st.Predecessor = ""
		logger.Info("leave", "je quitte le réseau")
	}

	// mise à jour de la table de topologie locale
	if pred != "" {
		link := st.Topology[pred]
		link.Succ = succ
		st.Topology[pred] = link
	}
	if succ != "" {
		link := st.Topology[succ]
		link.Pred = pred
		st.Topology[succ] = link
	}

	// notification CTL uniquement si le site courant reste dans le groupe
	if st.Name != removed {
		viewMsg := buildViewChange(st)
		sendToCtl(st.CtlOut, viewMsg)
		netState := buildNetStateToCtl(st)
		sendToCtl(st.CtlOut, netState)
	}

	logger.Info("leave", "vue membership = "+fmt.Sprintf("%d", st.MembershipVerion))
	logger.Info("leave", "anneau : pred="+st.Predecessor+" succ="+st.Successor)
}

// handleElection traite un message d'élection circulant dans l'anneau.
// L'objectif est de garantir qu'une seule admission est validée à la fois.
// La règle de priorité est simple : l'identifiant le plus petit gagne.
// Lorsque l'élection revient à son initiateur, celui-ci valide ou annule l'admission.
func handleElection(st *netstate.State, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	to := comm.FindVal(line, "to")
	id := comm.FindVal(line, "id")
	joining := comm.FindVal(line, "joining")
	winner := comm.FindVal(line, "winner")

	if st.Passive {
		return
	}
	if to != st.Name {
		return
	}
	if winner == "" {
		winner = src
	}

	logger.Info("election", "ELECTION reçue id="+id+" joining="+joining+" winner="+winner)

	time.Sleep(1 * time.Second)

	// détection d'un conflit entre deux admissions simultanées
	if st.Admission.Running && st.Admission.ID != id {
		if st.Admission.Host < winner {
			winner = st.Admission.Host
		}
		// si l'admission locale n'est pas gagnante, elle est annulée
		if winner != st.Admission.Host {
			logger.Info("election", "élection perdue : gagnant="+winner)
			st.Admission.Running = false
			st.Admission.Winner = winner
		}
	}

	// l'élection revient à son initiateur → tour de l'anneau terminé
	if src == st.Name && id == st.Admission.ID {
		logger.Info("election", "ELECTION terminée pour "+joining+" | gagnant="+winner)

		if winner == st.Name {
			// le site courant est gagnant → il valide l'admission
			logger.Info("election", "je suis gagnant → admission validée")

			oldSucc := st.Successor
			accept := buildJoinAccept(st.Name, joining, oldSucc)
			fmt.Fprintln(os.Stdout, accept)

			st.Members[joining] = true

			// reconfiguration de l'anneau : insertion du nouveau site
			reconf := buildReconfigureAdd(st.Name, joining, oldSucc)
			logger.Info("reconfigure", "insertion dans l'anneau : "+reconf)
			fmt.Fprintln(os.Stdout, reconf)

			st.Successor = joining
			st.MembershipVerion++

			// diffusion de la nouvelle liste des membres
			memberList := buildMemberList(st)
			fmt.Fprintln(os.Stdout, memberList)

			// notification de changement de vue vers CTL
			viewChange := buildViewChange(st)
			sendToCtl(st.CtlOut, viewChange)

			st.Admission.Running = false
			st.Admission.Winner = ""
		} else {
			// un autre site est gagnant → admission locale annulée
			logger.Info("election", "admission annulée, gagnant="+winner)
			st.Admission.Running = false
			st.Admission.Winner = winner
		}
		return
	}

	// l'élection n'est pas revenue à l'initiateur → transmission au successeur
	next := buildElection(src, st.Successor, id, joining, winner)
	logger.Info("election", "transmission ELECTION vers "+st.Successor+" | gagnant actuel="+winner)
	fmt.Fprintln(os.Stdout, next)
}

// handleReconfigureAdd applique l'insertion d'un nouveau participant dans l'anneau.
// Chaque site met à jour sa position locale (prédécesseur, successeur)
// et la table de topologie distribuée.
func handleReconfigureAdd(st *netstate.State, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	newNode := comm.FindVal(line, "new")
	oldSucc := comm.FindVal(line, "oldsucc")

	// clé unique pour éviter plusieurs reconfigurations du même événement
	key := "RECONFIGURE_ADD-" + src + "-" + newNode + "-" + oldSucc
	if st.SeenBroadcasts[key] {
		logger.Info("reconfigure", "RECONFIGURE_ADD déjà traité : "+key)
		return
	}

	if !st.Joined && st.Name != newNode {
		logger.Info("reconfigure", "ignoré : site pas encore membre")
		return
	}

	st.SeenBroadcasts[key] = true
	st.Members[newNode] = true

	// mise à jour de la table de topologie
	srcLink := st.Topology[src]
	oldSuccLink := st.Topology[oldSucc]

	st.Topology[src] = netstate.NodeLinks{Pred: srcLink.Pred, Succ: newNode}
	st.Topology[newNode] = netstate.NodeLinks{Pred: src, Succ: oldSucc}
	st.Topology[oldSucc] = netstate.NodeLinks{Pred: newNode, Succ: oldSuccLink.Succ}

	// mise à jour de la position locale selon le rôle du site courant
	if st.Name == src {
		st.Successor = newNode
		logger.Info("reconfigure", "mon successeur devient "+newNode)
	}
	if st.Name == newNode {
		st.Predecessor = src
		st.Successor = oldSucc
		st.Joined = true
		logger.Info("reconfigure", "mon prédécesseur devient "+src)
		logger.Info("reconfigure", "mon successeur devient "+oldSucc)
	}
	if st.Name == oldSucc {
		st.Predecessor = newNode
		logger.Info("reconfigure", "mon prédécesseur devient "+newNode)
	}

	logger.Info("reconfigure", "insertion demandée : "+src+" -> "+newNode+" -> "+oldSucc)
	logger.Info("reconfigure", "état anneau local : pred="+st.Predecessor+" succ="+st.Successor)
}

// handleShowNetState affiche l'état local du site NET dans les logs.
// Ce handler est déclenché par une commande SHOW_NET_STATE
// envoyée depuis CTL ou l'interface web.
func handleShowNetState(st *netstate.State, logger *utils.Logger, line string) {
	to := comm.FindVal(line, "to")
	if to != st.Name && to != "ALL" {
		return
	}

	logger.Info("state", "nom="+st.Name)
	logger.Info("state", "pred="+st.Predecessor+" succ="+st.Successor)
	logger.Info("state", "membres="+st.MembersString())
	logger.Info("state", "view="+fmt.Sprintf("%d", st.MembershipVerion))
}

// handlePing traite un message PING reçu d'un autre participant.
// Si le message est destiné au site courant, il répond immédiatement par un PONG.
// Sinon le message est retransmis vers le routeur.
func handlePing(st *netstate.State, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	to := comm.FindVal(line, "to")

	if to != st.Name {
		fmt.Fprintln(os.Stdout, line)
		return
	}

	logger.Info("heartbeat", "PING reçu de "+src)
	pong := buildPong(st.Name, src)
	logger.Info("heartbeat", "PONG envoyé à "+src)
	fmt.Fprintln(os.Stdout, pong)
}

// handlePong traite un message PONG reçu en réponse à un PING.
// Il met à jour la date du dernier contact avec le site émetteur dans LastSeen.
// Si le message n'est pas destiné au site courant, il est retransmis.
func handlePong(st *netstate.State, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	to := comm.FindVal(line, "to")

	if to != st.Name {
		fmt.Fprintln(os.Stdout, line)
		return
	}

	st.LastSeen[src] = time.Now()
	logger.Info("heartbeat", "PONG reçu de "+src)
}
