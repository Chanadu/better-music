# Better Music

Better Music is a self-hosted web app for tracking albums you want to listen to and albums you have listened to. It organizes artists and albums into a personal library, with ratings, notes, and listening dates.

The app supports Spotify metadata search, manual entry, and offline library editing. Its interface works on desktop and mobile.

[Features](#features) · [Getting started](#getting-started) · [Self-hosting](#self-hosting) · [Development](#development)

## How it works

1. **Add an album.** Search Spotify for its title, artist, release year, and artwork, or enter the details manually.
2. **Choose what to listen to.** Browse your queue or use the home page's random album suggestion. Linked albums can be opened in Spotify.
3. **Record a listen.** Mark the album as listened, give it a rating from 1–10, and save notes and a listening date.
4. **Review your library.** Browse listened albums and artists, search and sort records, and view listening statistics.

## Features

| Feature | Description |
| --- | --- |
| **Listening queue** | Keep the albums you want to hear together, with search, sorting, and a shuffled next-album pick. |
| **Listening history** | Save ratings, notes, and dates alongside your listened albums. See recent listens and additions on the home page. |
| **Artist and album pages** | Browse your collection by artist and open detail pages to explore and edit your library. |
| **Spotify metadata** | Find artists and albums with artwork and release information. Manual entry is always available. |
| **Statistics** | See albums listened, artists explored, average ratings, and a rating distribution for this month, this year, or all time. |
| **Appearance settings** | Choose a theme, customize rating labels and colors, and select native or custom dropdown menus. |
| **Offline editing** | Browse cached records and save edits without a connection. Pending changes sync when you're back online. |
| **JSON export** | Export your artists, albums, ratings, notes, and dates as JSON, including locally saved changes. |
| **Accounts and self-hosting** | Host the app yourself. Each email/password account has its own library and account settings. |

The app includes an installable progressive web app (PWA) for supported browsers. Theme and presentation preferences are saved on your device.

### Pages

**Home** offers your next album and recent activity. **Listen** holds your queue, **Albums** holds your listening history, and **Artists** brings your collection together by artist. **More** contains listening statistics, customization, export, sync controls, and account settings.

Playback happens in Spotify or your preferred music player. The Spotify integration searches metadata; playlist and listening-history imports are not currently supported. Library export is available as JSON; there is no import interface yet.

## Technology

**SvelteKit + Svelte 5 + TypeScript** on the frontend, **Go** for the API, and **PostgreSQL** for the library. Tailwind CSS and daisyUI provide the interface, while a service worker and IndexedDB support offline use.

The backend handles Spotify requests and authentication, including bcrypt passwords, JWT access tokens, and rotating refresh tokens. Swagger documentation and generated TypeScript API types keep the client and server contract in sync.

## Getting started

### Requirements

- Node.js **22.12 or newer** and npm
- Go **1.26.1 or newer**
- A running PostgreSQL server and a database user able to create the application's tables
- Optional: [`just`](https://github.com/casey/just) for the repository's command shortcuts
- Optional: a Spotify application's client ID and client secret for metadata search

### 1. Get the code and create a database

```bash
git clone https://github.com/Chanadu/better-music.git
cd better-music
```

Create a PostgreSQL database for the app, for example `better_music`. Use that database's credentials in the configuration below. The backend applies the migrations in `backend/migrations/` automatically at startup.

### 2. Configure the backend

Create `backend/.env` with your local settings:

```dotenv
LOG_ENABLE=false
LOG_DEBUG=true
LOG_DIR=./logs

POSTGRES_URL=postgres://postgres:postgres@localhost:5432/better_music
SERVER_HOST=localhost
SERVER_PORT=8080

JWT_SECRET=replace-with-a-long-random-secret
JWT_ACCESS_TOKEN_MINUTES=15
JWT_REFRESH_TOKEN_HOURS=720

SPOTIFY_CLIENT_ID=
SPOTIFY_CLIENT_SECRET=
```

Replace the example database credentials and JWT secret. Generate a secret with `openssl rand -hex 32`, for example. Keep credentials out of version control: the current root `.gitignore` does not exclude `.env` files.

You can leave the Spotify credentials blank to use manual entry. Both log flags must be set to `true` or `false`; the example disables file logging and uses a local database connection.

<details>
<summary>Configuration reference and logging options</summary>

| Variable | Behavior |
| --- | --- |
| `POSTGRES_URL` | PostgreSQL connection URL for the application's database. |
| `SERVER_HOST`, `SERVER_PORT` | Address and port the API listens on. Use `localhost` and `8080` for the included Vite proxy. |
| `JWT_SECRET` | Required signing secret for access tokens. |
| `JWT_ACCESS_TOKEN_MINUTES` | Access-token lifetime; defaults to 15 minutes. Must be positive. |
| `JWT_REFRESH_TOKEN_HOURS` | Refresh-token lifetime; defaults to 720 hours (30 days). Must be positive. |
| `LOG_ENABLE` | Required boolean. Enables file logging when `true`; stdout logging remains enabled either way. |
| `LOG_DEBUG` | Required boolean. When `true`, uses `debug.log` and appends `?sslmode=disable` to the database URL. |
| `LOG_DIR` | Directory for log files when file logging is enabled. |
| `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET` | Optional for manual library management; needed for Spotify search and metadata lookups. |

With `LOG_DEBUG=true`, use a database URL without an existing query string. For a URL that already contains options, set `LOG_DEBUG=false` and specify your intended `sslmode` in the URL directly.

If you enable file logging, create its directory before starting the backend:

```bash
mkdir -p backend/logs
```

</details>

### 3. Install dependencies and start the servers

From the repository root:

```bash
cd frontend
npm install
cd ..
just dev
```

Without `just`, run these in separate terminals from the repository root:

```bash
cd backend
go run .
```

```bash
cd frontend
npm run dev
```

Open [Better Music](http://localhost:5173) and create an account. Vite forwards `/api` requests to `http://localhost:8080`. The backend serves [Swagger UI](http://localhost:8080/swagger/index.html).

Run the backend from `backend/`: its `.env` loading and migration paths depend on the working directory.

## Offline and sync

Your library saves locally in the browser, so edits appear immediately and survive a reload. When you're online and signed in, pending changes synchronize with the server. Each account has its own local cache and queue.

Open the production app online and sign in before using it offline. Previously cached pages and artwork can then remain available without a connection. Spotify search and account changes still need internet access.

Under **More → Library sync**, you can check pending work, select **Sync now**, retry failures, and resolve conflicting edits by choosing server values, local values, or individual fields.

<details>
<summary>Browser requirements and keeping unsynced changes</summary>

- Use HTTPS or localhost with IndexedDB, localStorage, Web Locks, and `crypto.randomUUID` available. Synchronization requires Web Locks.
- The production PWA caches the app shell. The Vite development server does not provide the same offline experience, and artwork must be cached before it is available offline.
- Expired authentication preserves pending changes. Sign back into the same account to resume syncing.
- Explicit sign-out removes that account's local cache and pending changes once logout is eligible. Sync changes you want to keep before signing out. Clearing site data can also remove unsynced work.
- Sign-out is blocked while relevant requests are active or their outcomes are unknown.

The sync queue uses idempotency keys to replay uncertain requests without repeating a mutation, record versions to detect conflicts, and Web Locks to coordinate processing across tabs. Conflict resolutions save locally and still need to synchronize.

</details>

## Development

### Repository layout

```text
backend/
  config/       Environment configuration
  handlers/     Authentication, account, library, and Spotify endpoints
  models/       Database records and mutation persistence
  migrations/   PostgreSQL schema migrations
  docs/         Generated Swagger files
frontend/
  src/routes/   SvelteKit pages
  src/lib/      UI components, API client, local storage, and synchronization
  scripts/      API type generator
deploy/         Caddy, systemd, and deployment examples
justfile        Development and API-generation shortcuts
```

### Commands

Run these from `frontend/`:

```bash
npm run dev              # Development server
npm run build            # Static production app in frontend/build/
npm run preview          # Preview the frontend build
npm run check            # Svelte and TypeScript checks
npm run format:check     # Formatting checks
npm run api:types        # Generate types from backend/docs/swagger.json
npm run api:types:check  # Check that generated types are current
```

The API proxy is configured for the development server. To use a production build with the backend, serve both through a reverse proxy as described below.

Run these from `backend/`:

```bash
go run .
go build -o build/better-music-backend .
go test ./...
```

Backend integration tests start a disposable PostgreSQL cluster when `initdb` and `pg_ctl` are on `PATH`. Alternatively, set `TEST_POSTGRES_URL` to a PostgreSQL URL whose user can create test databases. Tests create separate databases rather than migrating the supplied database. Set `REQUIRE_TEST_DB=1` to fail instead of skipping when PostgreSQL binaries are unavailable.

After changing API handler annotations or models, run `just api` from the repository root to regenerate Swagger files and frontend types. Run `just api-check` to check frontend types against the checked-in Swagger contract. Do not edit `frontend/src/lib/scripts/api-types.ts` by hand.

<details>
<summary>API routes</summary>

Registration, login, refresh, and logout use `/api/auth/{register,login,refresh,logout}`. Other API routes require an `Authorization: Bearer <token>` header.

| Resource | Routes |
| --- | --- |
| Account | `GET /api/account`, `PUT /api/account/email`, `PUT /api/account/password`, `DELETE /api/account` |
| Artists | `GET/POST /api/artists`, `GET/PUT/DELETE /api/artists/{id}`, `GET /api/artists/{id}/albums` |
| Albums | `GET/POST /api/albums`, `GET/PUT/DELETE /api/albums/{id}` |
| Spotify | `GET /api/spotify/search/artists`, `GET /api/spotify/search/albums`, `GET /api/spotify/artists/{id}`, `GET /api/spotify/albums/{id}` |

See Swagger UI for request bodies, parameters, response schemas, and mutation version fields.

</details>

## Self-hosting

Build the frontend with `npm run build` in `frontend/` and the backend with `go build -o build/better-music-backend .` in `backend/`. Run the backend with `backend/` as its working directory and provide its environment configuration and database access.

Serve `frontend/build/` with a web server, proxy `/api/*` to the backend, and fall back to `200.html` for client-side routes. Use HTTPS for browser synchronization outside localhost. Proxy `/swagger/*` too if you want to expose API documentation; the Swagger document URL currently uses the host and port configured in `backend/main.go`.

<details>
<summary>Using the included Raspberry Pi deployment examples</summary>

The files in [`deploy/`](deploy/) are Raspberry Pi examples and need adjustment before use:

- Paths and the systemd user assume `/home/pi/better-music` and user `pi`.
- The Caddyfile listens on `:8081`, serves `frontend/build`, and falls back to `200.html` for frontend routes. Configure HTTPS for your hostname.
- The systemd unit runs `go run .`, even though the deployment script builds a binary. Update `ExecStart` to use the built binary if that is how you intend to run the service.
- `deploy.sh` resets the checkout to `origin/main`, installs dependencies, rebuilds, and restarts services. It overwrites local tracked changes.

</details>

Keep regular PostgreSQL backups for the server library. The in-app JSON export is a readable library snapshot, including pending local edits, but it does not include account credentials or a database restore mechanism.

## Roadmap

- [ ] Import from Spotify
- [ ] Spotify shuffle
- [ ] fancy hexagon background pattern
- [ ] List <-> Grid Options for the pages
## License

[MIT](LICENSE)
