/*
state.go
État local d'un contrôleur NET.

Chaque site possède sa propre instance de State qui centralise
toutes les informations nécessaires à la gestion du réseau :
  - la liste des membres connus du groupe ;
  - la topologie logique de l'anneau (prédécesseur, successeur) ;
  - l'état de l'admission en cours ;
  - les messages de diffusion déjà traités ;
  - les informations de surveillance pour la détection de panne ;
  - l'état passif du site après un départ.
*/
package net

import (
	"sort"
	"strings"
	"time"
)

// State représente l'état local complet d'un site NET.
type State struct {
	// identifiant unique du site dans le réseau
	Name string

	// ensemble des membres logiques connus du groupe
	Members map[string]bool

	// identifiants des messages de diffusion déjà traités
	// permet d'éviter de traiter plusieurs fois le même message
	SeenBroadcasts map[string]bool

	// état de l'admission en cours
	Admission AdmissionState

	// successeur du site dans l'anneau logique
	Successor string

	// prédécesseur du site dans l'anneau logique
	Predecessor string

	// indique si le site est officiellement membre du groupe
	Joined bool

	// site contacté pour rejoindre le groupe (utilisé par retryJoin)
	JoinTarget string

	// dernière date à laquelle chaque membre a été vu (pour le heartbeat)
	LastSeen map[string]time.Time

	// indique si le site est passif après un départ volontaire
	Passive bool

	// numéro de vue courant du groupe
	// incrémenté à chaque modification de la composition du groupe
	MembershipVerion int

	// table de topologie distribuée : associe chaque site à ses voisins
	// utilisée pour reconfigurer l'anneau en cas de panne
	Topology map[string]NodeLinks

	// chemin vers la FIFO d'écriture vers CTL
	CtlOut string
}

// AdmissionState représente l'état d'une admission en cours.
// Il permet de gérer les conflits entre admissions concurrentes
// grâce au mécanisme d'élection par extinction.
type AdmissionState struct {
	// indique si une admission est actuellement en cours
	Running bool

	// identifiant unique de l'admission (format : ADM-hôte-nouveauSite)
	ID string

	// site qui accueille le nouveau participant
	Host string

	// site qui souhaite rejoindre le groupe
	Joining string

	// gagnant actuel de l'élection en cours
	Winner string
}

// NodeLinks représente les liens d'un site dans l'anneau logique.
// Cette structure est utilisée dans la table de topologie distribuée.
type NodeLinks struct {
	Pred string // prédécesseur du site
	Succ string // successeur du site
}

// NewState initialise et retourne un nouvel état NET pour un site.
// Le site est automatiquement ajouté à sa propre liste de membres.
func NewState(name string) *State {
	s := &State{
		Name:             name,
		Members:          make(map[string]bool),
		SeenBroadcasts:   make(map[string]bool),
		Joined:           false,
		Successor:        "",
		Predecessor:      "",
		LastSeen:         make(map[string]time.Time),
		Passive:          false,
		MembershipVerion: 1,
		Topology:         make(map[string]NodeLinks),
	}

	// le site se connaît lui-même comme membre
	s.Members[name] = true

	return s
}

// MembersString retourne la liste des membres du groupe
// sous forme de chaîne triée alphabétiquement.
// Exemple : "A,B,C,D"
func (s *State) MembersString() string {
	members := []string{}
	for member := range s.Members {
		members = append(members, member)
	}
	sort.Strings(members)
	return strings.Join(members, ",")
}
