/**
 * app.js
 * Initialisation des événements de l'interface web.
 *
 * Ce fichier associe chaque bouton de l'interface
 * à sa fonction handler correspondante.
 * Il doit être chargé après tous les autres fichiers JS.
 */

document.getElementById("connecter").onclick    = connectWebSocket;
document.getElementById("fermer").onclick       = closeWebSocket;
document.getElementById("reserver").onclick     = sendReservation;
document.getElementById("snapshot").onclick     = sendSnapshot;
document.getElementById("showState").onclick    = sendShowState;
document.getElementById("showNetState").onclick = sendShowNetState;
document.getElementById("joinSite").onclick     = sendAddSite;
document.getElementById("leaveSite").onclick    = sendLeaveRequest;