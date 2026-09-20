/*
network.go
Outils de communication réseau du contrôleur CTL.

Responsabilités :
  - envoi des messages vers NET via FIFO ou stdout ;
  - envoi des messages vers APP via stdout ;
  - identification unique des messages pour la détection des doublons.

Architecture de communication :
  CTL → NET → ROUTER → NET → CTL
  CTL → APP (stdout)
*/

package main

import (
	"fmt"
	"os"
	"planning-reparti/internal/comm"
	"planning-reparti/internal/state_site"
	"planning-reparti/internal/utils"
)

// sendNetwork envoie un message vers NET.
// Si NetworkOut est configuré (FIFO CTL → NET), le message y est écrit.
// Sinon, le message est écrit sur stdout (comportement de secours).
func sendNetwork(st *state_site.SiteState, logger *utils.Logger, msg string) {
	if st.NetworkOut == "" {
		fmt.Println(msg)
		return
	}

	file, err := os.OpenFile(st.NetworkOut, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		logger.Error("network", "erreur écriture vers NET : "+err.Error())
		return
	}
	defer file.Close()

	fmt.Fprintln(file, msg)
	logger.Info("network", "message envoyé à NET : "+msg)
}

// sendApp envoie un message vers APP via stdout.
// APP lit ce flux et transmet les informations à l'interface web.
func sendApp(logger *utils.Logger, msg string) {
	fmt.Fprintln(os.Stdout, msg)
	os.Stdout.Sync()
	logger.Info("app", "message envoyé à APP : "+msg)
}

// messageID construit un identifiant unique pour un message reçu.
// Cet identifiant est utilisé dans SeenMessages pour détecter
// et ignorer les messages déjà traités, afin d'éviter les boucles
// dans l'anneau de diffusion.
//
// Format selon le type de message :
//   - MARKER : type-site-snap
//   - autres : type-site-hlg-res
func messageID(line string) string {
	typ := comm.FindVal(line, "type")
	site := comm.FindVal(line, "site")
	hlg := comm.FindVal(line, "hlg")
	snap := comm.FindVal(line, "snap")

	switch typ {
	case "MARKER":
		return typ + "-" + site + "-" + snap
	default:
		res := comm.FindVal(line, "res")
		return typ + "-" + site + "-" + hlg + "-" + res
	}
}
