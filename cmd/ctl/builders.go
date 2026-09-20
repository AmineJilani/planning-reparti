/*
Package main — builders.go
Construction de tous les messages émis par le contrôleur CTL.

Chaque fonction retourne une chaîne au format clé/valeur
utilisé dans le système réparti.

Messages supportés :
  - REQUEST          : demande d'accès à la section critique
  - ACK              : réponse à un REQUEST
  - UPDATE           : diffusion d'une modification du planning
  - RELEASE          : libération de la section critique
  - STATE            : état local envoyé à l'initiateur du snapshot
  - SYNC_RESERVATION : synchronisation du planning vers un nouveau site
*/

package main

import (
	"fmt"
	"planning-reparti/internal/comm"
	"planning-reparti/internal/planning"
	"planning-reparti/internal/state_site"
)

// buildRequest construit un message REQUEST.
// Ce message signifie que le site demande l'accès à la section critique
// pour modifier une ressource du planning (salle|date).
// Il contient l'estampille logique du site ainsi que l'horloge vectorielle.
// Retourne une chaîne vide si aucune réservation n'est en attente.
func buildRequest(st *state_site.SiteState) string {
	if st.PendingRes == nil {
		return ""
	}

	stamp := st.Clock.CurrentStamp()
	return comm.Build(map[string]string{
		"type":      "REQUEST",
		"site":      st.Name,
		"hlg":       fmt.Sprintf("%d", stamp.Time),
		"res":       st.PendingRes.Key(),
		"color":     st.Snapshot.Color,
		"initiator": st.Snapshot.Initiator,
		"snap":      st.Snapshot.CurrentSnapshotID,
		"vec":       st.Vector.String(),
	})
}

// buildUpdate construit un message UPDATE après validation locale d'une réservation.
// Ce message est diffusé à tous les autres sites pour qu'ils appliquent
// la même modification sur leur réplicat local du planning.
func buildUpdate(st *state_site.SiteState, r planning.Reservation) string {
	stamp := st.Clock.CurrentStamp()
	return comm.Build(map[string]string{
		"type":      "UPDATE",
		"op":        "reserve",
		"site":      st.Name,
		"hlg":       fmt.Sprintf("%d", stamp.Time),
		"id":        r.ID,
		"room":      r.Room,
		"date":      r.Date,
		"start":     r.Start,
		"end":       r.End,
		"user":      r.User,
		"res":       r.Key(),
		"color":     st.Snapshot.Color,
		"initiator": st.Snapshot.Initiator,
		"snap":      st.Snapshot.CurrentSnapshotID,
		"vec":       st.Vector.String(),
	})
}

// buildRelease construit un message RELEASE.
// Ce message indique que le site a terminé sa section critique
// et libère la ressource pour les autres sites en attente.
// Les ACK différés sont ensuite envoyés après ce message.
func buildRelease(st *state_site.SiteState, ress string) string {
	stamp := st.Clock.CurrentStamp()
	return comm.Build(map[string]string{
		"type":      "RELEASE",
		"site":      st.Name,
		"hlg":       fmt.Sprintf("%d", stamp.Time),
		"res":       ress,
		"color":     st.Snapshot.Color,
		"initiator": st.Snapshot.Initiator,
		"snap":      st.Snapshot.CurrentSnapshotID,
		"vec":       st.Vector.String(),
	})
}

// buildAck construit un message ACK à destination d'un site.
// Un ACK est envoyé immédiatement si la requête distante est prioritaire,
// ou différé si le site local est lui-même en section critique
// sur la même ressource avec une estampille plus petite.
func buildAck(st *state_site.SiteState, to string, ress string) string {
	stamp := st.Clock.CurrentStamp()
	return comm.Build(map[string]string{
		"type":      "ACK",
		"site":      st.Name,
		"to":        to,
		"hlg":       fmt.Sprintf("%d", stamp.Time),
		"res":       ress,
		"color":     st.Snapshot.Color,
		"initiator": st.Snapshot.Initiator,
		"snap":      st.Snapshot.CurrentSnapshotID,
		"vec":       st.Vector.String(),
	})
}

// buildState construit un message STATE envoyé à l'initiateur du snapshot.
// Il contient l'horloge logique, l'horloge vectorielle
// et le bilan des messages applicatifs en transit au moment du snapshot.
func buildState(st *state_site.SiteState, snapshotID string) string {
	return comm.Build(map[string]string{
		"type":  "STATE",
		"site":  st.Name,
		"snap":  snapshotID,
		"to":    st.Snapshot.Initiator,
		"clock": fmt.Sprintf("%d", st.Clock.Time),
		"vec":   st.Vector.String(),
		"bilan": fmt.Sprintf("%d", st.Snapshot.Bilan),
	})
}

// buildSyncReservation construit un message SYNC_RESERVATION.
// Ce message est envoyé par le coordinateur à un nouveau site
// qui vient de rejoindre le groupe, afin de lui transmettre
// l'historique complet des réservations existantes.
func buildSyncReservation(st *state_site.SiteState, r planning.Reservation, to string) string {
	return comm.Build(map[string]string{
		"type":  "SYNC_RESERVATION",
		"to":    to,
		"id":    r.ID,
		"room":  r.Room,
		"date":  r.Date,
		"start": r.Start,
		"end":   r.End,
		"user":  r.User,
		"site":  r.Site,
		"vec":   st.Vector.String(),
	})
}
