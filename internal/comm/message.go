/*CE FICHIER message.go SERT A CREER DES MESSAGES STRUCTURES*/

package comm

import "strings"

const FieldSep = "/"  // séparateur entre les champs du message envoyé
const KeyValSep = "=" // séparateur entre clé et valeur

func Pair(key, val string) string {
	//cette fonction Pair construit une paire clé-valeur exemple: Pair("A","Int") DONNE "/=A=Int"
	return FieldSep + KeyValSep + key + KeyValSep + val
}

func Build(champs map[string]string) string {
	//Build construit un message complet à partir de plusieurs champs
	var msg strings.Builder
	for k, v := range champs {
		msg.WriteString(Pair(k, v)) //Appel à la fonction Pair pour chaque champ
	}
	return msg.String()
}
