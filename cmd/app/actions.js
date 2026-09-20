/**
 * actions.js
 * Fonctions déclenchées par les interactions utilisateur dans l'interface web.
 *
 * Chaque fonction construit le message approprié et l'envoie
 * vers APP via WebSocket. Les actions sont bloquées si le site
 * est inactif ou si la WebSocket n'est pas connectée.
 *
 * Actions disponibles :
 *  - sendReservation   : envoie une demande de réservation
 *  - sendSnapshot      : déclenche un snapshot réparti
 *  - sendShowState     : demande l'état local CTL
 *  - sendShowNetState  : demande l'état local NET
 *  - sendAddSite       : ajoute dynamiquement un nouveau site
 *  - sendLeaveRequest  : demande le départ volontaire du site
 */

/**
 * sendReservation construit et envoie une commande de réservation
 * à partir des champs du formulaire de l'interface web.
 */
function sendReservation() {
    if (!siteActive) {
        addToLog("Site quitté : action impossible");
        return false;
    }
    if (!ws) {
        addToLog("WebSocket non connectée");
        return false;
    }

    var msg =
        "reserve "                                      +
        document.getElementById("id").value    + " "   +
        document.getElementById("room").value  + " "   +
        document.getElementById("date").value  + " "   +
        document.getElementById("start").value + " "   +
        document.getElementById("end").value   + " "   +
        document.getElementById("user").value;

    addToLog("Envoi : " + msg);
    ws.send(msg);
    return false;
}

/**
 * sendSnapshot déclenche un snapshot réparti.
 * Les états snapshot précédents sont réinitialisés avant l'envoi.
 */
function sendSnapshot() {
    if (!siteActive) {
        addToLog("Site quitté : action impossible");
        return false;
    }
    if (!ws) {
        addToLog("WebSocket non connectée");
        return false;
    }

    // réinitialisation des états snapshot affichés
    snapshotStates = {};
    renderSnapshots();

    var msg = "snapshot " + document.getElementById("snap").value;
    addToLog("Envoi : " + msg);
    ws.send(msg);
    return false;
}

/**
 * sendShowState demande l'affichage de l'état local CTL
 * (horloge, vecteur, membres, planning).
 */
function sendShowState() {
    if (!ws) {
        addToLog("WebSocket non connectée");
        return false;
    }
    ws.send("/=type=SHOW_CTL_STATE");
    return false;
}

/**
 * sendShowNetState demande l'affichage de l'état local NET
 * (prédécesseur, successeur, membres, numéro de vue).
 */
function sendShowNetState() {
    if (!ws) {
        addToLog("WebSocket non connectée");
        return false;
    }
    ws.send("/=type=SHOW_NET_STATE");
    return false;
}

/**
 * sendAddSite envoie une demande d'ajout dynamique d'un nouveau site.
 * Le site cible (target) indique quel membre existant accueillera le nouveau site.
 */
function sendAddSite() {
    if (!siteActive) {
        addToLog("Site quitté : action impossible");
        return false;
    }
    if (!ws) {
        addToLog("WebSocket non connectée");
        return false;
    }

    var site   = document.getElementById("newSite").value;
    var target = document.getElementById("joinTarget").value;
    var port   = document.getElementById("newPort").value;

    var msg =
        "/=type=JOIN_SITE"  +
        "/=site="   + site   +
        "/=target=" + target +
        "/=port="   + port;

    addToLog("Envoi : " + msg);
    ws.send(msg);
    return false;
}

/**
 * sendLeaveRequest envoie une demande de départ volontaire du site.
 * Après cette action, le site devient passif et quitte l'anneau.
 */
function sendLeaveRequest() {
    if (!siteActive) {
        addToLog("Site déjà quitté");
        return false;
    }
    if (!ws) {
        addToLog("WebSocket non connectée");
        return false;
    }
    ws.send("/=type=LEAVE_SITE");
    return false;
}