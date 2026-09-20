/*
display.go
Fonctions d'affichage et de notification vers APP.

Responsabilités :
  - affichage du planning local dans les logs ;
  - affichage des opérations en attente ;
  - affichage de la configuration globale du snapshot ;
  - émission des informations de planning vers APP ;
  - traitement du changement de vue (VIEW_CHANGE) ;
  - réponse à une demande d'état local (SHOW_CTL_STATE).
*/

package main

import (
	"fmt"
	"os"
	"planning-reparti/internal/clock"
	"planning-reparti/internal/comm"
	"planning-reparti/internal/planning"
	"planning-reparti/internal/state_site"
	"planning-reparti/internal/utils"
	"strings"
)

// displayReservations affiche toutes les réservations du planning local dans les logs.
func displayReservations(st *state_site.SiteState, logger *utils.Logger) {
	all := st.Planning.List()

	if len(all) == 0 {
		logger.Info("displayRes", "aucune réservation")
		return
	}

	logger.Info("displayRes", "planning local :")
	for _, r := range all {
		logger.Info("displayRes", fmt.Sprintf(
			"%s | %s | %s %s-%s | %s | site=%s | key=%s",
			r.ID, r.Room, r.Date, r.Start, r.End, r.User, r.Site, r.Key(),
		))
	}
}

// displayPending affiche l'opération locale actuellement en attente.
// Utile pour vérifier qu'une demande locale a bien été stockée
// avant l'entrée en section critique.
func displayPending(st *state_site.SiteState, logger *utils.Logger) {
	if st.PendingRes == nil {
		logger.Info("displayPend", "aucune opération en attente")
		return
	}

	logger.Info("displayPend", fmt.Sprintf(
		"pendingOp=%s | reservation=%s | key=%s",
		st.PendingOp,
		st.PendingRes.ID,
		st.PendingRes.Key(),
	))
}

// emitPlanning émet chaque réservation du planning local vers APP via stdout.
// APP transmet ensuite ces informations à l'interface web.
func emitPlanning(st *state_site.SiteState) {
	for _, r := range st.Planning.List() {
		msg := comm.Build(map[string]string{
			"type":  "PLANNING",
			"site":  r.Site,
			"id":    r.ID,
			"room":  r.Room,
			"date":  r.Date,
			"start": r.Start,
			"end":   r.End,
			"user":  r.User,
		})
		fmt.Fprintln(os.Stdout, msg)
		os.Stdout.Sync()
	}
}

// contains vérifie si une valeur est présente dans une liste de chaînes.
func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

// sendFutureReservationsToNewSite envoie toutes les réservations existantes
// à un nouveau site qui vient de rejoindre le groupe.
// Cela permet au nouveau site de synchroniser son planning local
// avec l'historique des réservations déjà validées.
func sendFutureReservationsToNewSite(st *state_site.SiteState, logger *utils.Logger, newSite string) {
	for _, r := range st.Planning.List() {
		msg := buildSyncReservation(st, r, newSite)
		logger.Info("sync", "envoi réservation future à "+newSite+" : "+r.ID)
		sendNetwork(st, logger, msg)
	}
}

// handleViewChange traite un changement de vue reçu depuis NET.
// Il met à jour la liste des membres, reconstruit l'horloge vectorielle
// et synchronise le planning vers les nouveaux sites détectés.
func handleViewChange(st *state_site.SiteState, logger *utils.Logger, line string) {
	members := comm.FindVal(line, "members")
	view := comm.FindVal(line, "view")
	oldSites := st.Sites

	// construction de la nouvelle liste des membres
	sites := []string{}
	for _, m := range strings.Split(members, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			sites = append(sites, m)
		}
	}

	st.Sites = sites
	st.NumSites = len(sites)

	// reconstruction de l'horloge vectorielle avec la nouvelle vue
	// en conservant les valeurs existantes pour les sites déjà connus
	oldVector := st.Vector
	st.Vector = clock.NewVectorClock(st.Name, sites)
	for _, site := range sites {
		st.Vector.Set(site, oldVector.Get(site))
	}

	// envoi du planning aux nouveaux sites détectés dans la vue
	for _, newSite := range sites {
		if !contains(oldSites, newSite) && newSite != st.Name {
			sendFutureReservationsToNewSite(st, logger, newSite)
		}
	}

	sendApp(logger, line)

	logger.Info("view", "VIEW_CHANGE reçu : view="+view+" members="+members)
	logger.Info("view", fmt.Sprintf("vue appliquée : nombre de sites=%d", st.NumSites))
	logger.Info("view", "nouveau vecteur : ["+st.Vector.String()+"]")
}

// sendPlanningToApp envoie une réservation individuelle vers APP.
// APP transmet ensuite cette information à l'interface web.
func sendPlanningToApp(st *state_site.SiteState, logger *utils.Logger, r planning.Reservation) {
	msg := comm.Build(map[string]string{
		"type":  "PLANNING",
		"site":  r.Site,
		"id":    r.ID,
		"room":  r.Room,
		"date":  r.Date,
		"start": r.Start,
		"end":   r.End,
		"user":  r.User,
	})

	sendApp(logger, msg)
	logger.Info("app", "planning envoyé à APP : "+msg)
}

// displayGlobalSnapshot affiche la configuration globale collectée
// par l'initiateur du snapshot dans les logs.
// Elle inclut l'état de l'initiateur, les états locaux reçus
// et les messages prépost collectés.
func displayGlobalSnapshot(st *state_site.SiteState, logger *utils.Logger) {
	logger.Info("snapshot", "===== CONFIGURATION GLOBALE =====")
	logger.Info("snapshot", "INITIATEUR : "+st.Snapshot.Initiator)
	logger.Info("snapshot", fmt.Sprintf(
		"site=%s clock=%d vector=%s bilan=%d",
		st.Name, st.Clock.Time, st.Vector.String(), st.Snapshot.Bilan,
	))
	logger.Info("snapshot", "ETATS LOCAUX :")
	for _, s := range st.Snapshot.CollectedStates {
		logger.Info("snapshot", s)
	}
	logger.Info("snapshot", "MESSAGES PREPOST :")
	if len(st.Snapshot.CollectedPreposts) == 0 {
		logger.Info("snapshot", "aucun")
	} else {
		for _, p := range st.Snapshot.CollectedPreposts {
			logger.Info("snapshot", p)
		}
	}
	logger.Info("snapshot", "==============================")
}

// handleShowCtlState traite une demande d'affichage de l'état local CTL.
// Il construit un message LOCAL_STATE contenant les informations clés du site
// et l'envoie vers APP, suivi de toutes les réservations du planning.
func handleShowCtlState(st *state_site.SiteState, logger *utils.Logger) {
	msg := fmt.Sprintf(
		"/=type=LOCAL_STATE/=site=%s/=events=%d/=clock=%d/=vector=%s/=members=%s",
		st.Name,
		st.EventCount,
		st.Clock.Time,
		st.Vector.String(),
		strings.Join(st.Sites, ","),
	)

	sendApp(logger, msg)

	for _, r := range st.Planning.List() {
		sendPlanningToApp(st, logger, r)
	}

	logger.Info("state", "état local envoyé à APP : "+msg)
}
