/*
Server.go : Serveur HTTP de l'interface web.
Responsabilités :
  - service des fichiers statiques (HTML, CSS, JS)
  - exposition du endpoint WebSocket /ws
  - lecture des messages stdin venant de CTL.
*/

package main

import (
	"bufio"
	"net/http"
	"os"
	"strings"
)

// startWebServer démarre le serveur HTTP et expose les routes nécessaires.
// Les fichiers statiques de l'interface web sont servis depuis cmd/app/.
// Le endpoint /ws gère la connexion WebSocket avec le navigateur.
func startWebServer(addr, port string, webChan chan<- string) {
	// service de la page principale
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "cmd/app/client.html")
	})

	// service des fichiers statiques JavaScript et CSS
	http.Handle("/DOM.js", http.FileServer(http.Dir("cmd/app")))
	http.Handle("/messages.js", http.FileServer(http.Dir("cmd/app")))
	http.Handle("/websocket.js", http.FileServer(http.Dir("cmd/app")))
	http.Handle("/actions.js", http.FileServer(http.Dir("cmd/app")))
	http.Handle("/app.js", http.FileServer(http.Dir("cmd/app")))
	http.Handle("/style.css", http.FileServer(http.Dir("cmd/app")))

	// endpoint WebSocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		doWebSocket(w, r, webChan)
	})

	logger.Info("web", "serveur web lancé sur http://"+addr+":"+port)

	err := http.ListenAndServe(addr+":"+port, nil)
	if err != nil {
		logger.Error("web", err.Error())
	}
}

// readStdin lit en continu les messages entrants depuis stdin.
// Ces messages proviennent de CTL et sont transmis au channel stdinChan
// pour traitement par la boucle principale.
func readStdin(stdinChan chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			stdinChan <- line
		}
	}
}
