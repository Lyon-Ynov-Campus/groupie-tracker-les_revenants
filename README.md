# Groupie Tracker – Blind Test & Petit Bac

Ce dépôt contient deux jeux multijoueurs pensés pour le web :

* **Blind Test** – Un quiz musical en ligne. Les joueurs rejoignent une salle, écoutent des extraits et répondent le plus vite possible pour marquer des points.
* **Petit Bac** – Le jeu des catégories repensé pour un salon en ligne avec configuration des catégories, du temps par manche et du nombre de rounds. Les joueurs soumettent leurs réponses simultanément, les votes valident les propositions et un tableau de scores se met à jour en direct via WebSocket.

Les deux jeux partagent la même authentification (création de compte, login, logout) et la même base SQLite localisée à la racine (`blindtest.db`). Les salons se gèrent côté back-end en Go tandis que les assets statiques (HTML/CSS/JS) sont servis depuis le dossier `web` pour l’accueil général et depuis `BlindTest` / `PetitBac` pour chaque jeu.

## Prérequis

1. **Go** version 1.20 ou plus récent (le projet cible Go 1.24).
2. **Git** pour cloner le dépôt.
3. Aucun service externe n’est requis : la base SQLite est créée automatiquement au lancement.

## Installation & Lancement

```bash
git clone https://github.com/<votre-utilisateur>/groupie-tracker-les_revenants.git
cd groupie-tracker-les_revenants
go mod download        # Récupère les dépendances Go (websocket, sqlite, etc.)
go run .               # Compile et lance le serveur HTTP sur le port 8080
```

Une fois le serveur démarré, ouvrez `http://localhost:8080` dans un navigateur moderne. L’accueil permet de créer un compte, de se connecter, puis de choisir entre **Blind Test** ou **Petit Bac**.

## Aperçu des jeux

### Blind Test

* Interface dédiée dans `BlindTest/` avec WebSocket pour mettre à jour les résultats en direct.
* Chaque salon peut accueillir plusieurs joueurs ; la bande-son et les réponses se synchronisent via le serveur Go.
* Les scores des manches sont centralisés dans la base SQLite (`blindtest.db`).

### Petit Bac

* Le flux de création de salon est découpé en étapes : choix des catégories, réglage du temps et du nombre de manches, puis génération d’un code de salon.  
* Un salon peut être rejoint via un code ; la salle d’attente affiche en temps réel les joueurs, leurs scores cumulés et propose un bouton de lancement pour l’hôte.
* Le jeu en lui-même (formulaire, timer, validations) est alimenté par WebSocket (`/ws`).  
* Les données de configuration et les joueurs connectés sont persistés via `linkDatabase.go`, ce qui permet de retrouver un salon et ses scores même après rechargement.

## Structure du projet

```
.
├── BlindTest/          # Code Go, templates et static du jeu Blind Test
├── PetitBac/           # Code Go, templates et static du jeu Petit Bac
├── web/                # Landing page + assets globaux (authentification)
├── main.go             # Point d’entrée : routes, middleware, lancement serveur
├── auth.go             # Gestion comptes, sessions, cookies
├── database.go         # Initialisation base SQLite partagée
└── go.mod/go.sum       # Dépendances Go
```

## Développement

* Lancez `go run .` pendant vos modifications ; le serveur recharge automatiquement les assets statiques.  
* Utilisez `go test ./...` si vous ajoutez des tests métier.  
* Pour mettre à jour les dépendances, utilisez `go get` puis `go mod tidy`.

## Dépannage

* **Erreur SQLite / CGO** – Sur Windows, assurez-vous que `modernc.org/sqlite` est bien téléchargé (`go mod tidy`). Cette implémentation n’exige pas CGO et fonctionne sans configuration supplémentaire.
* **Port 8080 occupé** – Changez l’appel `http.ListenAndServe` dans `main.go` (ex. `:3000`) si un autre service utilise déjà ce port.
* **Mise à jour des assets** – Après modification de CSS/JS, pensez à vider le cache navigateur (Ctrl+Shift+R) pour charger la dernière version.

## Aller plus loin

* Ajoutez vos propres catégories Petit Bac en modifiant `PetitBac/templates/ptitbac_create_categories.html` et en complétant la logique de validation dans `PetitBac/handlers.go`.
* Étendez Blind Test en ajoutant de nouvelles playlists/sons côté serveur et en adaptant la logique des rondes (`BlindTest/scoring.go`).
* Sécurisez davantage en activant HTTPS derrière un reverse proxy (Nginx, Caddy, etc.) si vous déployez sur un serveur public.

Bon jeu !
