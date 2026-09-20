/**
 * websocket.js
 * Gestion de la connexion WebSocket entre le navigateur et APP.
 *
 * Responsabilités :
 *  - établissement et fermeture de la connexion WebSocket ;
 *  - réception des messages entrants et routage vers handleMessage ;
 *  - affichage des événements de connexion dans les logs.
 */

// connexion WebSocket courante (null si non connectée)
var ws = null;

/**
 * connectWebSocket établit une connexion WebSocket vers APP.
 * L'hôte et le port sont lus depuis les champs du formulaire.
 * Si une connexion est déjà active, la fonction ne fait rien.
 */
function connectWebSocket() {
    if (ws) return false;

    var host = document.getElementById("host").value;
    var port = document.getElementById("port").value;

    addToLog("Connexion à ws://" + host + ":" + port + "/ws");

    ws = new WebSocket("ws://" + host + ":" + port + "/ws");

    // connexion établie
    ws.onopen = function() {
        addToLog("WebSocket ouverte");
        document.getElementById("currentSite").textContent = "port " + port;
    };

    // connexion fermée
    ws.onclose = function() {
        addToLog("WebSocket fermée");
        ws = null;
    };

    // message reçu depuis APP → routage vers handleMessage
    ws.onmessage = function(evt) {
        //console.log("RAW =", evt.data);
        //addToLog("RAW = " + evt.data);
        handleMessage(evt.data);
    };

    // erreur de connexion
    ws.onerror = function() {
        addToLog("Erreur WebSocket");
    };

    return false;
}

/**
 * closeWebSocket ferme la connexion WebSocket active.
 */
function closeWebSocket() {
    if (ws) ws.close();
    return false;
}