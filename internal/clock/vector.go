package clock

import (
	"sort"
	"strconv"
	"strings"
)

type VectorClock struct {
	Site   string
	Values map[string]int
}

func NewVectorClock(site string, sites []string) *VectorClock {
	values := make(map[string]int)

	for _, s := range sites {
		values[s] = 0
	}
	return &VectorClock{
		Site:   site,
		Values: values,
	}
}

func (v *VectorClock) Tick() {
	v.Values[v.Site]++
}

func (v *VectorClock) Merge(received map[string]int) {
	/*Lorsqu’un site reçoit une horloge vectorielle,
	il fusionne les informations en prenant le maximum pour chaque site.*/
	for site, value := range received {
		if value > v.Values[site] {
			v.Values[site] = value //nouvelle valeur = max(local, reçu)
		}
	}
	/*Ensuite, le site incrémente sa propre composante
	car la réception du message constitue un nouvel événement local.*/
	v.Tick()
}

// Construction de la chaine à partir de la map
func (v *VectorClock) String() string {
	var parts []string
	for site, value := range v.Values {
		parts = append(
			parts,
			site+":"+strconv.Itoa(value),
		)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func ParseVectorString(vec string) map[string]int {
	//retourne une représentation lisible de l'horloge vectorielle.
	result := map[string]int{}
	parts := strings.Split(vec, ",")

	for _, part := range parts {
		kv := strings.Split(part, ":")
		if len(kv) != 2 {
			continue
		}
		value, err := strconv.Atoi(kv[1])
		if err != nil {
			continue
		}
		result[kv[0]] = value
	}

	return result
}
func (vc *VectorClock) Get(site string) int {
	return vc.Values[site]
}
func (vc *VectorClock) Set(site string, value int) {
	vc.Values[site] = value
}
