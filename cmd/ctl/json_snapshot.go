/*
JSON SNAPSHOT :
Persistance des snapshots.

Responsabilités :
- création du dossier snapshots/
- sérialisation des états en JSON
- sauvegarde des configurations locales
- sauvegarde des configurations globales
*/

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"planning-reparti/internal/state_site"
	"planning-reparti/internal/utils"
)

func saveSnapshotJSON(st *state_site.SiteState, snapshotID string, logger *utils.Logger) {
	type ReservationJSON struct {
		ID    string `json:"id"`
		Room  string `json:"room"`
		Date  string `json:"date"`
		Start string `json:"start"`
		End   string `json:"end"`
		User  string `json:"user"`
		Site  string `json:"site"`
	}

	type SnapshotJSON struct {
		SnapshotID      string            `json:"snapshotId"`
		Initiator       string            `json:"initiator"`
		Site            string            `json:"site"`
		Clock           int               `json:"clock"`
		Vector          string            `json:"vector"`
		Bilan           int               `json:"bilan"`
		Planning        []ReservationJSON `json:"planning"`
		CollectedStates []string          `json:"collectedStates"`
	}

	data := SnapshotJSON{
		SnapshotID:      snapshotID,
		Initiator:       st.Snapshot.Initiator,
		Site:            st.Name,
		Clock:           st.Clock.Time,
		Bilan:           st.Snapshot.Bilan,
		Vector:          st.Vector.String(),
		CollectedStates: st.Snapshot.CollectedStates,
	}

	for _, r := range st.Planning.List() {
		data.Planning = append(data.Planning, ReservationJSON{
			ID:    r.ID,
			Room:  r.Room,
			Date:  r.Date,
			Start: r.Start,
			End:   r.End,
			User:  r.User,
			Site:  r.Site,
		})
	}

	_ = os.MkdirAll("snapshots", 0755)

	filePath := filepath.Join("snapshots", snapshotID+"-"+st.Name+".json")

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.Error("snapshot", "erreur JSON : "+err.Error())
		return
	}

	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		logger.Error("snapshot", "erreur écriture JSON : "+err.Error())
		return
	}

	logger.Info("snapshot", "fichier JSON créé : "+filePath)
}
