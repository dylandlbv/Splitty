# Splitty

Splitty gère le budget familial : chacun déclare ses revenus mensuels, chacun enregistre ce qu'il paie, et l'application calcule qui doit rembourser qui — au prorata de ce que chacun rapporte.

## Fonctionnalités

- **Comptes utilisateurs** : inscription / connexion (JWT, mots de passe hachés avec bcrypt).
- **Multi-foyers** : chaque foyer a ses membres, ses revenus et ses dépenses. Le créateur d'un foyer est administrateur et invite les autres membres par email.
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
| Backend    | Go 1.27 (net/http, pgx v5, JWT, bcrypt)       |
| Frontend   | React 19 + TypeScript + Vite + React Router   |
| Base       | PostgreSQL 17 (migrations SQL embarquées)     |
| Déploiement| Docker Compose (db + api + nginx)             |

```
backend/
├── cmd/splitty/           # point d'entrée
├── internal/
│   ├── api/               # handlers HTTP + middleware auth
│   ├── auth/              # JWT, bcrypt, génération d'ID
│   ├── domain/            # types + moteur de calcul des soldes (testé)
│   └── store/             # accès PostgreSQL + migrations embarquées
└── internal/store/migrations/

frontend/
└── src/
    ├── api.ts             # client API typé
    ├── auth.tsx           # contexte d'authentification
    ├── pages/             # Login, Households, Household
    └── styles.css
```

## Lancer en production (Docker)

Une seule image contient tout le front (nginx) et l'API (Go) — la seule dépendance externe est PostgreSQL :

```bash
export JWT_SECRET="une-longue-chaine-aleatoire"
export POSTGRES_PASSWORD="mot-de-passe-postgres"   # optionnel, "splitty" par défaut
docker compose up --build
```

L'application est disponible sur `http://localhost:8080` : nginx sert les fichiers statiques et proxifie `/api/` vers le binaire Go (ports 80 → 8080 de l'hôte, modifiable via `SPLITTY_PORT`).

## Développement local

1. PostgreSQL local :

   ```bash
   docker run -d --name splitty-db -p 5432:5432 \
     -e POSTGRES_USER=splitty -e POSTGRES_PASSWORD=splitty -e POSTGRES_DB=splitty \
     postgres:17-alpine
   ```

2. Backend :

   ```bash
   cd backend
   export DATABASE_URL="postgres://splitty:splitty@localhost:5432/splitty?sslmode=disable"
   export JWT_SECRET="dev-secret"          # requis
   export ALLOWED_ORIGIN="http://localhost:5173"  # CORS pour le front en dev
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

Deux workflows GitHub Actions (`.github/workflows/`) :

- **CI** — à chaque push sur `main` et à chaque PR : gofmt/build/vet/test du backend, `npm ci` + build (tsc + Vite) du frontend.
- **Docker** — à chaque push sur `main` et chaque tag git `v*` : build et publication de l'image unique `ghcr.io/dylandlbv/splitty` contenant nginx + API + frontend. Tags publiés : `latest` + `sha-<commit>` sur `main` ; `X.Y.Z` / `X.Y` / `X` / `latest` sur un tag git `vX.Y.Z`. Le workflow utilise le `GITHUB_TOKEN` intégré ; aucune configuration de secret n'est nécessaire. Pour sortir une version : `git tag v0.1.0 && git push origin v0.1.0`.

Le `Dockerfile` à la racine construit dans l'ordre : frontend (Node) → backend (Go) → image finale nginx avec le binaire, les statiques et `docker-entrypoint.sh` (lance l'API en fond + nginx en foreground ; si l'API meurt, le conteneur s'arrête).

## Tests

```bash
cd backend
go test ./...      # moteur de calcul des soldes : parts, arrondis, transferts
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
