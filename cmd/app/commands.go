/*
Commands.go : Transformation des commandes utilisateur en messages structurés.
Responsabilités :
  - parsing des commandes texte saisies dans l'interface web
  - construction des messages structurés correspondants
  - formatage des messages pour l'affichage dans les logs et l'interface web.
*/

package main

import (
	"fmt"
	"os"
	"strings"

	"planning-reparti/internal/comm"
)

// handleUserCommand transforme une commande texte en message structuré.
// Si la ligne est déjà au format structuré (commence par /=), elle est retournée telle quelle.
// Retourne une chaîne vide si la commande est invalide ou inconnue.
func handleUserCommand(line string) string {
	line = strings.TrimSpace(line)

	// message déjà structuré → transmis tel quel
	if strings.HasPrefix(line, "/=") {
		return line
	}

	parts := strings.Split(line, " ")
	if len(parts) == 0 {
		return ""
	}

	switch parts[0] {

	case "reserve":
		// format attendu : reserve <id> <salle> <date> <début> <fin> <utilisateur>
		if len(parts) != 7 {
			return ""
		}
		return comm.Build(map[string]string{
			"type":  "RESERVE_LOCAL",
			"id":    parts[1],
			"room":  parts[2],
			"date":  parts[3],
			"start": parts[4],
			"end":   parts[5],
			"user":  parts[6],
		})

	case "snapshot":
		// format attendu : snapshot <id>
		if len(parts) != 2 {
			return ""
		}
		return comm.Build(map[string]string{
			"type": "SNAPSHOT_LOCAL",
			"snap": parts[1],
		})
	}

	return ""
}

// format enrichit un message brut d'un préfixe lisible
// selon son type pour l'affichage dans les logs et l'interface web.
func format(msg string) string {
	switch {
	case strings.Contains(msg, "SNAPSHOT_START"):
		return "SNAPSHOT START → " + msg
	case strings.Contains(msg, "SNAPSHOT_LOCAL"):
		return "SNAPSHOT → " + msg
	case strings.Contains(msg, "STATE"):
		return "STATE → " + msg
	case strings.Contains(msg, "PLANNING"):
		return "PLANNING → " + msg
	case strings.Contains(msg, "REQUEST"):
		return "REQUEST → " + msg
	case strings.Contains(msg, "ACK"):
		return "ACK → " + msg
	case strings.Contains(msg, "UPDATE"):
		return "UPDATE → " + msg
	case strings.Contains(msg, "RELEASE"):
		return "RELEASE → " + msg
	case strings.Contains(msg, "RESERVE_LOCAL"):
		return "RESERVATION → " + msg
	default:
		return msg
	}
}

// uiLog affiche un message dans le terminal de APP
// et le pousse également vers l'interface web via WebSocket.
func uiLog(msg string) {
	formatted := format(msg)
	// affichage terminal
	fmt.Fprintln(os.Stderr, "[app-"+appName+"] "+formatted)
	// transmission vers l'interface web
	wsPush(formatted)
}
