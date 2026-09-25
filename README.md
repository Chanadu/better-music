# Better Music

Better Music is a self-hosted music tracker for keeping an album queue, recording listening history, and deciding what to play next.

The app pairs a SvelteKit frontend with a Go API and PostgreSQL. Spotify integration can fill in artist, album, release-year, and cover-art metadata, while manual entry remains available for anything Spotify does not have.

## Features

- Email/password accounts with short-lived JWT access tokens and rotating refresh tokens
- Separate views for the album queue, listened albums, and artists
- Create, edit, and delete artists and albums
- Spotify search and metadata refresh through the backend
- Ratings from 1–10, notes, and listened dates
- Search and persistent sorting preferences for album and artist lists
- A home page with a shuffled “next album” pick and recently listened/added albums
- Artist and album detail pages with library statistics
- Per-session library caching with automatic refresh when the app regains focus
- Installable progressive web app (PWA)
- Generated Swagger/OpenAPI documentation and frontend API types
- Responsive, mobile-first interface built with Tailwind CSS and daisyUI

## Tech Stack

### Frontend

- [SvelteKit](https://svelte.dev/docs/kit)
- Svelte 5 and TypeScript
- Vite 7
- Tailwind CSS 4 and daisyUI 5
- `vite-plugin-pwa`

### Backend

- Go 1.26.1
- Standard-library `net/http` server
- PostgreSQL
- `golang-migrate`
- `golang-jwt/jwt` and bcrypt
- Swagger UI via `swaggo/http-swagger`

## Repository Layout

```text
.
├── backend/   Go API, database models, migrations, and Swagger docs
├── frontend/  SvelteKit application and PWA configuration
├── deploy/    Caddy, systemd, and Raspberry Pi deployment files
├── justfile   Development and API-generation commands
└── LICENSE
```

## Local Development

### Prerequisites

- Node.js 22.12 or newer
- npm
- Go 1.26.1 or newer
- PostgreSQL
- [`just`](https://github.com/casey/just) (optional, for the root-level shortcuts)

### 1. Clone the repository

```bash
git clone https://github.com/Chanadu/better-music.git
cd better-music
```

### 2. Create the database

Create a PostgreSQL database named `better_music` (or use another name and update `POSTGRES_URL` below). Database migrations run automatically when the backend starts.

### 3. Configure the backend

Create `backend/.env`:

```env
LOG_ENABLE=true
LOG_DEBUG=true
LOG_DIR=./logs

POSTGRES_URL=postgres://postgres:postgres@localhost:5432/better_music

SERVER_HOST=localhost
SERVER_PORT=8080

SPOTIFY_CLIENT_ID=your_spotify_client_id
SPOTIFY_CLIENT_SECRET=your_spotify_client_secret

JWT_SECRET=replace-this-with-a-long-random-secret
JWT_ACCESS_TOKEN_MINUTES=15
JWT_REFRESH_TOKEN_HOURS=720
```

Create the configured log directory before starting the API:

```bash
mkdir -p backend/logs
```

`JWT_ACCESS_TOKEN_MINUTES` and `JWT_REFRESH_TOKEN_HOURS` are optional and default to 15 minutes and 720 hours. Spotify credentials are needed for Spotify search and metadata refresh; artist and album records can still be entered manually without them.

When `LOG_DEBUG=true`, the backend adds `sslmode=disable` to `POSTGRES_URL` for local PostgreSQL connections. Set `LOG_DEBUG=false` in environments that require SSL.

### 4. Install frontend dependencies

```bash
cd frontend
npm install
cd ..
```

### 5. Run the app

Start both development servers from the repository root:

```bash
just dev
```

Or run them in separate terminals:

```bash
cd backend
go run .
```

```bash
cd frontend
npm run dev
```

The frontend is available at `http://localhost:5173` and proxies `/api` to the backend at `http://localhost:8080`. Swagger UI is available at `http://localhost:8080/swagger/`.

## Commands

Run frontend commands from `frontend/`:

```bash
npm run dev              # Start the Vite development server
npm run build            # Create the static production build
npm run preview          # Preview the production build
npm run check            # Run Svelte and TypeScript checks
npm run format:check     # Check formatting
```

Run backend commands from `backend/`:

```bash
go run .
go build .
go test ./...
```

### API contract generation

The checked-in Swagger document is the source for the frontend API types. After changing API handlers or models, run this from the repository root:

```bash
just api
```

Verify that the generated frontend types are current with:

```bash
just api-check
```

The generated file is `frontend/src/lib/scripts/api-types.ts`; do not edit it by hand.

## API Overview

Authentication endpoints are public:

- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `POST /api/auth/logout`

Artist, album, and Spotify endpoints require an `Authorization: Bearer <token>` header:

- `GET|POST /api/artists`
- `GET|PUT|DELETE /api/artists/{id}`
- `GET /api/artists/{id}/albums`
- `GET|POST /api/albums`
- `GET|PUT|DELETE /api/albums/{id}`
- `GET /api/spotify/search/artists`
- `GET /api/spotify/search/albums`
- `GET /api/spotify/artists/{id}`
- `GET /api/spotify/albums/{id}`

See Swagger UI for request bodies, parameters, and response schemas.

## Data and Caching

PostgreSQL is the source of truth. After login, the frontend fetches the current user's artists and albums and stores that snapshot in `sessionStorage` so navigation within the tab is immediate. It refreshes stale data when the page becomes visible or the window regains focus.

The PWA caches the static application shell. Creating or changing library data still requires a connection to the backend; offline mutation queueing is not currently implemented.

## Deployment

The `deploy/` directory contains a Raspberry Pi-oriented deployment example:

- `Caddyfile` serves the static frontend and proxies `/api/*` and `/swagger/*` to `127.0.0.1:8080`
- `better-music-backend.service` runs the Go backend with systemd
- `deploy.sh` updates `main`, rebuilds both applications, restarts the backend, and reloads Caddy

These files assume the repository is installed at `/home/pi/better-music`. The included Caddyfile still points to `frontend/dist`, while SvelteKit's static adapter outputs `frontend/build` by default; align that path before deploying. Also review the service user, environment configuration, and network exposure before using the example on another host.

## Roadmap

- [ ] Automatically select “listened” when adding a rating
- [ ] Build the More page and add library statistics
- [ ] Add account settings, including password/email changes and account deletion
- [ ] Add a theme selector
- [ ] Add customizable rating labels
- [ ] Add Spotify shuffle shortcuts
- [ ] Add grid/list toggles for artists and albums
- [ ] Add skeleton loading states

## License

Better Music is available under the [MIT License](./LICENSE).
