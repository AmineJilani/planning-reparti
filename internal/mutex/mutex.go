package mutex

import "strconv"

// Request représente une demande d'accès à une section critique.
// Dans notre projet, la section critique correspond à une ressource du planning,
// par exemple : SalleA|2026-04-22.
// STURCTURE QUI DESIGNE UNE DEMANDE D'ACCES A UNE RESSOURCE PROTEGEE
// DEMANDE D'ENTRER DANS LA ZONE CRITIQUE (POUR MODIFIER UNE SALLE A UNE DATE)
type MutexRequest struct {
	Site string // site qui demande l'acces
	Hlg  int    // horloge logique
	Ress string //ressource demandée( salle|date )
}

// State contient l'état local du mutex pour un site.
// On garde une file d'attente par ressource, ainsi que les ACK différés.
type State struct {
	RequestQueues map[string][]MutexRequest
	DeferredAcks  map[string][]string
}

// NewState initialise l'état mutex.
func NewState() *State {
	return &State{
		RequestQueues: make(map[string][]MutexRequest),
		DeferredAcks:  make(map[string][]string),
	}
}

// BeforeRequest compare deux demandes.
// La demande avec la plus petite horloge est prioritaire.
// En cas d'égalité, on départage avec le nom du site.
// return true si a est prioritaire sur b
func BeforeRequest(a, b MutexRequest) bool {
	if a.Hlg < b.Hlg {
		return true
	}
	if a.Hlg > b.Hlg {
		return false
	}
	return a.Site < b.Site
}

/*AddRequest ajoute une demande dans la file de sa ressource,
puis trie la file selon l'ordre des estampilles.*/
func (m *State) AddRequest(req MutexRequest) {
	queue := m.RequestQueues[req.Ress]
	queue = append(queue, req)
	for i := 0; i < len(queue); i++ {
		for j := i + 1; j < len(queue); j++ {
			if BeforeRequest(queue[j], queue[i]) {
				queue[i], queue[j] = queue[j], queue[i]
			}
		}
	}
	m.RequestQueues[req.Ress] = queue
}

/* RemoveRequest retire la demande d'un site pour une ressource.
elle est appelée quand un RELEASE est reçu.*/
func (m *State) RemoveRequest(site string, res string) {
	queue := m.RequestQueues[res]
	newQueue := []MutexRequest{}
	for _, req := range queue {
		if req.Site != site {
			newQueue = append(newQueue, req)
		}
	}

	m.RequestQueues[res] = newQueue
}

// IsFirst vérifie si un site est en tête de file pour une ressource.
func (m *State) IsFirst(site string, res string) bool {
	queue := m.RequestQueues[res]
	if len(queue) == 0 {
		return false
	}

	return queue[0].Site == site
}

// QueueString retourne une représentation lisible de la file.
// Exemple : "(A,1) (B,3)"
func (m *State) QueueString(res string) string {
	queue := m.RequestQueues[res]
	out := ""
	for _, req := range queue {
		out += "(" + req.Site + "," + strconv.Itoa(req.Hlg) + ") "
	}

	return out
}

// DeferAck mémorise qu'un ACK doit être envoyé plus tard à un site.
func (m *State) DeferAck(res string, site string) {
	m.DeferredAcks[res] = append(m.DeferredAcks[res], site)
}

// GetDeferredAcks retourne la liste des sites en attente d'ACK pour une ressource.
func (m *State) GetDeferredAcks(res string) []string {
	return m.DeferredAcks[res]
}

// ClearDeferredAcks supprime les ACK différés après leur émission.
func (m *State) ClearDeferredAcks(res string) {
	delete(m.DeferredAcks, res)
}
