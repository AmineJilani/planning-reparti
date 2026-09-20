/*
state_site.go
État local global d'un site CTL.

Ce package centralise toutes les informations nécessaires
au fonctionnement du contrôleur CTL d'un site.

SiteState regroupe :
  - les horloges logique et vectorielle ;
  - le réplicat local du planning ;
  - l'état du mutex réparti ;
  - l'état des snapshots ;
  - les informations sur l'opération locale en attente ;
  - les messages déjà traités pour éviter les boucles.
*/
package state_site

import (
	"planning-reparti/internal/clock"
	"planning-reparti/internal/mutex"
	"planning-reparti/internal/planning"
	"planning-reparti/internal/snapshot"
)

// SiteState représente l'état local complet d'un site CTL.
// Il est créé au démarrage et partagé entre tous les handlers
// via un pointeur, garantissant un accès séquentiel grâce
// à l'architecture événementielle.
type SiteState struct {
	// identifiant unique du site
	Name string

	// compteur d'événements traités par ce site
	EventCount int

	// map des ACK reçus par ressource et par site
	// AckMap[ressource][site] = true si l'ACK a été reçu
	AckMap map[string]map[string]bool

	// nombre total de sites dans le système
	// mis à jour à chaque VIEW_CHANGE
	NumSites int

	// liste des sites connus du système
	// mise à jour à chaque VIEW_CHANGE
	Sites []string

	// indique si le site est actif ou passif
	// un site passif ignore tous les messages sauf SHOW_CTL_STATE
	Active bool

	// horloge logique de Lamport
	Clock *clock.LogicalClock

	// horloge vectorielle
	Vector *clock.VectorClock

	// réplicat local du planning partagé
	Planning *planning.Planning

	// état du mutex réparti (files d'attente, ACK différés)
	Mutex *mutex.State

	// état des snapshots répartis
	Snapshot *snapshot.State

	// type de l'opération locale en attente ("reserve" ou "")
	PendingOp string

	// réservation locale stockée en attente d'entrée en section critique
	PendingRes *planning.Reservation

	// horloge logique associée à la requête locale en attente
	// utilisée pour comparer les priorités avec les requêtes distantes
	PendingHlg int

	// identifiants des messages déjà traités
	// permet d'éviter les boucles dans l'anneau de diffusion
	SeenMessages map[string]bool

	// chemin vers la FIFO d'écriture vers NET
	NetworkOut string
}

// NewSiteState initialise et retourne un nouvel état local pour un site CTL.
// Toutes les structures de données sont initialisées à leur valeur par défaut.
func NewSiteState(name string, sites []string) *SiteState {
	return &SiteState{
		Name:       name,
		EventCount: 0,
		AckMap:     make(map[string]map[string]bool),
		NumSites:   len(sites),
		Sites:      sites,
		Active:     true,

		Clock:    clock.NewLogicalClock(name),
		Vector:   clock.NewVectorClock(name, sites),
		Planning: planning.NewPlanning(),
		Mutex:    mutex.NewState(),
		Snapshot: snapshot.NewState(),

		PendingOp:    "",
		PendingRes:   nil,
		PendingHlg:   0,
		SeenMessages: make(map[string]bool),

		NetworkOut: "",
	}
}

// IncEvent incrémente le compteur d'événements traités par le site.
// Appelée au début de chaque handleEvent.
func (s *SiteState) IncEvent() {
	s.EventCount++
}

// Before compare deux estampilles logiques et retourne true si a est prioritaire sur b.
// La priorité est déterminée par l'horloge logique.
// En cas d'égalité, le nom du site départage les deux estampilles
// par ordre alphabétique.
func Before(a, b clock.Stamp) bool {
	if a.Time < b.Time {
		return true
	}
	if a.Time == b.Time && a.Site < b.Site {
		return true
	}
	return false
}
