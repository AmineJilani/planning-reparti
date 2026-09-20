package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"planning-reparti/internal/comm"
)

var (
	routes = make(map[string]string) // routes contient la table dynamique de routage , stocke un identifiant de site à son tube d'entrée
	mu     sync.Mutex                // Mutex utilisé pour proteger l'acces concurrent à la table routes
)

func main() {
	fmt.Fprintln(os.Stderr, "[router] démarré")

	seen := make(map[string]bool) //map pour retenir les sites déja surveillés
	for {
		files, _ := filepath.Glob("/tmp/out_*") //chercher les fichiers où nom commence par /tmp/out_
		for _, outPath := range files {
			site := strings.TrimPrefix(filepath.Base(outPath), "out_") // on extrait le nom de site
			//on véirifie si le site est déja surveillé
			if seen[site] {
				continue //si surveillé on poursuit
			}

			seen[site] = true
			fmt.Fprintln(os.Stderr, "[router] surveillance", site, "->", outPath)
			//on lance une goroutine pour lire la sortie du si il n'est pas surveillé (par exp : le routeur ouvre /tmp/out_D et lit tous les messages que NET-D écrit dedans)
			go readSite(site, outPath) // chaque sortie de site est lue indépendammment --> routeur peut recevoir des msgs de plusieurs sites
		}

		time.Sleep(1 * time.Second)
	}
}

/*readSite lit en continu les messages produits par un site (goroutine dans main)
 */
func readSite(site string, outPath string) {
	file, err := os.OpenFile(outPath, os.O_RDONLY, os.ModeNamedPipe) //Le routeur ouvre le tube de sortie du site.
	if err != nil {
		fmt.Fprintln(os.Stderr, "[router] erreur ouverture", outPath, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		//chaque ligne correspond à un message structuré
		line := scanner.Text()
		// le msg est transmis au routeur pour l'analyser ou rediriger
		handleMessage(site, line)
	}
}

/*handleMessage analyse un message (msg) reçu par un site
*puis il décide s'il faut enregistrer le site, envoyer à un destinataire ou diffuser à tous le monde le msg
 */
func handleMessage(from string, line string) {
	typ := comm.FindVal(line, "type")

	//REGISTER permet à un site de s'enregistrer dynamiquement auprés du routeur
	if typ == "REGISTER" {
		site := comm.FindVal(line, "site")
		input := comm.FindVal(line, "input")
		in := "/tmp/" + input //chemin complet du tube d'entrée (/tmp/in_A par exemple)
		// on protege la table de routage par mutex
		mu.Lock()
		routes[site] = in
		mu.Unlock()

		fmt.Fprintln(os.Stderr, "[router] REGISTER", site, "->", in)
		return
	}
	// Tous les autres messages doivent contenir un champ "to"
	to := comm.FindVal(line, "to")

	if to == "" {
		fmt.Fprintln(os.Stderr, "[router] message sans to ignoré :", line)
		return
	}
	// to=ALL signifie que le msg doit etre diffuser à tous les sites enregitrés
	if to == "ALL" {
		broadcast(from, line)
		return
	}
	//sinon le msg est envoyé uniquement au destinataire indiqué par "to"
	sendTo(to, line)
}

/*sendTo envoie un message vers un site précis
**/
func sendTo(to string, line string) {
	mu.Lock()
	inPath, ok := routes[to]
	mu.Unlock()

	if !ok {
		fmt.Fprintln(os.Stderr, "[router] destination inconnue :", to)
		return
	}

	file, err := os.OpenFile(inPath, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[router] erreur écriture vers", to, err)
		return
	}
	defer file.Close()
	//le routeur écrit le message dans l'entrée du destinataire
	fmt.Fprintln(file, line)
}

/*broadcast diffuse un message à tous les sites enregistrés
*par défaut, on ne renvoie pas le message à l'émetteur
 */
func broadcast(from string, line string) {
	mu.Lock()
	//on copie la liste des sites
	sites := make([]string, 0, len(routes))
	for site := range routes {
		sites = append(sites, site)
	}
	mu.Unlock()

	for _, site := range sites {
		// on ne renvoie pas le message à l'émetteur pour éviter les boucles inutiles sauf SHOW STATE pour debug
		if site == from && comm.FindVal(line, "type") != "SHOW_STATE" {
			continue
		}
		sendTo(site, line)
	}
}
