# GWI Favourites Service

A small Go service that lets users save their favourite assets (charts, insights, audiences). It talks to PostgreSQL for storage and is meant to be run with Docker Compose.

## How it works

There are two main concepts:

- **Assets** — these are platform entities like Charts, Insights, and Audiences. They're seeded into an in-memory store when the app starts up. Think of them as the "catalog" of things a user can favourite.
- **Favourites** — a user picks an asset they like and optionally adds a personal description. When you fetch a user's favourites, each one comes back enriched with its full asset data.

## API

Every request needs a JWT token in the `Authorization: Bearer <token>` header. The user ID is pulled from the token's `sub` claim — there's no user ID in the URL.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/favourites` | Get all favourites for the authenticated user |
| `POST` | `/api/v1/favourites` | Add a new favourite |
| `PUT` | `/api/v1/favourites/{asset_id}` | Update a favourite's description |
| `DELETE` | `/api/v1/favourites/{asset_id}` | Remove a favourite |
| `GET` | `/health/ready` | Health check (served on a separate port) |

Here's what the request/response bodies look like:

**Adding a favourite (POST):**
```json
{ "asset_id": "chart-1", "description": "My favourite chart" }
```

**Updating a description (PUT):**
```json
{ "description": "Updated description" }
```

**Listing favourites (GET) — returns something like:**
```json
[
  {
    "asset_id": "chart-1",
    "type": "chart",
    "description": "My favourite chart",
    "data": { "title": "Social Media Usage", "x_axis": "Age Group", "y_axis": "Hours", "points": [...] }
  }
]
```

There's also a full OpenAPI spec in `api/swagger.yaml` if you want the complete picture.

## Configuration

The app reads port settings from `config.yaml` and/or environment variables (env vars win if both are set). Database and JWT settings only come from environment variables.

| Setting | Env Variable | Config File Key | Default |
|---------|-------------|-----------------|---------|
| API port | `API_PORT` | `api_port` | `8000` |
| Health port | `HEALTH_PORT` | `health_port` | `8001` |
| DB host | `POSTGRES_HOST` | — | `postgres` (in Compose) |
| DB port | `POSTGRES_PORT` | — | `5432` |
| DB user | `POSTGRES_USER` | — | — |
| DB password | `POSTGRES_PASSWORD` | — | — |
| DB name | `POSTGRES_DB` | — | — |
| JWT secret | `JWT_SECRET` | — | empty (accepts unsigned tokens) |

You can point to a different config file by setting the `CONFIG_PATH` env var.

## Getting it running

The service needs PostgreSQL, so Docker Compose is the way to go.

**1. Set up your `.env` file:**

```bash
cp .env.example .env
```

Open `.env` and fill in at least `POSTGRES_USER` and `POSTGRES_PASSWORD`.

**2. Start everything:**

```bash
docker compose up --build
```

This builds the Go binary, spins up Postgres, waits for it to be healthy, and then starts the app. Add `-d` to run it in the background.

**3. When you're done:**

```bash
docker compose down
```

If you want a clean slate (wipe the database too):

```bash
docker compose down --volumes
```

## Testing

### Unit tests

```bash
go test ./...
```

These use an in-memory repository, so no database needed.

### End-to-end tests

The E2E tests hit the real running services, so you need Compose up first:

```bash
docker compose up --build
```

Once both containers are healthy, open another terminal:

```bash
cd e2e_tests
go clean -testcache
go test -v -count=1
```

The tests automatically wait up to 60 seconds for the health endpoint before running, so you don't need to time it perfectly.

## Authentication

The API uses JWTs. There's a handy little tool included to generate tokens:

```bash
go run ./tools/tokengen -user alice
```

Use the token with curl like this:

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8000/api/v1/favourites
```

By default (when `JWT_SECRET` isn't set), the service accepts unsigned tokens (`alg=none`), which keeps local development simple. For production you'd set `JWT_SECRET` to enforce HS256-signed tokens.

## Storage

Favourites are stored in PostgreSQL. The table uses a composite primary key `(user_id, asset_id)` and keeps the polymorphic asset data in a `jsonb` column. The schema creates itself on startup (`CREATE TABLE IF NOT EXISTS`), so there's no migration step.

A few things I'd consider for production:

- **Caching** — something like Redis for hot user favourite lists.
- **Scaling** — partitioning the favourites table by user ID hash for horizontal scaling.
