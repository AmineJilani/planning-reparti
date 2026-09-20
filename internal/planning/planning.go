/*
planning.go
Logique métier du planning local d'un site.

Ce package gère le réplicat local du planning partagé entre tous les sites.
Chaque site possède sa propre instance de Planning qui est maintenue
cohérente grâce aux mécanismes d'exclusion mutuelle et de diffusion UPDATE.

Responsabilités :
  - création d'un planning vide ;
  - détection de conflits entre réservations ;
  - ajout de réservations avec ou sans vérification de conflit ;
  - consultation de la liste des réservations.
*/
package planning

// Planning représente le réplicat local du planning d'un site.
// Il contient la liste de toutes les réservations validées localement.
type Planning struct {
	Reservations []Reservation
}

// NewPlanning crée et retourne un planning vide.
func NewPlanning() *Planning {
	return &Planning{
		Reservations: []Reservation{},
	}
}

// overlaps vérifie si deux créneaux horaires se chevauchent.
// Les créneaux sont représentés par des chaînes au format "HH:MM".
// Deux créneaux se chevauchent si l'un commence avant que l'autre ne se termine.
func overlaps(start1, end1, start2, end2 string) bool {
	return start1 < end2 && start2 < end1
}

// HasConflict vérifie si une réservation entre en conflit
// avec une réservation déjà présente dans le planning.
// Un conflit existe si deux réservations concernent la même salle,
// la même date et des créneaux horaires qui se chevauchent.
func (p *Planning) HasConflict(r Reservation) bool {
	for _, existing := range p.Reservations {
		if existing.Room == r.Room &&
			existing.Date == r.Date &&
			overlaps(existing.Start, existing.End, r.Start, r.End) {
			return true
		}
	}
	return false
}

// AddReservation ajoute une réservation dans le planning sans vérification.
// Cette méthode est utilisée lorsque la réservation a déjà été validée,
// par exemple lors de la réception d'un message UPDATE ou SYNC_RESERVATION.
func (p *Planning) AddReservation(r Reservation) {
	p.Reservations = append(p.Reservations, r)
}

// AddReservationIfPossible tente d'ajouter une réservation après vérification.
// Retourne true si la réservation a été ajoutée, false en cas de conflit.
// Cette méthode est utilisée lors de l'entrée en section critique.
func (p *Planning) AddReservationIfPossible(r Reservation) bool {
	if p.HasConflict(r) {
		return false
	}
	p.AddReservation(r)
	return true
}

// List retourne toutes les réservations du planning local.
func (p *Planning) List() []Reservation {
	return p.Reservations
}
