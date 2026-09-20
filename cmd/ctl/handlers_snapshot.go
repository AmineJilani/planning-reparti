/*
handlers_snapshot.go
Gestion des snapshots répartis.

Ce fichier implémente un mécanisme de snapshot réparti simplifié,
inspiré de l'algorithme de Chandy-Lamport.

Messages traités :
  - SNAPSHOT_LOCAL  : déclenchement local d'un snapshot par l'utilisateur
  - SNAPSHOT_START  : diffusion du snapshot aux autres sites
  - STATE           : réception des états locaux des autres sites

Fonctionnement :
  1. Le site initiateur sauvegarde son état local et diffuse SNAPSHOT_START.
  2. Chaque site qui reçoit SNAPSHOT_START sauvegarde son état
     et envoie un message STATE à l'initiateur.
  3. L'initiateur collecte tous les états et reconstruit
     une vue cohérente du système.

Limite actuelle :
  La collecte complète des messages prépost (messages en transit
  au moment du snapshot) n'a pas été implémentée.
*/

package main

import (
	"fmt"
	"planning-reparti/internal/comm"
	"planning-reparti/internal/state_site"
	"planning-reparti/internal/utils"
)

// handleSnapshotLocal déclenche un snapshot réparti à l'initiative du site local.
// Il sauvegarde l'état local, puis diffuse un message SNAPSHOT_START
// à tous les autres participants via le réseau.
func handleSnapshotLocal(st *state_site.SiteState, logger *utils.Logger, line string) {
	snapshotID := comm.FindVal(line, "snap")
	if snapshotID == "" {
		snapshotID = "S1"
	}

	// un snapshot déjà réalisé ne peut pas être relancé
	if st.Snapshot.IsDone(snapshotID) {
		logger.Info("snapshot", "snapshot déjà réalisé : "+snapshotID)
		return
	}

	// initialisation de l'état snapshot local
	st.Snapshot.Initiator = st.Name
	st.Snapshot.IsInitiator = true
	st.Snapshot.CurrentSnapshotID = snapshotID
	st.Snapshot.NbEtatsAttendus = st.NumSites - 1
	st.Snapshot.NbMsgAttendus = st.Snapshot.Bilan
	st.Snapshot.CollectedStates = []string{}
	st.Snapshot.CollectedStateSites = make(map[string]bool)
	st.Snapshot.CollectedPreposts = []string{}
	st.Snapshot.CollectedPrepostIDs = make(map[string]bool)

	logger.Info("snapshot", "déclenchement snapshot global : "+snapshotID)

	// sauvegarde de l'état local
	saveLocalSnapshot(st, logger, snapshotID)

	// diffusion du SNAPSHOT_START à tous les participants
	start := comm.Build(map[string]string{
		"type":      "SNAPSHOT_START",
		"site":      st.Name,
		"to":        "ALL",
		"initiator": st.Name,
		"snap":      snapshotID,
	})

	logger.Info("snapshot", "émission SNAPSHOT_START : "+start)
	sendNetwork(st, logger, start)
}

// handleSnapshotStart traite la réception d'un SNAPSHOT_START.
// Si le site n'a pas encore réalisé ce snapshot, il sauvegarde
// son état local et envoie un message STATE à l'initiateur.
// Le SNAPSHOT_START est ensuite retransmis pour continuer la diffusion.
func handleSnapshotStart(st *state_site.SiteState, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	initiator := comm.FindVal(line, "initiator")
	snapshotID := comm.FindVal(line, "snap")

	// le message est revenu à l'initiateur → fin de diffusion
	if st.Name == initiator && src == initiator {
		logger.Info("snapshot", "SNAPSHOT_START revenu à l'initiateur → fin de diffusion")
		return
	}

	if !st.Snapshot.IsDone(snapshotID) {
		st.Snapshot.Initiator = initiator
		st.Snapshot.CurrentSnapshotID = snapshotID

		logger.Info("snapshot", "SNAPSHOT_START reçu → sauvegarde locale")

		// sauvegarde de l'état local
		saveLocalSnapshot(st, logger, snapshotID)

		// envoi de l'état local vers l'initiateur
		state := buildState(st, snapshotID)
		logger.Info("snapshot", "émission STATE : "+state)
		sendNetwork(st, logger, state)
	}

	// retransmission du SNAPSHOT_START pour continuer la diffusion
	logger.Info("snapshot", "retransmission SNAPSHOT_START : "+line)
	sendNetwork(st, logger, line)
}

// handleState traite la réception d'un état local envoyé par un autre site.
// L'initiateur collecte les états reçus jusqu'à avoir reçu
// tous les états attendus, puis affiche et sauvegarde la configuration globale.
// Les états non destinés à l'initiateur sont retransmis dans l'anneau.
func handleState(st *state_site.SiteState, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	snapshotID := comm.FindVal(line, "snap")

	// le message est revenu à l'émetteur → fin de diffusion
	if src == st.Name {
		logger.Info("snapshot", "STATE revenu à l'émetteur → fin de diffusion")
		return
	}

	if st.Snapshot.IsInitiator && snapshotID == st.Snapshot.CurrentSnapshotID {
		// éviter de collecter deux fois l'état du même site
		if st.Snapshot.CollectedStateSites[src] {
			logger.Info("snapshot", "STATE déjà reçu de "+src+" → ignoré")
			return
		}

		// collecte de l'état reçu
		stateInfo := fmt.Sprintf(
			"site=%s clock=%s vector=%s bilan=%s",
			src,
			comm.FindVal(line, "clock"),
			comm.FindVal(line, "vec"),
			comm.FindVal(line, "bilan"),
		)

		st.Snapshot.CollectedStateSites[src] = true
		st.Snapshot.CollectedStates = append(st.Snapshot.CollectedStates, stateInfo)
		st.Snapshot.NbEtatsAttendus--

		sendApp(logger, line)

		logger.Info("snapshot", fmt.Sprintf(
			"STATE reçu de %s | états restants=%d",
			src,
			st.Snapshot.NbEtatsAttendus,
		))

		// tous les états ont été collectés → snapshot global terminé
		if st.Snapshot.NbEtatsAttendus == 0 {
			logger.Info("snapshot", "SNAPSHOT GLOBAL TERMINÉ")
			logger.Info("snapshot", fmt.Sprintf("bilan local=%d", st.Snapshot.Bilan))
			displayGlobalSnapshot(st, logger)
			saveSnapshotJSON(st, snapshotID, logger)
		}

		return
	}

	// retransmission du STATE vers l'initiateur
	logger.Info("snapshot", "retransmission STATE : "+line)
	sendNetwork(st, logger, line)
}

// saveLocalSnapshot sauvegarde l'état local du site au moment du snapshot.
// Elle enregistre : l'horloge logique, l'horloge vectorielle,
// la réservation en attente et le planning local.
// Si le site n'est pas l'initiateur, elle génère également le fichier JSON.
func saveLocalSnapshot(st *state_site.SiteState, logger *utils.Logger, snapshotID string) {
	// protection contre une double sauvegarde
	if st.Snapshot.IsDone(snapshotID) {
		logger.Info("snapshot", "snapshot déjà réalisé, ignoré")
		return
	}

	st.Snapshot.MarkDone(snapshotID)

	logger.Info("snapshot", "// SNAPSHOT "+snapshotID+" //")
	logger.Info("snapshot", "site="+st.Name)
	logger.Info("snapshot", fmt.Sprintf("clock=%d", st.Clock.Time))
	logger.Info("snapshot", "vector=["+st.Vector.String()+"]")

	// affichage de la réservation en attente si elle existe
	if st.PendingRes != nil {
		logger.Info("snapshot", "pending="+st.PendingRes.ID+" key="+st.PendingRes.Key())
	} else {
		logger.Info("snapshot", "pending=aucune")
	}

	// affichage du planning local
	logger.Info("snapshot", "planning local :")
	for _, r := range st.Planning.List() {
		logger.Info("snapshot", fmt.Sprintf(
			"%s | %s | %s %s-%s | %s | site=%s",
			r.ID, r.Room, r.Date, r.Start, r.End, r.User, r.Site,
		))
	}

	// affichage des messages capturés pendant le snapshot
	msgs := st.Snapshot.Messages(snapshotID)
	if len(msgs) == 0 {
		logger.Info("snapshot", "messages capturés : aucun")
	} else {
		logger.Info("snapshot", "messages capturés :")
		for _, m := range msgs {
			logger.Info("snapshot", m)
		}
	}

	// sauvegarde JSON uniquement pour les sites non initiateurs
	// l'initiateur sauvegarde après avoir collecté tous les états
	if !st.Snapshot.IsInitiator {
		saveSnapshotJSON(st, snapshotID, logger)
	}

	logger.Info("snapshot", "==============================")
}
