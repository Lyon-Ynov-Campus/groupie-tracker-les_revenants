# Groupie Tracker – Petit Bac & Blind Test

Ce dépôt regroupe deux mini‑jeux web temps réel écrits en Go : une version multijoueur du Petit Bac et un Blind Test musical. Le serveur unique (`main.go`) expose les deux expériences sur `http://localhost:8080` avec authentification basique, WebSocket et stockage SQLite.

## Prérequis

- Go ≥ 1.24 (ou la version indiquée dans `go.mod`)
- Accès internet (pour récupérer les modules Go et les extraits Deezer utilisés par le Blind Test)
- SQLite (le binaire embarqué via `github.com/mattn/go-sqlite3` crée automatiquement `blindtest.db`)

## Initialiser le projet Go

1. Placez‑vous à la racine du projet.
2. Créez (ou recréez) le module Go à partir du nom attendu par le code :
   ```bash
   go mod init groupie-tracker
   ```
   Ce fichier existe déjà dans le dépôt ; la commande est utile uniquement si vous repartez d’un dossier vide.
3. Récupérez toutes les dépendances nécessaires (Gorilla WebSocket, Google UUID, SQLite, x/crypto, …) et nettoyez les imports avec :
   ```bash
   go mod tidy
   ```
4. Si votre proxy Go est restreint, installez manuellement les modules externes avant de relancer `go mod tidy` :
   ```bash
   go get github.com/gorilla/websocket@v1.5.3 \
          github.com/google/uuid@v1.6.0 \
          github.com/mattn/go-sqlite3@v1.14.32 \
          golang.org/x/crypto@v0.46.0
   ```

À ce stade `go.sum` est à jour et `go run .` peut compiler tout le serveur.

## Lancer le serveur

```bash
go run .
```

Une fois le serveur démarré :

- Accès général / authentification : `http://localhost:8080`, `/login`, `/register`
- Jeu Petit Bac : `http://localhost:8080/PetitBac` (WebSocket `ws://localhost:8080/ws`)
- Jeu Blind Test : `http://localhost:8080/BlindTest` (WebSocket `ws://localhost:8080/blindtest/ws`)

La base `blindtest.db` est créée automatiquement au premier lancement.

## Organisation des jeux

### Petit Bac (`PetitBac/…`)

- `PetitBac/templates/ptitbac.html` contient la page complète du jeu (UI, timer, table de scores, dialogues).
- `PetitBac/Pstatic/styles.css` regroupe le style spécifique (responsive, états de manche, animations).
- La logique temps réel vit côté serveur dans `main.go` et `logic.go` : génération des lettres (`lettreAleatoire`), catégories (`listeCategories`), calculs de scores collectifs et synchronisation via WebSocket `/ws`.
- Les joueurs peuvent ajuster les catégories, la durée d’une manche et le nombre total de manches (`/config`). Chaque manche démarre automatiquement, les réponses sont contrôlées côté serveur (`reponseValide`) et les scores collectifs récompensent les réponses uniques.

### Blind Test (`BlindTest/…`)

- `BlindTest/game.go` encapsule la totalité du backend : gestion des rooms (`Room`), des joueurs, synchronisation via WebSocket `/blindtest/ws`, appels Deezer pour récupérer des extraits audio (`fetchTracksFromPlaylist`, `fetchTracksFromGenre`) et attribution des points en fonction de la rapidité.
- `BlindTest/static/index.html`, `style.css` et `app.js` fournissent l’interface du jeu (création/connexion à une room, lecture audio, saisie des réponses, chat en direct).
- `RegisterRoutes` est appelé depuis `main.go` pour protéger la page du Blind Test par le middleware d’authentification et exposer les fichiers statiques sous `/blindtest/static/`.

Les deux jeux partagent la couche d’authentification (`auth.go`), l’initialisation SQLite (`database.go`) et le serveur HTTP (handlers dans `main.go`). Chaque dossier (`PetitBac` et `BlindTest`) contient uniquement les ressources spécifiques à son gameplay, ce qui permet de maintenir ou déployer l’un sans impacter l’autre.
