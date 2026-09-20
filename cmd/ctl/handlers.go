/*
handler.go
Boucle événementielle centrale du contrôleur CTL.

Ce fichier contient la fonction handleEvent qui reçoit tous les événements
entrants et les route vers le handler approprié.

Chaque événement est traité séquentiellement grâce au channel events
alimenté par une goroutine dédiée de lecture (readInput).
Cela garantit :
  - la séquentialité du traitement ;
  - l'atomicité des modifications de l'état local ;
  - l'absence de concurrence sur les structures de données.

Types d'événements traités :
  - RESERVE_LOCAL   : demande locale de réservation
  - REQUEST         : demande d'accès à la section critique
  - ACK             : réponse à un REQUEST
  - UPDATE          : synchronisation du planning
  - RELEASE         : libération de la section critique
  - SNAPSHOT_LOCAL  : déclenchement local d'un snapshot
  - SNAPSHOT_START  : diffusion du snapshot aux autres sites
  - STATE           : réception d'un état local pour le snapshot
  - VIEW_CHANGE     : changement de vue du groupe
  - SHOW_CTL_STATE  : demande d'affichage de l'état local
  - SHOW_NET_STATE  : demande d'affichage de l'état NET
  - NET_STATE       : état NET reçu depuis NET
  - JOIN_SITE       : information sur un nouveau site à ajouter
  - LEAVE_SITE      : demande de départ volontaire du site local
  - SYNC_RESERVATION: synchronisation d'une réservation vers un nouveau site
*/

package main

import (
	"fmt"
	"planning-reparti/internal/comm"
	"planning-reparti/internal/planning"
	"planning-reparti/internal/state_site"
	"planning-reparti/internal/utils"
)

// handleEvent est le point d'entrée de tous les événements du contrôleur.
// Il incrémente les horloges, détecte les doublons, capture les messages
// pour les snapshots en cours, et route vers le handler approprié.
func handleEvent(st *state_site.SiteState, logger *utils.Logger, line string) {
	// incrémentation des horloges à chaque événement
	st.IncEvent()
	st.Clock.Tick()
	st.Vector.Tick()

	stamp := st.Clock.CurrentStamp()
	logger.Debugf("handleEvent",
		"event #%d | estampille=(%d,%s) | vector=[%s]",
		st.EventCount, stamp.Time, stamp.Site, st.Vector.String(),
	)

	typ := comm.FindVal(line, "type")

	// détection des doublons pour les messages réseau circulant dans l'anneau
	// les événements locaux et les messages de contrôle sont exclus de cette vérification
	if typ != "RESERVE_LOCAL" &&
		typ != "SNAPSHOT_LOCAL" &&
		typ != "SHOW_CTL_STATE" &&
		typ != "SHOW_NET_STATE" &&
		typ != "NET_STATE" &&
		typ != "VIEW_CHANGE" &&
		typ != "LEAVE_SITE" {
		msgID := messageID(line)
		if st.SeenMessages[msgID] {
			logger.Info("handleEvent", "message déjà traité → ignoré : "+msgID)
			return
		}
		st.SeenMessages[msgID] = true
	}

	// capture des messages applicatifs pendant un snapshot en cours
	for _, snapID := range st.Snapshot.ActiveSnapshots() {
		if typ != "SNAPSHOT_LOCAL" {
			st.Snapshot.CaptureMessage(snapID, line)
			logger.Info("snapshot", "message capturé pour "+snapID+" : "+line)
		}
	}

	// un site inactif (après départ) ignore tous les messages
	// sauf la demande d'affichage de son état local
	if !st.Active && typ != "SHOW_CTL_STATE" {
		logger.Info("main", "site inactif : commande ignorée")
		return
	}

	// routage vers le handler approprié selon le type de message
	switch typ {

	case "RESERVE_LOCAL":
		handleReserveLocal(st, logger, line)

	case "REQUEST":
		handleRequest(st, logger, line)

	case "ACK":
		handleAck(st, logger, line)

	case "UPDATE":
		handleUpdate(st, logger, line)

	case "RELEASE":
		handleRelease(st, logger, line)

	case "SNAPSHOT_LOCAL":
		handleSnapshotLocal(st, logger, line)

	case "SNAPSHOT_START":
		handleSnapshotStart(st, logger, line)

	case "STATE":
		handleState(st, logger, line)

	case "VIEW_CHANGE":
		// traitement du changement de vue puis notification vers APP
		handleViewChange(st, logger, line)
		sendApp(logger, line)

	case "SHOW_CTL_STATE":
		handleShowCtlState(st, logger)

	case "SHOW_NET_STATE":
		// transmission de la demande vers NET avec l'identité du site
		msg := line + "/=to=" + st.Name
		sendNetwork(st, logger, msg)

	case "NET_STATE":
		// transmission de l'état NET reçu vers APP
		sendApp(logger, line)

	case "JOIN_SITE":
		handleJoinSite(st, logger, line)

	case "LEAVE_SITE":
		handleLeaveSite(st, logger)

	case "SYNC_RESERVATION":
		handleSyncReservation(st, logger, line)
	}
}

// handleJoinSite traite une information sur un nouveau site à ajouter.
// CTL informe APP du site à ajouter via le script dédié.
func handleJoinSite(st *state_site.SiteState, logger *utils.Logger, line string) {
	site := comm.FindVal(line, "site")
	logger.Info("join", "demande d'ajout du site "+site)
	sendApp(logger,
		"/=type=INFO/=msg=Ajouter le site "+site+" avec ./scripts/add-site.sh "+site,
	)
}

// handleLeaveSite traite la demande de départ volontaire du site local.
// Le site devient inactif et envoie un LEAVE_REQUEST vers NET
// afin que la reconfiguration de l'anneau soit effectuée.
func handleLeaveSite(st *state_site.SiteState, logger *utils.Logger) {
	msg := comm.Build(map[string]string{
		"type": "LEAVE_REQUEST",
		"site": st.Name,
		"to":   st.Name,
	})

	// le site devient passif avant d'envoyer le message
	st.Active = false

	logger.Info("leave", "demande de départ volontaire : "+msg)
	sendNetwork(st, logger, msg)
	sendApp(logger, "/=type=STATUS/=state=LEFT/=msg=Site quitté")
}

// handleSyncReservation traite la réception d'une réservation de synchronisation.
// Ce message est envoyé par le coordinateur à un nouveau site
// qui vient de rejoindre le groupe, afin de lui transmettre
// l'historique des réservations existantes.
func handleSyncReservation(st *state_site.SiteState, logger *utils.Logger, line string) {
	mergeVectorFromMessage(st, line)

	r := planning.Reservation{
		ID:    comm.FindVal(line, "id"),
		Room:  comm.FindVal(line, "room"),
		Date:  comm.FindVal(line, "date"),
		Start: comm.FindVal(line, "start"),
		End:   comm.FindVal(line, "end"),
		User:  comm.FindVal(line, "user"),
		Site:  comm.FindVal(line, "site"),
	}

	// vérification de conflit avant ajout
	if st.Planning.HasConflict(r) {
		logger.Info("sync", "réservation ignorée car conflit : "+r.ID)
		return
	}

	st.Planning.AddReservation(r)
	sendPlanningToApp(st, logger, r)

	logger.Info("sync", fmt.Sprintf("nb réservations=%d", len(st.Planning.List())))
	logger.Info("sync", "réservation synchronisée : "+r.ID)
}
