# Better Music

Better Music is a full-stack music tracker for managing artists, album queues, and listened albums in a way that still works when your connection does not.

It combines an Astro frontend, a Go backend, PostgreSQL persistence, JWT auth, Spotify-assisted metadata search, and an offline-first local cache with queued background sync.

## Highlights

- Email/password authentication with JWT access tokens and rotating refresh tokens
- Artist library management with create, edit, delete, search, filters, and sorting
- Album queue management with create, edit, delete, search, filters, and sorting
- Separate listened view with ratings, comments, listened dates, and quick listened updates
- Spotify-assisted artist and album search to prefill metadata and cover art
- Offline-first library snapshot stored in the browser
- Queued offline mutations for create, update, and delete operations
- Background sync retry when the device reconnects
- Installable PWA with service worker caching and offline fallback pages
- Mobile-focused app shell with bottom tab navigation and swipe navigation between core pages
- Theme toggle, settings menu, and custom/system dropdown mode support
- Swagger docs for the backend API

## Feature Overview

### Authentication

- Register and sign in with email and password
- Automatic session restore on app launch
- Access token refresh using stored refresh tokens
- Logout from the current device
- Route guarding for authenticated app pages

### Artists

- Create artists manually or from Spotify search
- Store artist name, cover art, and Spotify ID
- Edit and delete artists
- Search artists by name and related album metadata
- Filter artists by whether they have albums
- Sort by alphabetical order, creation date, or average rating
- See pending sync state when changes were made offline
- Refresh the page from the database without using Spotify

### Albums

- Create albums manually or from Spotify search
- Link albums to artists
- Store title, cover art, year, Spotify ID, listened state, rating, comment, and listened date
- Edit and delete albums
- Search albums by title, artist, year, note, and rating-adjacent metadata
- Filter by artist
- Sort queue and listened pages independently
- Mark albums as listened with a quick update flow
- Refresh album pages from the database without using Spotify

### Listened View

- Dedicated listened page separate from the unlistened queue
- Ratings with decimal support
- Optional comments and listened date tracking
- Drag-to-rerank behavior for listened albums
- Alternate default sort behavior for completed listening history

### Offline and PWA

- App pages and shell assets cached by the service worker
- Offline fallback route for disconnected sessions
- Library snapshot persisted in `localStorage`
- Optimistic local create, update, and delete mutations
- Queued sync with retry on reconnect
- Background Sync registration when available
- Faster cached navigation between core pages
- Installable standalone app experience

### UI and UX

- Sticky app header and bottom navigation
- Touch swipe navigation between queue, listened, and artists pages
- Create menu for quick entry points
- Page skeletons and loading overlays
- Light/dark theme toggle
- Custom styled dropdowns with a fallback system-menu mode

## Tech Stack

### Frontend

- [Astro](https://astro.build/)
- Tailwind CSS v4
- `vite-plugin-pwa`
- `sortablejs`

### Backend

- Go 1.26
- Standard `net/http` server
- PostgreSQL
- `golang-migrate`
- `golang-jwt/jwt`
- `bcrypt`
- Swagger via `swaggo/http-swagger`

## Repository Layout

```text
.
├── backend/   Go API, auth, models, migrations, Swagger docs
├── frontend/  Astro app, PWA, offline cache logic, UI
├── deploy/    Caddy config, systemd service, deploy script
└── README.md
```

## Local Development

### Prerequisites

- Node.js and npm
- Go 1.26+
- PostgreSQL

### 1. Clone the repository

```bash
git clone https://github.com/Chanadu/better-music.git
cd better-music
```

### 2. Configure the backend

Create `backend/.env`:

```env
LOG_ENABLE=true
LOG_DEBUG=true
LOG_DIR=./logs

POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_URL=postgres://postgres:postgres@localhost:5432/better_music

SERVER_HOST=localhost
SERVER_PORT=8080

SPOTIFY_CLIENT_ID=your_spotify_client_id
SPOTIFY_CLIENT_SECRET=your_spotify_client_secret

JWT_SECRET=replace-this-with-a-long-random-secret
JWT_ACCESS_TOKEN_MINUTES=15
JWT_REFRESH_TOKEN_HOURS=720
```

Notes:

- Create the log directory before starting the API if you keep `LOG_DIR=./logs`:

```bash
mkdir -p backend/logs
```

- `POSTGRES_URL` is what the backend actually uses to connect and migrate.
- Spotify search now reads `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` from the backend environment only.
- In debug mode the backend appends `?sslmode=disable` to the database URL.
- The backend runs migrations automatically on startup.

### 3. Configure the frontend

No frontend Spotify credentials are required. The browser uses backend `/api/spotify/*` proxy routes.

### 4. Start the backend

```bash
cd backend
go run .
```

The API starts on `http://localhost:8080`.

Swagger docs are available at:

```text
http://localhost:8080/swagger/
```

### 5. Start the frontend

In a second terminal:

```bash
cd frontend
npm install
npm run dev
```

The frontend runs on Astro's dev server and proxies `/api` requests to `http://localhost:8080`.

## Available Commands

### Frontend

```bash
cd frontend
npm install
npm run dev
npm run build
npm run preview
```

### Backend

```bash
cd backend
go run .
go build .
```

## API Overview

Public auth endpoints:

- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `POST /api/auth/logout`

Protected resource endpoints:

- `GET /api/artists`
- `GET /api/artists/{id}`
- `POST /api/artists`
- `PUT /api/artists/{id}`
- `DELETE /api/artists/{id}`
- `GET /api/artists/{id}/albums`
- `GET /api/albums`
- `GET /api/albums/{id}`
- `POST /api/albums`
- `PUT /api/albums/{id}`
- `DELETE /api/albums/{id}`

## Deployment

The `deploy/` directory contains a simple Raspberry Pi-style deployment setup:

- `deploy/Caddyfile` serves the built frontend and reverse proxies `/api/*` and `/swagger/*` to the backend
- `deploy/better-music-backend.service` defines a systemd unit for the Go server
- `deploy/deploy.sh` fetches the latest `main`, rebuilds backend and frontend, and reloads services

Current deployment shape:

- Backend: `127.0.0.1:8080`
- Caddy/static frontend: `:8081`

## Offline Sync Model

Better Music is intentionally not just "cached pages". The app keeps a real local working copy of your library:

- The latest library snapshot is stored in the browser
- New artists and albums can be created offline with temporary local IDs
- Updates and deletes are queued while offline
- When the app reconnects, queued mutations are replayed against the API
- Synced IDs are remapped from temporary local entities to real database IDs
- Pending changes remain visible in the UI until sync completes

## Current State

This repository already includes:

- working frontend and backend applications
- database migrations
- offline-first local cache and mutation queueing
- installable PWA support
- deployment scaffolding

## License

No license file is currently included in this repository.
