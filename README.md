# Splitty

Splitty gère le budget familial : chacun déclare ses revenus mensuels, chacun enregistre ce qu'il paie, et l'application calcule qui doit rembourser qui — au prorata de ce que chacun rapporte.

## Fonctionnalités

- **Comptes utilisateurs** : inscription / connexion (JWT, mots de passe hachés avec bcrypt).
- **Multi-foyers** : chaque foyer a ses membres, ses revenus et ses dépenses. Le créateur d'un foyer est administrateur, invite les autres membres par email et peut aussi créer des **membres sans compte** (sans login) dont il gère les revenus et les dépenses comme les siens.
- **Revenus mensuels** : chaque membre déclare son revenu pour chaque mois.
- **Dépenses personnalisées** : nom, montant, payeur, dépense commune ou personnelle, répartition **égalitaire** ou **proportionnelle** aux revenus déclarés.
- **Soldes mensuels** : pour chaque membre
( payé, part due, solde)
; et la liste des remboursements suggérés (*qui doit quoi à qui*).

## Règles de calcul

Seules les dépenses **communes** entrent dans le soldes ; les dépenses personnelles sont tracées mais exclues des remboursements.

- **Répartition égalitaire** : le montant est divisé en parts égales entre tous les membres du foyer (arrondi au centime, le reliquat est réparti à titre égal).
- **Répartition proportionnelle** : la part de chacun est proportionnelle à son revenu déclaré du mois : `part = revenu / Σ revenus × montant`.
  - Un membre **sans revenu déclaré** (0 €) ce mois-là est **exclu par défaut** de la répartition proportionnelle.
  - Si aucun membre n'a de revenu déclaré, la dépense retombe sur une répartition **égalitaire**.
  - Les centimes d'arrondi sont redistribués un par un, dans l'ordre des membres.
- **Remboursements** : les débiteurs remboursent les créanciers par la méthode gloutonne (du plus gros débit vers le plus gros crédit), ce qui minimise le nombre de virements à montant total transféré égal.

Tous les montants sont stockés en **centimes** (bigint) pour éviter les erreurs de virgule flottante.

## Stack technique

| Couche     | Techno                                        |
| ---------- | --------------------------------------------- |
| Backend    | Go 1.27 (net/http, database/sql, JWT, bcrypt) |
| Frontend   | React 19 + TypeScript + Vite + React Router   |
| Base       | SQLite (par défaut) ou PostgreSQL 17          |
| Déploiement| Docker Compose (SQLite par défaut ; variante Postgres) |

```
backend/
├── cmd/splitty/           # point d'entrée
├── internal/
│   ├── api/               # handlers HTTP + middleware auth
│   ├── auth/              # JWT, bcrypt, génération d'ID
│   ├── domain/            # types + moteur de calcul des soldes (testé)
│   └── store/             # accès SQLite / PostgreSQL + migrations embarquées
└── internal/store/migrations/

frontend/
└── src/
    ├── api.ts             # client API typé
    ├── auth.tsx           # contexte d'authentification
    ├── pages/             # Login, Households, Household
    └── styles.css
```

## Lancer en production (Docker)

Une seule image contient tout le front (nginx) et l'API (Go). Par défaut, les données sont stockées dans une base **SQLite** — aucun service externe n'est nécessaire :

```bash
export JWT_SECRET="une-longue-chaine-aleatoire"
docker compose up --build
```

La base SQLite vit dans le volume `sqldata` (fichier `/data/splitty.db` dans le conteneur, modifiable via `SQLITE_PATH`).

Pour utiliser **PostgreSQL** plutôt que SQLite :

```bash
export JWT_SECRET="une-longue-chaine-aleatoire"
export POSTGRES_PASSWORD="mot-de-passe-postgres"   # requis
docker compose -f docker-compose.yml -f docker-compose.postgres.yml up --build
```

L'application est disponible sur `http://localhost:8080` : nginx sert les fichiers statiques et proxifie `/api/` vers le binaire Go (ports 80 → 8080 de l'hôte, modifiable via `SPLITTY_PORT`).

## Développement local

SQLite est la base par défaut : aucun service externe n'est nécessaire.

1. Backend :

   ```bash
   cd backend
   export JWT_SECRET="dev-secret"          # requis
   export ALLOWED_ORIGIN="http://localhost:5173"  # CORS pour le front en dev
   export SQLITE_PATH="./data/splitty.db"  # optionnel, ./data/splitty.db par défaut
   go run ./cmd/splitty
   ```

   Pour développer avec **PostgreSQL** à la place :

   ```bash
   docker run -d --name splitty-db -p 5432:5432 \
     -e POSTGRES_USER=splitty -e POSTGRES_PASSWORD=splitty -e POSTGRES_DB=splitty \
     postgres:17-alpine

   export DB_DRIVER=postgres
   export DATABASE_URL="postgres://splitty:splitty@localhost:5432/splitty?sslmode=disable"
   go run ./cmd/splitty
   ```

3. Frontend (dans un autre terminal) :

   ```bash
   cd frontend
   npm install
   npm run dev             # http://localhost:5173, /api est proxifié vers :8080
   ```

Les migrations SQL s'appliquent automatiquement au démarrage du backend.

## CI/CD

Quatre workflows GitHub Actions (`.github/workflows/`) :

- **CI** — à chaque push sur `main` et à chaque PR : gofmt/build/vet/test du backend, `npm ci` + build (tsc + Vite) du frontend.
- **Tests** — pipeline QA à chaque push sur `main` et à chaque PR : lint (gofmt, `go vet`), tests backend avec `-race` et couverture, smoke e2e avec le vrai binaire, suite store rejouée sur un service PostgreSQL 17, et build frontend (tsc + Vite). Voir la section [Tests](#tests) pour les détails et pour rejouer la même pipeline en local.
- **Docker** — à chaque push sur `main` et chaque tag git `v*` : build et publication de l'image unique `ghcr.io/dylandlbv/splitty` contenant nginx + API + frontend. Tags publiés : `latest` + `sha-<commit>` sur `main` ; `X.Y.Z` / `X.Y` / `X` / `latest` sur un tag git `vX.Y.Z`. Le workflow utilise le `GITHUB_TOKEN` intégré ; aucune configuration de secret n'est nécessaire.
- **Release** — à chaque tag git `v*` (et en rattrapage à chaque push sur `main`, ou manuellement via `workflow_dispatch`) : crée la release GitHub associée au tag avec des notes générées automatiquement. Pour sortir une version : `git tag v0.1.0 && git push origin v0.1.0` (publie l'image versionnée et la release GitHub).

Le `Dockerfile` à la racine construit dans l'ordre : frontend (Node) → backend (Go) → image finale nginx avec le binaire, les statiques et `docker-entrypoint.sh` (lance l'API en fond + nginx en foreground ; si l'API meurt, le conteneur s'arrête).

## Tests

Pipeline de tests complète, identique en local et en CI (workflow **Tests**, `.github/workflows/test.yml`) :

```bash
./scripts/test.sh  # gofmt + go vet + go test ./... -race (backend) + npm ci + build (frontend)
```

Niveaux de tests :

- **Unitaires** (`backend/internal/domain`) : moteur de calcul des soldes — parts égales / proportionnelles, arrondis au centime, exclusion des membres sans revenu, repli égalitaire, remboursements gloutons.
- **Store** (`backend/internal/store`) : la suite est rejouée sur SQLite (fichier temporaire, aucune infrastructure) et, si `TEST_DATABASE_URL` est défini, sur PostgreSQL (`TestPostgresSuite`, activé en CI via un service postgres:17).
- **API** (`backend/internal/api`) : scénarios de bout en bout via `httptest` — auth (validation, doublons, mauvais mot de passe, tokens manquants/invalides), foyers et permissions (admin / membre / non-membre), revenus (upsert, validations), dépenses (validations, filtre par mois, suppression), et soldes comparés au moteur de calcul.
- **E2E smoke** (`backend/e2e`) : le vrai binaire est compilé puis lancé avec une base SQLite temporaire ; un scénario complet est joué en HTTP réel (inscription → connexion → foyer → invitation → revenus → dépenses → soldes → suppression → isolement d'un non-membre), et le refus de démarrer sans `JWT_SECRET` est vérifié.

```bash
cd backend
go test ./...                 # tout : unitaires + store + API + e2e
go test ./internal/... -race -coverprofile=coverage.out   # couverture : go tool cover -func=coverage.out
go test ./e2e/... -v          # smoke e2e seul
```

## API

| Méthode | Route | Description |
| --- | --- | --- |
| POST | `/api/auth/register` | `{email, name, password}` → `{token, user}` |
| POST | `/api/auth/login` | `{email, password}` → `{token, user}` |
| GET | `/api/me` | utilisateur courant |
| GET / POST | `/api/households` | lister / créer un foyer |
| GET | `/api/households/{id}` | foyer + membres |
| POST | `/api/households/{id}/members` | inviter un membre par email (admin) |
| GET | `/api/households/{id}/incomes?month=YYYY-MM` | revenus du mois |
| PUT | `/api/households/{id}/incomes/YYYY-MM` | `{incomes: [{user_id, amount_cents}]}` |
| GET / POST | `/api/households/{id}/expenses` | dépenses du mois (`?month=`) / créer |
| DELETE | `/api/households/{id}/expenses/{expenseID}` | supprimer |
| GET | `/api/households/{id}/balances/YYYY-MM` | soldes + remboursements |

Toutes les routes sauf `register` / `login` exigent `Authorization: Bearer <token>`.

## Pistes d'évolution

- Export / archivage des mois précédents, graphiques de répartition.
- Répartition proportionnelle avec quotités personnalisées par dépense.
- Liens d'invitation auto-inscription au lieu de l'invitation par email (utilisateur existant).
- Catégories de dépenses et budgets mensuels.
