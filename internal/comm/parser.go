/*CE FICHIER SERT A LIRE LES CHAMPS DANS UN MESSAGE (ISSUS DE message.go)*/

package comm

import "strings"

func FindVal(msg string, key string) string {
	//FindVal récupère la valeur associé à une clé dans un message
	if len(msg) < 4 {
		return ""
	}

	sep := msg[0:1]                    // le premier caratère = séparateur de champs
	all := strings.Split(msg[1:], sep) //découpage du message en plusieurs paires (clé-valeur)

	for _, item := range all {
		if len(item) < 3 {
			continue
		}

		eq := item[0:1]                      // premier caratère de chaque paire = séparateur clé-valeur
		parts := strings.Split(item[1:], eq) // découpage du paire pour récupérer clé et valeur

		if len(parts) != 2 {
			continue
		}

		if parts[0] == key {
			return parts[1] // retourner  valeur
		}
	}

	return ""
}
