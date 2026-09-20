/*
ain.go --> Point d'entrée de l'application APP.

APP joue le rôle d'intermédiaire entre l'interface web et le contrôleur CTL.
Responsabilités :
  - lecture des messages entrants depuis CTL via stdin
  - réception des commandes utilisateur depuis l'interface web via WebSocket
  - transformation des commandes en messages structurés vers CTL via stdout
  - transmission des événements distribués vers l'interface web.

Flux de communication :
  Navigateur → WebSocket → APP → CTL (stdout)
  CTL → APP (stdin) → WebSocket → Navigateur
*/

package main

import (
	"flag"

	"planning-reparti/internal/utils"
)

// variables globales partagées entre les fichiers du package
var (
	appName string
	logger  *utils.Logger
)

func main() {
	name := flag.String("n", "A", "nom du site")
	addr := flag.String("addr", "localhost", "adresse du serveur web")
	port := flag.String("port", "4444", "port du serveur web")
	flag.Parse()

	appName = *name
	logger = utils.NewLogger("app-" + appName)

	logger.Info("main", "démarrage app")

	// channel pour les messages venant de l'interface web
	webChan := make(chan string)
	// channel pour les messages venant de CTL via stdin
	stdinChan := make(chan string)

	// goroutine de lecture de stdin (messages CTL → APP)
	go readStdin(stdinChan)

	// goroutine du serveur web et WebSocket
	go startWebServer(*addr, *port, webChan)

	// boucle principale : routage des messages selon leur origine
	for {
		select {
		// message venant de l'interface web → transformation et envoi vers CTL
		case msg := <-webChan:
			handleWebMessage(msg)

		// message venant de CTL → transmission vers l'interface web
		case msg := <-stdinChan:
			handleCtlMessage(msg)
		}
	}
}
