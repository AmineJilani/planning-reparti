/*
handlers.go : Traitement des messages reçus depuis le web et depuis CTL.
Responsabilités :
  - transformation des commandes web en messages structurés vers CTL
  - routage des messages CTL vers l'interface web selon leur type
  - gestion de la commande d'ajout dynamique de site.
*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"planning-reparti/internal/comm"
)

// handleWebMessage traite un message reçu depuis l'interface web.
// Si le message est une commande JOIN_SITE, il lance le script d'ajout de site.
// Sinon, il transforme la commande en message structuré et l'envoie vers CTL.
func handleWebMessage(msg string) {
	uiLog("WEB → " + msg)

	// cas particulier : ajout dynamique d'un site
	if comm.FindVal(msg, "type") == "JOIN_SITE" {
		handleJoinSiteCommand(msg)
		return
	}

	// transformation de la commande utilisateur en message structuré
	structured := handleUserCommand(msg)
	if structured == "" {
		uiLog("commande invalide")
		return
	}

	uiLog("SEND TO CTL → " + structured)

	_, err := fmt.Fprintln(os.Stdout, structured)
	if err != nil {
		uiLog("ERREUR ÉCRITURE STDOUT → " + err.Error())
		return
	}

}

// handleCtlMessage traite un message reçu depuis CTL via stdin.
// Il filtre les messages internes et route les messages utiles
// vers l'interface web via WebSocket selon leur type.
func handleCtlMessage(msg string) {
	uiLog("CTL ← " + msg)

	// filtrage des messages internes non destinés à l'interface web
	if isInternal(msg) {
		return
	}

	typ := comm.FindVal(msg, "type")

	switch typ {
	case "VIEW_CHANGE":
		// changement de vue du groupe → mise à jour de l'interface
		logger.Info("ws", "envoi VIEW_CHANGE au navigateur")
		wsPush("VIEW_CHANGE|" + msg)

	case "STATUS":
		// état du site (ex: LEFT après départ)
		wsPush("STATUS|" + msg)

	case "LOCAL_STATE":
		// état local CTL demandé par l'utilisateur
		wsPush("STATE|" + msg)

	case "PLANNING":
		// mise à jour du planning local
		wsPush("PLANNING|" + msg)

	case "NET_STATE":
		// état local NET demandé par l'utilisateur
		wsPush("NET_STATE|" + msg)

	case "INFO":
		// message d'information générale
		wsPush("INFO|" + msg)

	case "STATE", "SNAPSHOT_START":
		// événement lié à un snapshot réparti
		wsPush("SNAPSHOT|" + msg)
	}
}

// handleJoinSiteCommand traite la commande d'ajout dynamique d'un site.
// Elle lance le script add_site.sh avec les paramètres fournis
// par l'interface web.
func handleJoinSiteCommand(msg string) {
	site := comm.FindVal(msg, "site")
	port := comm.FindVal(msg, "port")
	target := comm.FindVal(msg, "target")

	if site == "" || port == "" {
		uiLog("Erreur : site ou port manquant")
		return
	}

	// site cible par défaut : A
	if target == "" {
		target = "A"
	}

	uiLog("Ajout du site " + site + " sur le port " + port + "...")

	cmd := exec.Command("./scripts/add_site.sh", site, target, port)
	err := cmd.Start()
	if err != nil {
		uiLog("Erreur ajout site " + site + " : " + err.Error())
		return
	}

	uiLog("Site " + site + " lancé")
	uiLog("Interface : http://localhost:" + port)
}

// isInternal filtre les messages internes qui ne doivent pas
// être transmis à l'interface web.
func isInternal(msg string) bool {
	return strings.Contains(msg, "WebSocket ouverte") ||
		strings.Contains(msg, "CTL ← CTL") ||
		strings.HasPrefix(msg, "[app-")
}
