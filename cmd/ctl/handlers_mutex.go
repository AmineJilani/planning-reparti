/*
handlers_mutex.go
Gestion de l'exclusion mutuelle répartie.

Ce fichier implémente l'algorithme de Ricart-Agrawala adapté
au système de réservation réparti.

Messages traités :
  - RESERVE_LOCAL : demande locale de réservation d'un créneau
  - REQUEST       : demande d'accès à la section critique
  - ACK           : réponse à un REQUEST
  - UPDATE        : diffusion d'une modification du planning
  - RELEASE       : libération de la section critique

Fonctionnement :
  1. Le site stocke la réservation en attente et émet un REQUEST.
  2. Les autres sites répondent ACK immédiatement ou différent leur ACK
     si leur propre requête est prioritaire (estampille plus petite).
  3. Quand tous les ACK sont reçus et que le site est en tête de file,
     il entre en section critique, applique la réservation,
     diffuse UPDATE et envoie RELEASE.
  4. Les ACK différés sont envoyés après la sortie de section critique.
*/

package main

import (
	"fmt"
	"planning-reparti/internal/clock"
	"planning-reparti/internal/comm"
	"planning-reparti/internal/mutex"
	"planning-reparti/internal/planning"
	"planning-reparti/internal/state_site"
	"planning-reparti/internal/utils"
	"strconv"
)

// handleReserveLocal traite une demande locale de réservation.
// Le site stocke la réservation en attente, ajoute sa requête
// dans la file mutex locale et émet un REQUEST vers tous les autres sites.
func handleReserveLocal(st *state_site.SiteState, logger *utils.Logger, line string) {
	// extraction des informations de la réservation depuis le message
	r := planning.Reservation{
		ID:    comm.FindVal(line, "id"),
		Room:  comm.FindVal(line, "room"),
		Date:  comm.FindVal(line, "date"),
		Start: comm.FindVal(line, "start"),
		End:   comm.FindVal(line, "end"),
		User:  comm.FindVal(line, "user"),
		Site:  st.Name,
	}

	// stockage de la réservation en attente avant entrée en section critique
	st.PendingOp = "reserve"
	st.PendingRes = &r
	stamp := st.Clock.CurrentStamp()
	st.PendingHlg = stamp.Time

	// ajout de la requête locale dans la file mutex
	req := mutex.MutexRequest{
		Site: st.Name,
		Hlg:  stamp.Time,
		Ress: r.Key(),
	}
	st.Mutex.AddRequest(req)

	logger.Info("handleEvent", "réservation stockée en attente : "+r.ID)
	logger.Info("handleEvent", "ressource visée : "+r.Key())
	logger.Info("handleEvent", "file locale : "+st.Mutex.QueueString(r.Key()))
	displayPending(st, logger)

	// émission du REQUEST pour demander l'accès à la section critique
	request := buildRequest(st)
	if request != "" {
		logger.Info("handleEvent", "émission du REQUEST : "+request)
		st.Snapshot.Bilan++
		logger.Info("bilan", fmt.Sprintf("bilan=%d", st.Snapshot.Bilan))
		sendNetwork(st, logger, request)
	}
}

// handleRequest traite un message REQUEST reçu d'un autre site.
// Le site ajoute la requête distante dans sa file locale,
// puis décide d'envoyer un ACK immédiat ou de le différer
// selon la priorité des estampilles.
func handleRequest(st *state_site.SiteState, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	ts := comm.FindVal(line, "hlg")
	res := comm.FindVal(line, "res")

	// le REQUEST est revenu à l'émetteur → fin de diffusion dans l'anneau
	if src == st.Name {
		logger.Info("handleEvent", "REQUEST revenu à l'émetteur → fin de diffusion")
		return
	}

	st.Snapshot.Bilan--
	logger.Info("bilan", fmt.Sprintf("bilan=%d", st.Snapshot.Bilan))

	hlg, err := strconv.Atoi(ts)
	if err != nil {
		logger.Warn("handleEvent", "timestamp REQUEST invalide : "+ts)
		return
	}

	st.Clock.Update(hlg)
	mergeVectorFromMessage(st, line)

	// ajout de la requête distante dans la file mutex locale
	req := mutex.MutexRequest{
		Site: src,
		Hlg:  hlg,
		Ress: res,
	}
	st.Mutex.AddRequest(req)

	logger.Info("handleEvent", "file pour "+res+" : "+st.Mutex.QueueString(res))

	// décision : ACK immédiat ou différé selon la priorité des estampilles
	if mustDeferAck(st, req) {
		st.Mutex.DeferAck(res, src)
		logger.Info("handleEvent", "ACK différé pour "+src+" sur res="+res)
	} else {
		sendAck(st, logger, src, res)
	}

	// retransmission du REQUEST dans l'anneau
	logger.Info("handleEvent", "retransmission REQUEST : "+line)
	sendNetwork(st, logger, line)
}

// handleAck traite un message ACK reçu en réponse à un REQUEST.
// Quand tous les ACK sont reçus et que le site est en tête de file,
// il entre en section critique, applique la réservation,
// diffuse UPDATE et RELEASE, puis envoie les ACK différés.
func handleAck(st *state_site.SiteState, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	to := comm.FindVal(line, "to")
	res := comm.FindVal(line, "res")

	// ACK non destiné à ce site → retransmission
	if to != st.Name {
		logger.Info("handleEvent", "pas destiné à moi → retransmission ACK")
		sendNetwork(st, logger, line)
		return
	}

	st.Snapshot.Bilan--
	logger.Info("bilan", fmt.Sprintf("bilan=%d", st.Snapshot.Bilan))
	mergeVectorFromMessage(st, line)

	// aucune réservation en cours → ACK ignoré
	if st.PendingRes == nil {
		logger.Info("handleEvent", "ACK reçu mais aucune réservation en attente")
		return
	}

	// ACK pour une autre ressource → ignoré
	if st.PendingRes.Key() != res {
		logger.Info("handleEvent", "ACK ignoré (ressource différente)")
		return
	}

	// initialisation de la map ACK si nécessaire
	if st.AckMap[res] == nil {
		st.AckMap[res] = make(map[string]bool)
	}
	if !st.AckMap[res][src] {
		st.AckMap[res][src] = true
	}
	count := len(st.AckMap[res])

	logger.Info("handleEvent", fmt.Sprintf(
		"ACK reçu de %s pour res=%s | total=%d", src, res, count,
	))

	// vérification des conditions d'entrée en section critique :
	// tous les ACK reçus ET site en tête de file
	if st.PendingRes != nil {
		key := st.PendingRes.Key()
		if count >= st.NumSites-1 && st.Mutex.IsFirst(st.Name, key) {
			reserved := *st.PendingRes

			// tentative d'ajout au planning local
			if st.Planning.AddReservationIfPossible(reserved) {
				logger.Info("handleEvent", "TOUS LES ACK REÇUS → entrée en section critique")

				// nettoyage de la file et de la map ACK
				st.Mutex.RemoveRequest(st.Name, res)
				st.AckMap[res] = nil

				logger.Info("handleEvent", "réservation appliquée : "+reserved.ID)
				displayReservations(st, logger)
				emitPlanning(st)

				// diffusion UPDATE pour synchroniser les autres sites
				update := buildUpdate(st, reserved)
				logger.Info("handleEvent", "émission UPDATE : "+update)
				st.Snapshot.Bilan++
				logger.Info("bilan", fmt.Sprintf("bilan=%d", st.Snapshot.Bilan))
				sendNetwork(st, logger, update)

				// émission RELEASE pour libérer la ressource
				release := buildRelease(st, reserved.Key())
				logger.Info("handleEvent", "émission RELEASE : "+release)
				st.Snapshot.Bilan++
				logger.Info("bilan", fmt.Sprintf("bilan=%d", st.Snapshot.Bilan))
				sendNetwork(st, logger, release)

				// envoi des ACK différés aux sites qui attendaient
				sendDeferredAcks(st, logger, reserved.Key())

			} else {
				logger.Warn("handleEvent", "conflit détecté → réservation refusée : "+reserved.ID)
			}

			// nettoyage de l'état local après sortie de section critique
			delete(st.AckMap, res)
			st.PendingRes = nil
			st.PendingOp = ""
			st.PendingHlg = 0
		}
	}
}

// handleUpdate traite un message UPDATE reçu d'un autre site.
// Le site applique la même modification sur son réplicat local du planning
// afin de garantir la cohérence entre tous les réplicats.
func handleUpdate(st *state_site.SiteState, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	op := comm.FindVal(line, "op")
	ts := comm.FindVal(line, "hlg")

	// le UPDATE est revenu à l'émetteur → fin de diffusion
	if src == st.Name {
		logger.Info("handleEvent", "UPDATE revenu à l'émetteur → fin de diffusion")
		return
	}

	st.Snapshot.Bilan--
	logger.Info("bilan", fmt.Sprintf("bilan=%d", st.Snapshot.Bilan))

	hlg, err := strconv.Atoi(ts)
	if err != nil {
		logger.Warn("handleEvent", "horloge UPDATE invalide : "+ts)
		return
	}

	st.Clock.Update(hlg)
	mergeVectorFromMessage(st, line)

	logger.Info("handleEvent", fmt.Sprintf(
		"UPDATE reçu depuis %s | op=%s | hlg=%d", src, op, hlg,
	))

	// application de la réservation reçue sur le planning local
	if op == "reserve" {
		r := planning.Reservation{
			ID:    comm.FindVal(line, "id"),
			Room:  comm.FindVal(line, "room"),
			Date:  comm.FindVal(line, "date"),
			Start: comm.FindVal(line, "start"),
			End:   comm.FindVal(line, "end"),
			User:  comm.FindVal(line, "user"),
			Site:  src,
		}

		if st.Planning.AddReservationIfPossible(r) {
			logger.Info("handleEvent", "réservation ajoutée via UPDATE : "+r.ID)
			displayReservations(st, logger)
			emitPlanning(st)
		} else {
			logger.Warn("handleEvent", "conflit détecté lors UPDATE : "+r.ID)
		}
	}

	// retransmission du UPDATE dans l'anneau
	sendNetwork(st, logger, line)
}

// handleRelease traite un message RELEASE reçu d'un autre site.
// Le site retire la requête du site émetteur de sa file mutex locale,
// signalant que la ressource est à nouveau disponible.
func handleRelease(st *state_site.SiteState, logger *utils.Logger, line string) {
	src := comm.FindVal(line, "site")
	ts := comm.FindVal(line, "hlg")
	res := comm.FindVal(line, "res")

	// le RELEASE est revenu à l'émetteur → fin de diffusion
	if src == st.Name {
		logger.Info("handleEvent", "RELEASE revenu à l'émetteur → fin de diffusion")
		return
	}

	st.Snapshot.Bilan--
	logger.Info("bilan", fmt.Sprintf("bilan=%d", st.Snapshot.Bilan))

	hlg, err := strconv.Atoi(ts)
	if err != nil {
		logger.Warn("handleEvent", "horloge RELEASE invalide : "+ts)
		return
	}

	st.Clock.Update(hlg)
	mergeVectorFromMessage(st, line)

	// retrait de la requête du site émetteur de la file locale
	st.Mutex.RemoveRequest(src, res)

	logger.Info("handleEvent", fmt.Sprintf(
		"RELEASE reçu depuis %s pour ressource=%s", src, res,
	))

	// retransmission du RELEASE dans l'anneau
	sendNetwork(st, logger, line)
}

// sendDeferredAcks envoie tous les ACK qui avaient été différés
// pour une ressource donnée après la sortie de section critique.
// Ces ACK permettent aux sites en attente de progresser.
func sendDeferredAcks(st *state_site.SiteState, logger *utils.Logger, res string) {
	sites := st.Mutex.GetDeferredAcks(res)

	if len(sites) == 0 {
		logger.Info("deferAck", "aucun ACK différé pour "+res)
		return
	}

	for _, site := range sites {
		ack := buildAck(st, site, res)
		logger.Info("deferAck", "émission ACK différé vers "+site+" : "+ack)
		sendNetwork(st, logger, ack)
	}

	st.Mutex.ClearDeferredAcks(res)
}

// mustDeferAck détermine si le site doit différer son ACK
// en réponse à une requête distante.
// Un ACK est différé si le site local a une requête en cours
// sur la même ressource avec une estampille plus petite (priorité locale).
func mustDeferAck(st *state_site.SiteState, req mutex.MutexRequest) bool {
	if st.PendingRes == nil {
		return false
	}
	if st.PendingRes.Key() != req.Ress {
		return false
	}

	localReq := mutex.MutexRequest{
		Site: st.Name,
		Hlg:  st.PendingHlg,
		Ress: req.Ress,
	}

	// la requête locale est prioritaire → on diffère l'ACK
	return mutex.BeforeRequest(localReq, req)
}

// sendAck construit et envoie un ACK immédiat vers un site.
func sendAck(st *state_site.SiteState, logger *utils.Logger, to string, res string) {
	ack := buildAck(st, to, res)
	logger.Info("handleEvent", "émission ACK : "+ack)
	st.Snapshot.Bilan++
	logger.Info("bilan", fmt.Sprintf("bilan=%d", st.Snapshot.Bilan))
	sendNetwork(st, logger, ack)
}

// mergeVectorFromMessage extrait l'horloge vectorielle d'un message reçu
// et la fusionne avec l'horloge vectorielle locale.
// Cette fusion garantit la cohérence causale entre les sites.
func mergeVectorFromMessage(st *state_site.SiteState, line string) {
	vec := comm.FindVal(line, "vec")
	if vec == "" {
		return
	}
	received := clock.ParseVectorString(vec)
	st.Vector.Merge(received)
}
