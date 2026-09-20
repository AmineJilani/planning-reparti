/* Websocket.go
Gestion de la connexion WebSocket entre APP et le navigateur.

Responsabilités :
  - upgrade de la connexion HTTP vers WebSocket ;
  - envoi des messages vers le navigateur (wsPush) ;
  - lecture des messages entrants depuis le navigateur ;
  - gestion de la connexion et déconnexion.
*/

package main

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	ws      *websocket.Conn
	wsMutex sync.Mutex
)

var upgrader = websocket.Upgrader{
	// autorisation de toutes les origines pour le développement
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsPush envoie un message texte au navigateur via WebSocket.
// Si aucun navigateur n'est connecté, le message est ignoré.
func wsPush(msg string) {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	if ws == nil {
		logger.Warn("ws", "aucun navigateur connecté, message non envoyé : "+msg)
		return
	}

	logger.Info("ws", "push navigateur : "+msg)
	_ = ws.WriteMessage(websocket.TextMessage, []byte(msg))
}

// doWebSocket gère le cycle de vie d'une connexion WebSocket.
// Il lit les messages entrants depuis le navigateur et les transmet
// au channel webChan pour traitement par la boucle principale.
func doWebSocket(w http.ResponseWriter, r *http.Request, webChan chan<- string) {
	cnx, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("ws", err.Error())
		return
	}

	// enregistrement de la connexion active
	wsMutex.Lock()
	ws = cnx
	wsMutex.Unlock()

	uiLog("WebSocket ouverte")

	for {
		_, msg, err := cnx.ReadMessage()
		if err != nil {
			uiLog("WebSocket fermée")
			wsMutex.Lock()
			ws = nil
			wsMutex.Unlock()
			return
		}
		// transmission du message vers la boucle principale
		webChan <- string(msg)
	}
}
