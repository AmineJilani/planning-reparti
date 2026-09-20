/**
 * DOM.js
 * Fonctions de manipulation du DOM pour l'interface web.
 *
 * Responsabilités :
 *  - affichage des logs dans la zone dédiée ;
 *  - mise à jour du tableau de réservations ;
 *  - affichage des états snapshot collectés ;
 *  - désactivation des actions après départ du site.
 */

// stockage des réservations par clé unique (salle-date-début-fin)
var planningRows = {};

// indique si le site est encore actif dans le réseau
var siteActive = true;

/**
 * addToLog ajoute un message dans la zone de logs de l'interface web.
 * Le scroll est automatiquement positionné en bas.
 */
function addToLog(message) {
    var logs = document.getElementById("logs");
    var d = document.createElement("div");
    d.textContent = message;
    logs.appendChild(d);
    logs.scrollTop = logs.scrollHeight;
}

/**
 * updatePlanning met à jour le tableau de réservations affiché.
 * Chaque réservation est identifiée par une clé unique salle-date-début-fin
 * pour éviter les doublons lors des mises à jour successives.
 */
function updatePlanning(data) {
    if (data.type !== "PLANNING") return;

    var key =
        data.room  + "-" +
        data.date  + "-" +
        data.start + "-" +
        data.end;

    planningRows[key] = data;

    var body = document.getElementById("planningBody");
    body.innerHTML = "";

    Object.keys(planningRows).forEach(function(k) {
        var r = planningRows[k];
        var tr = document.createElement("tr");
        tr.innerHTML =
            "<td>" + (r.site  || "") + "</td>" +
            "<td>" + (r.id    || "") + "</td>" +
            "<td>" + (r.room  || "") + "</td>" +
            "<td>" + (r.date  || "") + "</td>" +
            "<td>" + (r.start || "") + "</td>" +
            "<td>" + (r.end   || "") + "</td>" +
            "<td>" + (r.user  || "") + "</td>";
        body.appendChild(tr);
    });
}

// stockage des états snapshot reçus par site
var snapshotStates = {};

/**
 * renderSnapshots affiche les états snapshot collectés
 * sous forme de cartes dans la zone dédiée de l'interface.
 */
function renderSnapshots() {
    var container = document.getElementById("snapshots");
    container.innerHTML = "";

    Object.keys(snapshotStates).forEach(function(site) {
        var s = snapshotStates[site];

        var card = document.createElement("div");
        card.className = "snapshot-card";
        card.innerHTML =
            "<strong>Site " + site + "</strong><br>" +
            "Snapshot : " + s.snap   + "<br>" +
            "Horloge : "  + s.clock  + "<br>" +
            "Vecteur : "  + s.vector + "<br>";

        // correction : appendChild après la construction du innerHTML
        container.appendChild(card);
    });
}

/**
 * updateSnapshot traite un message lié à un snapshot reçu depuis APP.
 * Si c'est un STATE, il met à jour l'état du site concerné et re-render.
 * Si c'est un SNAPSHOT_START, il log le déclenchement.
 */
function updateSnapshot(raw) {
    var data = parseProtocolMessage(raw);

    if (data.type === "STATE") {
        var site = data.site || "?";
        snapshotStates[site] = {
            snap:   data.snap || document.getElementById("snap").value || "?",
            clock:  data.clock || "?",
            vector: data.vec   || "?",
        };
        renderSnapshots();
        return;
    }

    if (data.type === "SNAPSHOT_START") {
        addToLog("Snapshot démarré : " + (data.snap || "?"));
    }

    addToLog("Snapshot : " + raw);
}

/**
 * disableActions désactive tous les boutons d'action de l'interface
 * lorsque le site quitte le réseau. Un message est ajouté dans les logs.
 */
function disableActions() {
    var buttons = [
        "reserver",
        "snapshot",
        "joinSite",
        "leaveSite",
        "showState",
        "showNetState"
    ];

    buttons.forEach(function(id) {
        var btn = document.getElementById(id);
        if (btn) btn.disabled = true;
    });

    addToLog("Site quitté : actions désactivées");
}