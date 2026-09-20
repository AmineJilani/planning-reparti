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
- une interface web

## 2. Fonctionnalités principales

- Réservation distribuée de salles
- Exclusion mutuelle répartie
- Gestion des conflits concurrents
- ACK différés
- Horloges logiques de Lamport
- Horloges vectorielles
- Snapshot réparti inspiré de Chandy-Lamport
- Communication en anneau
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
- une meilleure concurrence qu’une file globale unique
Exemple :
- SalleA|2026-04-22
- SalleB|2026-04-22
Ces deux ressources étant différentes, deux sites peuvent réserver simultanément sans blocage mutuel.

## 4.Architecture globale
Le système est séparé en plusieurs couches :
Interface Web
↓
app
↓
ctl
↓
réseau distribué

Rôle des composants :
- web : interface utilisateur HTML + WebSocket
- app : transforme les commandes utilisateur en messages structurés
- ctl :gère toute la logique distribuée :
  - exclusion mutuelle
  - horloges
  - snapshots
  - propagation des messages
  - cohérence des réplicats


## 5. Architecture du projet 
planning-reparti/
├── cmd/
│   ├── app/     # application + interface web + websocket
│   ├── ctl/     # contrôleur distribué
│  
├── internal/
│   ├── clock/      # horloges logiques et vectorielles
│   ├── comm/       # messages structurés
│   ├── mutex/      # exclusion mutuelle répartie
│   ├── planning/   # logique métier du planning
│   ├── snapshot/   # sauvegarde répartie
│   ├── state_site/ # état global d'un site
│   └── utils/      # logs
├── scripts/
│   ├── run-ring.sh
│   ├── test-conflict.sh
│   └── test-normal.sh
|   └── test-noconflict.sh
│
├── go.mod
├── go.sum
└── README.md

## 6. Architecture événementielle
Le contrôleur fonctionne selon une architecture événementielle.
Une goroutine dédiée lit les messages entrants puis les transmet dans un canal partagé nommé events.
La boucle principale du contrôleur traite ensuite les événements un par un grâce à un select.
Cela garantit :
- un traitement séquentiel
- l’absence de concurrence sur l’état local
- l’atomicité des modifications
- la cohérence des horloges et du planning
Chaque événement est traité par :
handleEvent()
Cette fonction centralise :
- REQUEST
- ACK
- UPDATE
- RELEASE
- MARKER
- SNAPSHOT_LOCAL
- RESERVE_LOCAL

## 4. Packages principaux 
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
- messages MARKER
- capture des messages reçus aprés snapshot

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

## 5. Compilation
A la racine du projet : go build ./...

## 6. Exclusion mutuelle répartie

### Fonctionnement normal
Lorsqu’un site veut réserver une ressource :
1. il stocke la réservation localement
2. il ajoute sa requête dans sa file mutex
3. il diffuse un REQUEST
4. les autres sites répondent par ACK
5. le site attend tous les ACK
6. il vérifie qu’il est premier dans la file
7. il entre en section critique
8. il applique la réservation
9. il diffuse UPDATE
10. il envoie RELEASE
Cela garantit qu’un seul site modifie une ressource donnée à un instant donné.

### Gestion des conflits
Deux sites peuvent demander simultanément : SalleA|2026-04-22

Pour départager les requêtes concurrentes, nous utilisons une estampille : (hlg, site)
Exemple : (2,A) < (2,B)

La priorité est donnée :
1. à la plus petite horloge logique
2. puis au nom du site en cas d’égalité

Si un site reçoit une requête prioritaire :
- il répond immédiatement ACK

Sinon :
- il différé son ACK
- il stocke la demande
- il termine sa section critique
- il envoie ensuite les ACK différés

Cela garantit :
- l’exclusion mutuelle
- l’absence d’interblocage
- l’équité entre les sites

## 7. Horloges

### Horloge logique
Chaque site possède une horloge logique de Lamport.
Elle est incrémentée :
- à chaque événement local
- à la réception d’un message
Lorsqu’un message est reçu :
Clock = max(local, reçu) + 1
Elle permet de construire une estampille : (hlg, site)
Exemple : (3,A)
Cette estampille permet :
- d’ordonner les événements
- de départager deux requêtes concurrentes

### Horloge vectorielle
Chaque site maintient également une horloge vectorielle.

Exemple : [A:5,B:2,C:1]
Chaque case représente la connaissance du site sur les événements des autres sites.

Lors de la réception d’un vecteur :

Vect_local[i] = max(Vect_local[i], Vect_reçu[i])
Puis le site incrémente son propre compteur.

Les horloges vectorielles sont utilisées pour :
- les snapshots répartis
- le suivi de causalité
- l’observation de l’état global distribué

## 8. Snapshot réparti
Le système implémente un snapshot réparti simplifiée inspiré de l’algorithme de Chandy-Lamport.

Le snapshot est déclenché par : SNAPSHOT_LOCAL

Le site initiateur :

- sauvegarde son état local
- sauvegarde son planning
- sauvegarde ses horloges
- envoie un MARKER

Les autres sites :

- reçoivent le MARKER
- sauvegardent leur état local
- propagent le MARKER

Pendant le snapshot :
- les messages reçus après sauvegarde sont capturés

Cela permet d’obtenir :
- une vue cohérente de l’état global distribué
- un état du système sans arrêter l’exécution
En revanche, dans cette version, nous ne reconstituons pas encore complètement l’état global chez l’initiateur avec collecte des états locaux et messages prépost.

## 9. Interface web et WebSocket
Une interface web minimale a été développée.

Elle permet :
- de saisir une réservation
- de déclencher un snapshot
- de visualiser les logs
- de communiquer avec l’application via WebSocket

Le navigateur communique avec app grâce à : ws://localhost:4444/ws
Les commandes utilisateur sont transformées en messages structurés puis envoyées vers ctl.

## 10. Scripts de test

Le projet contient plusieurs scripts shell permettant de simuler plusieurs sites.

Exemple : ./scripts/run-ring.sh

Ce script :
- crée des FIFOs : - Chaque site possède un FIFO d’entrée
- lance plusieurs contrôleurs
- stdin représente les messages reçus
- stdout représente les messages envoyés
- tee duplique les messages vers les autres sites
- connecte les sites en anneau
- permet de simuler un réseau distribué

## 11. Tests principaux
Dans un terminal 1, on lance le script shell ./scripts/run-ring.sh
et dans un autre terminal, on lance un de ces tests pour valider les différents cas de l'application.
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

### Snapshot réparti 
    Aprés avoir effectuer une réservation validée on peut lancer un SNAPSHOT avec cette commande : 
    echo "/=type=SNAPSHOT_LOCAL/=snap=S1" > /tmp/in_A
- MARKER
- capture messages
- sauvegarde état local

### Communication Web → app → ctl

Ce test permet de vérifier la chaîne de communication entre l’interface web, l’application et le contrôleur.
Commande :
go run ./cmd/app -n A -port 4444 | go run ./cmd/ctl -n A -sites 3

Puis ouvrir le navigateur :  http://localhost:4444/

Dans l’interface web, cliquer sur Connecter, puis envoyer une commande:
reserve R1 SalleA 2026-04-22 10:00 11:00 AMINE (en cliquant sur Envoyer la réservation)

Résultat attendu :
- l’interface web envoie la commande à app via WebSocket
- app convertit la commande utilisateur en message structuré
- ctl reçoit un message RESERVE_LOCAL
- ctl émet ensuite un REQUEST.

Ce test valide :

- la communication WebSocket
- la génération de messages structurés
- la liaison app → ctl.

RQ : dans cette version, le retour complet ctl → app → web n’est pas encore entièrement implémenté. Les messages distribués REQUEST, ACK, UPDATE, RELEASE sont principalement observés dans les logs du contrôleur.


## 12. Utilisation de l’intelligence artificielle

Des outils d’intelligence artificielle générative ont été utilisés comme assistance ponctuelle pendant le développement.

L’IA a principalement servi pour :

- obtenir des explications techniques sur Go, notamment sur les `struct`, les `map`, les pointeurs, les méthodes et l’utilisation des packages
- comprendre l’intérêt de certains choix de structures de données, par exemple l’utilisation de `map` pour stocker les ACK reçus, les messages déjà traités ou les snapshots réalisés
- aider au débogage d’erreurs Go, par exemple des erreurs de syntaxe, d’imports, de types incompatibles ou de compilation
- aider à comprendre et corriger certains problèmes liés aux channels, goroutines et à la boucle 'select'
- proposer des exemples de scripts Shell pour lancer plusieurs sites avec des FIFOs et automatiser les tests
- aider au développement de l’interface web, notamment pour certaines fonctions JavaScript, la gestion des événements HTML et la compréhension des WebSocket.

L’utilisation de l’IA a surtout permis d’accélérer la compréhension de Go et de certaines parties JavaScript/WebSocket, qui étaient des technologies nouvelles pour nous.

Outils utilisés :
- Go
- PlantUML (pour générer des diagrammes et des schméas clairs)
- ChatGPT
- Bash

## 13. Organisation du travail

Le projet a été développé en utilisant Git avec une organisation par branches.

Chaque membre travaillait sur une branche dédiée afin de développer ses fonctionnalités de manière autonome avant intégration dans le projet principal.

### Répartition des tâches
ASMA KHOULA : 
- Messages structurés 
- Logique métier du planning

VASCO GASPAR : 
- Snapshot réparti
- Gestion de l’état local d’un site 

AMINE JILANI : 
- Exclusion mutuelle, 
- Horloges logiques/vectorielles, 
- Séparation app/ctl, 
- Interface WebSocket,
- Architecture événementielle 

### Tâches communes
- Architecture générale du projet
- Tests distribués
- Démonstrations
- Débogage
- Rédaction du rapport

## 14. Limites et améliorations possibles

Limites actuelles :

- Actuellement, WEB → APP → CTL fonctionne complètement.
Le flux inverse CTL → APP → WEB reste partiellement implémenté.
Les événements distribués sont principalement visualisés dans les logs terminal du contrôleur.
- communication simulée par FIFOs
- absence de vrai réseau TCP
- snapshots simplifiés
- interface graphique minimale

Améliorations possibles :

- communication réseau réelle
- interface temps réel avancée
- persistance des snapshots : collecter les états locaux chez l'initiateur
- visualisation graphique des horloges et files mutex
- Une amélioration future serait de faire remonter tous les événements distribués : REQUEST, ACK, UPDATE, RELEASE, MARKER vers l’application puis l’interface Web afin d’obtenir une visualisation temps réel complète du système distribué.

## 15. Conclusion

Ce projet met en œuvre plusieurs concepts fondamentaux des systèmes répartis :

- exclusion mutuelle répartie
- cohérence distribuée
- horloges logiques
- horloges vectorielles
- snapshots répartis
- architecture événementielle

L’application permet de simuler un système distribué cohérent tout en illustrant les mécanismes classiques utilisés dans les systèmes répartis modernes.


