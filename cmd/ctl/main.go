/*
CONTROLEUR REPARTI (CTL) :
Point d'entrée du système.
Responsabilités :
- initialisation du site
- création de l'état local
- lecture des événements entrants
- routage vers les différents handlers

Le traitement métier est délégué aux autres fichiers afin de conserver un main simple et lisible.
*/

package main

import (
	"bufio" // lire l'entrée standard ligne par ligne
	"flag"  // donner un nom au site avec l'option -n dans la ligne de commande

	"os" //utiliser les sorties, entrées standard et erreur standard
	"strings"

	//Appel aux autres packages
	"planning-reparti/internal/state_site"
	"planning-reparti/internal/utils"
)

func main() {
	name := flag.String("n", "A", "nom du site")
	nSites := flag.Int("sites", 3, "nombre total de sites")
	netOut := flag.String("net-out", "", "FIFO CTL vers NET")

	flag.Parse()

	logger := utils.NewLogger("ctl-" + *name)
	logger.Info("main ctl", "démarrage ctl")

	sites := []string{}
	for i := 0; i < *nSites; i++ {
		sites = append(sites, string(rune('A'+i)))
	}

	st := state_site.NewSiteState(*name, sites)
	if *netOut != "" {
		st.NetworkOut = *netOut
	}

	events := make(chan string)
	go readInput(events, logger)
	for {
		select {
		case line, ok := <-events:
			if !ok {
				logger.Warn("main", "fin de l'entrée standard")
				return
			}

			logger.Info("main", "événement reçu : "+line)

			handleEvent(st, logger, line)
		}
	}
}

func readInput(events chan string, logger *utils.Logger) {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		events <- line
	}

	if err := scanner.Err(); err != nil {
		logger.Error("stdin", "erreur scanner : "+err.Error())
	}

	close(events)
}
