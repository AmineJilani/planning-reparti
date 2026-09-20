/*
Package planning — reservation.go
Définition de la structure d'une réservation et de la ressource logique.

Une réservation identifie un créneau dans une salle à une date donnée.
La ressource logique protégée par l'exclusion mutuelle est définie
par le couple salle|date, ce qui permet des réservations parallèles
sur des salles ou des dates différentes.

Exemple de ressource : "SalleA|2026-04-22"
*/
package planning

// Reservation représente une réservation de salle dans le planning partagé.
type Reservation struct {
	ID    string // identifiant unique de la réservation
	Room  string // salle réservée
	Date  string // date de la réservation (format : YYYY-MM-DD)
	Start string // heure de début (format : HH:MM)
	End   string // heure de fin (format : HH:MM)
	User  string // utilisateur ayant effectué la réservation
	Site  string // site créateur de la réservation
}

// ResourceKey construit la clé de ressource logique à partir d'une salle et d'une date.
// Cette clé est utilisée par l'exclusion mutuelle pour identifier
// la ressource à protéger. Deux réservations sur des salles ou des dates
// différentes utilisent des ressources différentes et peuvent donc
// être traitées en parallèle sans blocage mutuel.
// Exemple : ResourceKey("SalleA", "2026-04-22") → "SalleA|2026-04-22"
func ResourceKey(room, date string) string {
	return room + "|" + date
}

// Key retourne la clé de ressource logique de la réservation.
// Raccourci vers ResourceKey(r.Room, r.Date).
func (r Reservation) Key() string {
	return ResourceKey(r.Room, r.Date)
}
