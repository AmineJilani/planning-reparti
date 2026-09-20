/*
Package snapshot — state.go
État local lié aux snapshots répartis.

Ce package implémente un mécanisme de snapshot réparti simplifié,
inspiré de l'algorithme de Chandy-Lamport.

L'état snapshot d'un site contient :
  - les snapshots déjà réalisés (pour éviter les doublons) ;
  - les messages capturés pendant un snapshot en cours ;
  - les états locaux collectés par l'initiateur ;
  - les informations de terminaison du snapshot global.

Limite actuelle :

	La collecte complète des messages prépost (en transit au moment
	du snapshot) n'est pas encore implémentée.
*/
package snapshot

// State représente l'état local lié aux snapshots d'un site.
type State struct {
	// identifiant du site qui a initié le snapshot courant
	Initiator string

	// couleur du site pour le protocole snapshot (white ou red)
	Color string

	// indique si ce site est l'initiateur du snapshot courant
	IsInitiator bool

	// compteur de messages applicatifs en transit
	// incrémenté à l'émission, décrémenté à la réception
	Bilan int

	// nombre d'états locaux encore attendus par l'initiateur
	NbEtatsAttendus int

	// nombre de messages prépost encore attendus (non utilisé dans la version actuelle)
	NbMsgAttendus int

	// identifiant du snapshot actuellement en cours
	CurrentSnapshotID string

	// snapshots déjà réalisés par ce site (clé : snapshotID)
	SnapshotsDone map[string]bool

	// snapshots globaux terminés (clé : snapshotID)
	SnapshotsClosed map[string]bool

	// messages reçus pendant chaque snapshot (clé : snapshotID)
	SnapshotMessages map[string][]string

	// snapshots dont la terminaison a été détectée (clé : snapshotID)
	SnapshotTerminated map[string]bool

	// états locaux collectés par l'initiateur
	CollectedStates []string

	// sites dont l'état a déjà été reçu (pour éviter les doublons)
	CollectedStateSites map[string]bool

	// messages prépost collectés (non utilisé dans la version actuelle)
	CollectedPreposts []string

	// identifiants des messages prépost déjà reçus (pour éviter les doublons)
	CollectedPrepostIDs map[string]bool
}

// NewState initialise et retourne un nouvel état snapshot.
// Le site démarre en couleur blanche, sans snapshot en cours.
func NewState() *State {
	return &State{
		Color:             "white",
		IsInitiator:       false,
		Bilan:             0,
		CurrentSnapshotID: "",

		SnapshotsDone:      make(map[string]bool),
		SnapshotsClosed:    make(map[string]bool),
		SnapshotMessages:   make(map[string][]string),
		SnapshotTerminated: make(map[string]bool),

		CollectedStates:     []string{},
		CollectedStateSites: make(map[string]bool),
		CollectedPreposts:   []string{},
		CollectedPrepostIDs: make(map[string]bool),
	}
}

// IsDone indique si un snapshot a déjà été réalisé par ce site.
// Permet d'éviter de sauvegarder deux fois le même état local.
func (s *State) IsDone(snapshotID string) bool {
	return s.SnapshotsDone[snapshotID]
}

// MarkDone marque un snapshot comme réalisé par ce site.
func (s *State) MarkDone(snapshotID string) {
	s.SnapshotsDone[snapshotID] = true
}

// CaptureMessage enregistre un message reçu pendant un snapshot en cours.
// Ces messages sont capturés pour permettre une future collecte complète
// des messages en transit au moment du snapshot.
func (s *State) CaptureMessage(snapshotID string, msg string) {
	s.SnapshotMessages[snapshotID] = append(s.SnapshotMessages[snapshotID], msg)
}

// Messages retourne la liste des messages capturés pour un snapshot donné.
func (s *State) Messages(snapshotID string) []string {
	return s.SnapshotMessages[snapshotID]
}

// ActiveSnapshots retourne les identifiants de tous les snapshots
// déjà déclenchés sur ce site.
func (s *State) ActiveSnapshots() []string {
	ids := []string{}
	for id, done := range s.SnapshotsDone {
		if done {
			ids = append(ids, id)
		}
	}
	return ids
}

// IsClosed indique si le snapshot global est terminé.
// Un snapshot global est terminé lorsque l'initiateur
// a collecté tous les états locaux attendus.
func (s *State) IsClosed(snapshotID string) bool {
	return s.SnapshotsClosed[snapshotID]
}

// MarkClosed marque le snapshot global comme terminé.
func (s *State) MarkClosed(snapshotID string) {
	s.SnapshotsClosed[snapshotID] = true
}
