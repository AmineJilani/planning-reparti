/**
 * messages.js
 * Parsing et routage des messages reçus depuis APP via WebSocket.
 *
 * Responsabilités :
 *  - parsing des messages au format clé/valeur du protocole ;
 *  - routage vers les fonctions d'affichage selon le préfixe du message.
 *
 * Préfixes supportés :
 *  PLANNING|     → mise à jour du tableau de réservations
 *  SNAPSHOT|     → affichage des états snapshot
 *  STATE|        → affichage de l'état local CTL
 *  STATUS|       → mise à jour du statut du site
 *  NET_STATE|    → affichage de l'état local NET et topologie
 *  INFO|         → message d'information générale
 */

/**
 * parseProtocolMessage parse un message au format clé/valeur du protocole.
 * Format : /=clé=valeur/=clé=valeur...
 * Retourne un objet JavaScript avec les paires clé/valeur extraites.
 */
function parseProtocolMessage(msg) {
    var result = {};
    var parts = msg.split("/=");

    for (var i = 0; i < parts.length; i++) {
        if (!parts[i]) continue;

        var index = parts[i].indexOf("=");
        if (index === -1) continue;

        var key = parts[i].substring(0, index);
        var value = parts[i].substring(index + 1);
        result[key] = value;
    }

    return result;
}

/**
 * handleMessage reçoit un message brut depuis WebSocket
 * et le route vers la fonction d'affichage appropriée selon son préfixe.
 */
function handleMessage(raw) {
    addToLog("Réception : " + raw);

    // mise à jour du tableau de réservations
    if (raw.startsWith("PLANNING|")) {
        var data = parseProtocolMessage(raw.substring("PLANNING|".length));
        updatePlanning(data);
        return;
    }

    // affichage des états snapshot collectés
    if (raw.startsWith("SNAPSHOT|")) {
        updateSnapshot(raw.substring("SNAPSHOT|".length));
        return;
    }

    // affichage de l'état local CTL
    if (raw.startsWith("STATE|")) {
        document.getElementById("stateView").textContent =
            raw.substring("STATE|".length);
        return;
    }

    // mise à jour du statut du site (ex : site quitté)
    if (raw.startsWith("STATUS|")) {
        var data = parseProtocolMessage(raw.substring("STATUS|".length));
        var status = document.getElementById("siteStatus");

        if (data.state === "LEFT") {
            siteActive = false;
            status.textContent = "Quitté";
            status.className = "status left";
            document.getElementById("topology").innerHTML =
                "<strong>Site quitté du réseau</strong><br>" +
                "Ce site ne participe plus à l'anneau.";
            document.getElementById("stateView").textContent = "Site inactif";
            document.getElementById("netStateView").textContent = "Site inactif";
            disableActions();
        }
        return;
    }

    // affichage de l'état local NET et mise à jour de la topologie
    if (raw.startsWith("NET_STATE|")) {
        var protocol = raw.substring("NET_STATE|".length);
        var data = parseProtocolMessage(protocol);

        document.getElementById("netStateView").textContent = protocol;

        var site    = data.site    || "?";
        var pred    = data.pred    || "?";
        var succ    = data.succ    || "?";
        var members = data.members || "";
        var view    = data.view    || "?";

        document.getElementById("topology").innerHTML =
            "<strong>Anneau local :</strong><br>" +
            "<span class='node'>" + pred + "</span>" +
            " → " +
            "<span class='node current'>" + site + "</span>" +
            " → " +
            "<span class='node'>" + succ + "</span>" +
            "<br><br>" +
            "<strong>Membres :</strong> " + members +
            "<br>" +
            "<strong>Vue :</strong> " + view;
        return;
    }

    // message d'information générale
    if (raw.startsWith("INFO|")) {
        var data = parseProtocolMessage(raw.substring("INFO|".length));
        var status = document.getElementById("siteStatus");

        addToLog(data.msg);

        if (data.msg && data.msg.includes("Site quitté")) {
            siteActive = false;
            status.textContent = "Quitté";
            status.className = "status left";
            disableActions();
        }
        return;
    }

    // fallback : tentative de parsing direct si message au format protocole
    var index = raw.indexOf("/=");
    if (index !== -1) {
        var data = parseProtocolMessage(raw.substring(index));
        updatePlanning(data);
    }
}