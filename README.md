# Planning réparti

Projet d'application répartie en Go pour la réservation de créneaux dans un planning partagé entre plusieurs sites.

## 1. Objectif du projet
L'objectif est de développer une application répartie où plusieurs sites disposent chacun d'un réplicat local du planning.

Chaque site peut demander une réservation. Pour garantir la cohérence des réplicats, une seule modification d'une même ressource est autorisée à la fois.

Le projet implémente :
- une donnée partagée répliquée : le planning
- une exclusion mutuelle répartie
- des estampilles logiques
- une horloge vectorielle
- un mécanisme de snapshot réparti
- une gestion dynamique de la composition du groupe (admission, départ, détection de panne)
- une interface web

## 2. Fonctionnalités principales

- Réservation distribuée de salles
- Exclusion mutuelle répartie
- Gestion des conflits concurrents
- ACK différés
- Horloges logiques de Lamport
- Horloges vectorielles
- Snapshot réparti inspiré de Chandy-Lamport
- Communication en anneau logique
- Admission dynamique de nouveaux sites, départ volontaire et détection de panne
- Routage dynamique des messages via un routeur central
- Interface web via WebSocket
- Architecture événementielle

## 3. Scénario choisi
Le scénario retenu est la réservation de salles.

Une réservation contient :
- un identifiant
- une salle
- une date
- une heure de début
- une heure de fin
- un utilisateur
- le site créateur

La ressource critique est définie par : Salle|Date
Exemple : SalleA|2026-04-22
Ce choix permet :
- une exclusion mutuelle plus fine
- plusieurs réservations parallèles sur différentes salles
- une file mutex par ressource
- une meilleure concurrence qu'une file globale unique
Exemple :
- SalleA|2026-04-22
- SalleB|2026-04-22
Ces deux ressources étant différentes, deux sites peuvent réserver simultanément sans blocage mutuel.

## 4. Architecture globale
Le système est composé de quatre exécutables indépendants, communiquant par flux texte (stdin/stdout/FIFO) :

```
Navigateur ⇄ WebSocket ⇄ APP ⇄ (stdio) ⇄ CTL ⇄ (FIFO) ⇄ NET ⇄ (FIFO) ⇄ ROUTER ⇄ NET (autres sites)
```

Rôle des composants :
- **web** : interface utilisateur HTML + WebSocket
- **app** (`cmd/app`) : transforme les commandes utilisateur en messages structurés et relaie les événements distribués vers le navigateur
- **ctl** (`cmd/ctl`) : le contrôleur distribué, il gère toute la logique répartie liée aux réservations :
  - exclusion mutuelle
  - horloges
  - snapshots
  - propagation des messages
  - cohérence des réplicats
- **net** (`cmd/net`) : le contrôleur réseau, il gère la composition dynamique du groupe :
  - admission d'un nouveau site (élection sur anneau)
  - départ volontaire et reconfiguration de l'anneau
  - détection de panne par heartbeat (PING/PONG)
  - diffusion des changements de vue vers CTL
- **router** (`cmd/router`) : une couche de transport minimale qui simule le réseau ; il route les messages entre les FIFOs des sites via une table dynamique construite à l'enregistrement (`REGISTER`), sans participer aux algorithmes répartis eux-mêmes

## 5. Architecture du projet

```
planning-reparti/
├── cmd/
│   ├── app/        # application + interface web + websocket
│   ├── ctl/        # contrôleur distribué (mutex, horloges, snapshots)
│   ├── net/        # contrôleur réseau (anneau, admission, panne)
│   └── router/     # routeur minimal (simulation du réseau)
├── internal/
│   ├── clock/      # horloges logiques et vectorielles
│   ├── comm/       # messages structurés
│   ├── mutex/      # exclusion mutuelle répartie
│   ├── net/        # état local d'un site NET (membres, topologie, admission)
│   ├── planning/   # logique métier du planning
│   ├── snapshot/   # état des snapshots répartis
│   ├── state_site/ # état global d'un site CTL
│   └── utils/      # logs
├── scripts/
│   ├── ctl-net.sh          # lance la pile complète (router + net + ctl + app) pour 3 sites
│   ├── add_site.sh         # ajoute dynamiquement un site au groupe
│   ├── crash_site.sh       # simule l'arrêt brutal d'un site
│   ├── reconfigure.sh      # démo de reconfiguration de l'anneau (NET seul)
│   ├── run-ring.sh         # anneau CTL seul, sans NET/APP
│   ├── net-start.sh        # anneau NET seul, sans CTL/APP
│   ├── net-conflict-add.sh # admissions concurrentes (à lancer après net-start.sh)
│   ├── test.sh              # variante alternative APP+CTL seuls (sans NET/router)
│   ├── test-normal.sh      # scénario de réservation normale
│   ├── test-conflict.sh    # scénario de réservations en conflit
│   └── test-noconflict.sh  # scénario de réservations sans conflit
├── go.mod
├── go.sum
└── README.md
```

## 6. Architecture événementielle
Le contrôleur fonctionne selon une architecture événementielle.
Une goroutine dédiée lit les messages entrants puis les transmet dans un canal partagé nommé events.
La boucle principale du contrôleur traite ensuite les événements un par un grâce à un select.
Cela garantit :
- un traitement séquentiel
- l'absence de concurrence sur l'état local
- l'atomicité des modifications
- la cohérence des horloges et du planning
Chaque événement est traité par :
handleEvent()
Cette fonction centralise :
- REQUEST
- ACK
- UPDATE
- RELEASE
- SNAPSHOT_LOCAL
- SNAPSHOT_START
- STATE
- RESERVE_LOCAL
- VIEW_CHANGE

## 7. Packages et exécutables principaux

### internal/comm
Ce paquet gère les messages structurés sous forme de paires clés-valeur
exemple d'un message : /=type=REQUEST/=site=A/=hlg=1/=res=SalleA|2026-04-22

### internal/planning
Ce paquet contient la logique métier :
- Création d'une réservation
- Ajout d'un planning
- Détection de conflits entre 2 réservations
- Clé de ressource salle|date

### internal/clock
Ce paquet contient :
- Horloge logique
- Estampille (hlg,site)
- Horloge vectorielle

### internal/mutex
Ce paquet contient l'exclusion mutuelle répartie :
- REQUEST
- ACK
- RELEASE
- file d'attente par ressource
- ACK différés

### internal/snapshot
Ce paquet contient la logique snapshot réparti :
- snapshots identifiés par ID
- capture des messages reçus pendant un snapshot en cours
- collecte des états locaux par l'initiateur

### internal/net
Ce paquet contient l'état local d'un site NET : membres connus, topologie de l'anneau (prédécesseur/successeur), état d'admission en cours, date du dernier contact de chaque membre.

### cmd/app
Application simple qui transforme une commande utilisateur en message structuré
exp : reserve R1 SalleA 2026-04-22 10:00 11:00 USERNAME devient
/=type=RESERVE_LOCAL/=id=R1/=room=SalleA/=date=2026-04-22/=start=10:00/=end=11:00/=user=USERNAME

### cmd/ctl
Contrôleur distribué où il gère :
- les horloges
- mutex
- file d'attente
- ACK
- UPDATE
- RELEASE
- snapshot

### cmd/net
Contrôleur réseau où il gère :
- l'admission d'un nouveau site par élection sur l'anneau
- le départ volontaire et la reconfiguration de l'anneau
- la détection de panne par heartbeat (PING/PONG)
- la diffusion des changements de vue (MEMBER_LIST) vers CTL

### cmd/router
Routeur minimal qui simule le réseau : chaque site NET s'enregistre auprès du routeur au démarrage (`REGISTER`), qui construit une table dynamique site → FIFO d'entrée, puis route chaque message vers son destinataire (`to=X`) ou le diffuse à tous (`to=ALL`).

## 8. Compilation
A la racine du projet : go build ./...

## 9. Exclusion mutuelle répartie

### Fonctionnement normal
Lorsqu'un site veut réserver une ressource :
1. il stocke la réservation localement
2. il ajoute sa requête dans sa file mutex
3. il diffuse un REQUEST
4. les autres sites répondent par ACK
5. le site attend tous les ACK
6. il vérifie qu'il est premier dans la file
7. il entre en section critique
8. il applique la réservation
9. il diffuse UPDATE
10. il envoie RELEASE
Cela garantit qu'un seul site modifie une ressource donnée à un instant donné.

### Gestion des conflits
Deux sites peuvent demander simultanément : SalleA|2026-04-22

Pour départager les requêtes concurrentes, nous utilisons une estampille : (hlg, site)
Exemple : (2,A) < (2,B)

La priorité est donnée :
1. à la plus petite horloge logique
2. puis au nom du site en cas d'égalité

Si un site reçoit une requête prioritaire :
- il répond immédiatement ACK

Sinon :
- il différé son ACK
- il stocke la demande
- il termine sa section critique
- il envoie ensuite les ACK différés

Cela garantit :
- l'exclusion mutuelle
- l'absence d'interblocage
- l'équité entre les sites

## 10. Horloges

### Horloge logique
Chaque site possède une horloge logique de Lamport.
Elle est incrémentée :
- à chaque événement local
- à la réception d'un message
Lorsqu'un message est reçu :
Clock = max(local, reçu) + 1
Elle permet de construire une estampille : (hlg, site)
Exemple : (3,A)
Cette estampille permet :
- d'ordonner les événements
- de départager deux requêtes concurrentes

### Horloge vectorielle
Chaque site maintient également une horloge vectorielle.

Exemple : [A:5,B:2,C:1]
Chaque case représente la connaissance du site sur les événements des autres sites.

Lors de la réception d'un vecteur :

Vect_local[i] = max(Vect_local[i], Vect_reçu[i])
Puis le site incrémente son propre compteur.

Les horloges vectorielles sont utilisées pour :
- les snapshots répartis
- le suivi de causalité
- l'observation de l'état global distribué

## 11. Snapshot réparti
Le système implémente un snapshot réparti simplifié inspiré de l'algorithme de Chandy-Lamport.

Le snapshot est déclenché par : SNAPSHOT_LOCAL

Le site initiateur :

- sauvegarde son état local (horloge, vecteur, planning)
- diffuse un message SNAPSHOT_START à tous les participants

Les autres sites :

- reçoivent le SNAPSHOT_START
- sauvegardent leur état local
- renvoient un message STATE à l'initiateur
- propagent le SNAPSHOT_START

Pendant le snapshot :
- les messages applicatifs reçus après la sauvegarde locale sont capturés (`CaptureMessage`)

Une fois tous les états locaux collectés par l'initiateur, la configuration globale est affichée et sauvegardée en JSON dans le dossier `snapshots/`.

Cela permet d'obtenir :
- une vue cohérente de l'état global distribué
- un état du système sans arrêter l'exécution
En revanche, dans cette version, nous ne reconstituons pas encore complètement l'état global chez l'initiateur avec collecte des messages prépost (messages applicatifs en transit au moment exact du snapshot).

## 12. Gestion dynamique de la composition du groupe
Le contrôleur NET permet d'ajouter, de retirer ou de faire disparaître un site sans redémarrer les autres.

### Admission d'un nouveau site
Un site candidat envoie un `JOIN_REQUEST` à un membre déjà présent. Celui-ci lance une élection (`ELECTION`, identifiant `ADM-hôte-nouveauSite`) qui circule dans l'anneau afin qu'une seule admission soit validée à la fois — l'identifiant le plus petit l'emporte en cas de conflit. Une fois l'élection revenue à son initiateur, le nouveau site reçoit un `JOIN_ACCEPT` et s'insère dans l'anneau (`RECONFIGURE_ADD`), puis la nouvelle composition est diffusée (`MEMBER_LIST`).

### Départ volontaire
Le site sortant diffuse un `LEAVE_REQUEST` contenant ses voisins ; ceux-ci se reconnectent directement (`MEMBER_REMOVED`) et une nouvelle vue est diffusée.

### Détection de panne
Chaque site envoie un `PING` à chaque membre connu toutes les 5 secondes et attend un `PONG`. Si un membre reste silencieux plus de 30 secondes, il est suspecté défaillant ; seul son prédécesseur dans l'anneau diffuse alors le `MEMBER_REMOVED`, ce qui évite les suppressions concurrentes.

## 13. Interface web et WebSocket
Une interface web minimale a été développée.

Elle permet :
- de saisir une réservation
- de déclencher un snapshot
- de visualiser les logs
- de communiquer avec l'application via WebSocket

Le navigateur communique avec app grâce à : ws://localhost:4444/ws
Les commandes utilisateur sont transformées en messages structurés puis envoyées vers ctl.

## 14. Scripts et procédure de lancement

Le projet est prévu pour être lancé sous un shell Unix (Bash avec `mkfifo`, `pkill`, `lsof`). Sous Windows, il faut passer par WSL ou un environnement Unix équivalent (voir section Limites).

### Lancement de la pile complète (recommandé)
```
./scripts/ctl-net.sh
```
Ce script lance, dans l'ordre : le routeur, 3 sites NET (A/B/C, anneau pré-configuré), 3 serveurs web APP (ports 4444/4445/4446), 3 contrôleurs CTL. Interfaces accessibles sur `http://127.0.0.1:4444|4445|4446`.

### Ajout dynamique d'un site
```
./scripts/add_site.sh <SITE> <SITE_CIBLE> <PORT>
```
Déclenche l'admission par élection dans l'anneau NET, avec sa propre interface web sur le port indiqué.

### Autres scripts de démonstration
- `crash_site.sh <SITE>` : tue brutalement un site (test de détection de panne).
- `reconfigure.sh` : démonstration de reconfiguration de l'anneau côté NET seul.
- `run-ring.sh` : anneau CTL seul, sans NET/APP (utile pour tester uniquement l'exclusion mutuelle/les snapshots).
- `net-start.sh` : anneau NET seul, sans CTL/APP (utile pour tester l'admission/le heartbeat isolément).
- `net-conflict-add.sh` : à lancer après `net-start.sh`, simule deux admissions concurrentes.
- `test.sh` : variante alternative APP+CTL seuls, sans NET ni router.

## 15. Tests principaux
Dans un terminal 1, on lance le script shell `./scripts/run-ring.sh` (pour tester uniquement CTL) ou `./scripts/ctl-net.sh` (pour la pile complète), et dans un autre terminal, on lance un de ces tests pour valider les différents cas de l'application.

### Cas normal
    ./scripts/test-normal.sh
- REQUEST
- ACK
- UPDATE
- RELEASE

### Cas concurrent
    ./scripts/test-conflict.sh
- conflits
- ACK différés
- priorité par estampille

### Cas sans conflit
    ./scripts/test-noconflict.sh
- réservations sur des ressources différentes, traitées en parallèle sans blocage

### Snapshot réparti
    Aprés avoir effectuer une réservation validée on peut lancer un SNAPSHOT avec cette commande :
    echo "/=type=SNAPSHOT_LOCAL/=snap=S1" > /tmp/in_A
- diffusion SNAPSHOT_START
- capture messages
- sauvegarde état local

### Communication Web → app → ctl

Ce test permet de vérifier la chaîne de communication entre l'interface web, l'application et le contrôleur.
Commande :
go run ./cmd/app -n A -port 4444 | go run ./cmd/ctl -n A -sites 3

Puis ouvrir le navigateur :  http://localhost:4444/

Dans l'interface web, cliquer sur Connecter, puis envoyer une commande:
reserve R1 SalleA 2026-04-22 10:00 11:00 AMINE (en cliquant sur Envoyer la réservation)

Résultat attendu :
- l'interface web envoie la commande à app via WebSocket
- app convertit la commande utilisateur en message structuré
- ctl reçoit un message RESERVE_LOCAL
- ctl émet ensuite un REQUEST.

Ce test valide :

- la communication WebSocket
- la génération de messages structurés
- la liaison app → ctl.

RQ : dans cette version, le retour complet ctl → app → web n'est pas encore entièrement implémenté. Les messages distribués REQUEST, ACK, UPDATE, RELEASE sont principalement observés dans les logs du contrôleur.

## 16. Utilisation de l'intelligence artificielle

Des outils d'intelligence artificielle générative ont été utilisés comme assistance ponctuelle pendant le développement.

L'IA a principalement servi pour :

- obtenir des explications techniques sur Go, notamment sur les `struct`, les `map`, les pointeurs, les méthodes et l'utilisation des packages
- comprendre l'intérêt de certains choix de structures de données, par exemple l'utilisation de `map` pour stocker les ACK reçus, les messages déjà traités ou les snapshots réalisés
- aider au débogage d'erreurs Go, par exemple des erreurs de syntaxe, d'imports, de types incompatibles ou de compilation
- aider à comprendre et corriger certains problèmes liés aux channels, goroutines et à la boucle 'select'
- proposer des exemples de scripts Shell pour lancer plusieurs sites avec des FIFOs et automatiser les tests
- aider au développement de l'interface web, notamment pour certaines fonctions JavaScript, la gestion des événements HTML et la compréhension des WebSocket.

L'utilisation de l'IA a surtout permis d'accélérer la compréhension de Go et de certaines parties JavaScript/WebSocket, qui étaient des technologies nouvelles pour nous.

## 17. Outils et technologies utilisés

- **Go 1.26.2** : langage principal du projet
- **github.com/gorilla/websocket** : bibliothèque WebSocket utilisée par `cmd/app` pour la communication avec le navigateur
- **Bash** : scripts de lancement et d'orchestration (`scripts/`), FIFOs (`mkfifo`)
- **Git** : gestion de version, organisation par branches
- **PlantUML** : génération des diagrammes et schémas du rapport
- **ChatGPT** : assistance ponctuelle au développement (voir section précédente)

## 18. Organisation du travail

Le projet a été développé en utilisant Git avec une organisation par branches.

Chaque membre travaillait sur une branche dédiée afin de développer ses fonctionnalités de manière autonome avant intégration dans le projet principal.

### Tâches communes
- Architecture générale du projet
- Tests distribués
- Démonstrations
- Débogage
- Rédaction du rapport

## 19. Limites et améliorations possibles

Limites actuelles :

- WEB → APP → CTL fonctionne complètement. Le flux inverse CTL → APP → WEB reste partiellement implémenté. Les événements distribués sont principalement visualisés dans les logs terminal du contrôleur.
- communication simulée par FIFOs, absence de vrai réseau TCP
- **dépendance à un environnement Unix pour les scripts de lancement** (`mkfifo`, `pkill`, `lsof`) : ils ne fonctionnent pas nativement sous Windows (les FIFOs créées par `mkfifo` sous Git Bash/MSYS ne sont pas interopérables avec des binaires Go natifs) — il faut passer par WSL ou un environnement Unix équivalent
- snapshots simplifiés : pas de collecte des messages prépost en transit au moment exact du snapshot
- interface graphique minimale
- **pas de tests automatisés Go** (`*_test.go`) : la validation repose uniquement sur les scripts shell de `scripts/` et l'inspection des logs
- **gestion de panne partielle côté CTL** : NET détecte les pannes réseau (heartbeat), mais CTL n'a pas de mécanisme équivalent — s'il attend des ACK d'un site tombé en panne pendant une réservation, il reste bloqué indéfiniment (pas de timeout sur l'attente des ACK), et le nombre de sites attendu (`NumSites`) n'est pas recalculé si la composition du groupe change pendant une réservation en cours

Améliorations possibles :

- communication réseau réelle
- interface temps réel avancée
- persistance des snapshots : collecter les messages prépost en plus des états locaux
- visualisation graphique des horloges et files mutex
- une amélioration future serait de faire remonter tous les événements distribués : REQUEST, ACK, UPDATE, RELEASE, SNAPSHOT_START vers l'application puis l'interface Web afin d'obtenir une visualisation temps réel complète du système distribué
- ajouter un timeout sur l'attente des ACK côté CTL pour tolérer la panne d'un site pendant une réservation en cours
- ajouter des tests unitaires Go sur les packages `internal/` (mutex, clock, planning), qui sont facilement testables indépendamment des scripts d'intégration

## 20. Conclusion

Ce projet met en œuvre plusieurs concepts fondamentaux des systèmes répartis :

- exclusion mutuelle répartie
- cohérence distribuée
- horloges logiques
- horloges vectorielles
- snapshots répartis
- gestion dynamique de la composition d'un groupe
- architecture événementielle

L'application permet de simuler un système distribué cohérent tout en illustrant les mécanismes classiques utilisés dans les systèmes répartis modernes.
