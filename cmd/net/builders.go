/*
builders.go
Construction de tous les messages échangés entre les participants du réseau.

Chaque fonction retourne une chaîne au format clé/valeur
utilisé dans le système réparti.

Messages supportés :
  - REGISTER       : enregistrement auprès du routeur
  - JOIN_REQUEST   : demande d'admission dans le groupe
  - JOIN_ACCEPT    : acceptation d'un nouveau participant
  - MEMBER_LIST    : diffusion de la vue courante du groupe
  - ELECTION       : résolution des conflits d'admission simultanés
  - RECONFIGURE_ADD: reconfiguration de l'anneau après admission
  - LEAVE_REQUEST  : demande volontaire de départ
  - MEMBER_REMOVED : suppression d'un membre du groupe
  - PING / PONG    : messages de surveillance pour la détection de panne
  - VIEW_CHANGE    : notification de changement de vue vers CTL
  - NET_STATE      : état local NET envoyé vers CTL
*/

package main

import (
	"fmt"
	"planning-reparti/internal/comm"
	netstate "planning-reparti/internal/net"
)

// buildRegister construit le message d'enregistrement auprès du routeur.
// Ce message est envoyé au démarrage pour que le routeur construise
// sa table de routage dynamique : site → tube d'entrée.
func buildRegister(site string) string {
	return comm.Build(map[string]string{
		"type":  "REGISTER",
		"site":  site,
		"input": "in_" + site,
	})
}

// buildJoinRequest construit une demande d'admission dans le groupe.
// Le site src envoie ce message au site to, qui joue le rôle de point d'entrée.
func buildJoinRequest(site string, to string) string {
	return comm.Build(map[string]string{
		"type": "JOIN_REQUEST",
		"site": site,
		"to":   to,
	})
}

// buildJoinAccept construit la réponse positive à une demande d'admission.
// Elle contient le prédécesseur et le successeur du nouveau site dans l'anneau.
func buildJoinAccept(site string, to string, succ string) string {
	return comm.Build(map[string]string{
		"type": "JOIN_ACCEPT",
		"site": site,
		"to":   to,
		"succ": succ,
		"pred": site,
	})
}

// buildMemberList construit le message de diffusion de la vue courante du groupe.
// Il contient la liste complète des membres ainsi que le numéro de vue.
func buildMemberList(st *netstate.State) string {
	return comm.Build(map[string]string{
		"type":    "MEMBER_LIST",
		"site":    st.Name,
		"to":      "ALL",
		"id":      "ML-" + st.Name + "-" + st.MembersString(),
		"members": st.MembersString(),
		"view":    fmt.Sprintf("%d", st.MembershipVerion),
	})
}

// buildElection construit un message d'élection.
// Ce message circule dans l'anneau pour résoudre les conflits
// entre admissions simultanées. Le site avec l'identifiant le plus petit gagne.
func buildElection(site string, to string, id string, joining string, winner string) string {
	return comm.Build(map[string]string{
		"type":    "ELECTION",
		"site":    site,
		"to":      to,
		"id":      id,
		"joining": joining,
		"winner":  winner,
	})
}

// buildReconfigureAdd construit le message de reconfiguration de l'anneau
// suite à l'ajout d'un nouveau participant.
// Il permet à chaque site de mettre à jour son prédécesseur et son successeur.
func buildReconfigureAdd(site string, newNode string, oldSuccessor string) string {
	return comm.Build(map[string]string{
		"type":    "RECONFIGURE_ADD",
		"site":    site,
		"new":     newNode,
		"to":      "ALL",
		"oldsucc": oldSuccessor,
	})
}

// buildLeaveRequest construit la demande volontaire de départ du groupe.
func buildLeaveRequest(site string) string {
	return comm.Build(map[string]string{
		"type": "LEAVE_REQUEST",
		"site": site,
	})
}

// buildMemberRemoved construit le message informant l'ensemble des participants
// qu'un membre quitte le groupe, volontairement ou suite à une panne détectée.
// Il contient le prédécesseur et le successeur du site supprimé
// afin de permettre la reconfiguration de l'anneau.
func buildMemberRemoved(site string, pred string, succ string) string {
	return comm.Build(map[string]string{
		"type": "MEMBER_REMOVED",
		"site": site,
		"pred": pred,
		"succ": succ,
		"to":   "ALL",
	})
}

// buildPing construit un message de surveillance envoyé périodiquement
// à chaque membre connu pour vérifier qu'il est toujours actif.
func buildPing(site string, to string) string {
	return comm.Build(map[string]string{
		"type": "PING",
		"site": site,
		"to":   to,
	})
}

// buildPong construit la réponse à un message PING.
// À chaque PONG reçu, le champ LastSeen du membre est mis à jour.
func buildPong(site string, to string) string {
	return comm.Build(map[string]string{
		"type": "PONG",
		"site": site,
		"to":   to,
	})
}

// buildViewChange construit le message de changement de vue
// envoyé par NET vers CTL après chaque modification du groupe.
func buildViewChange(st *netstate.State) string {
	return comm.Build(map[string]string{
		"type":    "VIEW_CHANGE",
		"view":    fmt.Sprintf("%d", st.MembershipVerion),
		"members": st.MembersString(),
	})
}

// buildNetStateToCtl construit le message d'état local NET
// envoyé vers CTL pour lui communiquer la topologie courante.
func buildNetStateToCtl(st *netstate.State) string {
	return comm.Build(map[string]string{
		"type":    "NET_STATE",
		"site":    st.Name,
		"pred":    st.Predecessor,
		"succ":    st.Successor,
		"members": st.MembersString(),
		"view":    fmt.Sprintf("%d", st.MembershipVerion),
	})
}
