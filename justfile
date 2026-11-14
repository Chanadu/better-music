# Start both development servers.
[parallel]
dev: backend frontend

# Start the Go backend development server.
backend:
    cd backend && go run .

# Start the SvelteKit frontend development server.
frontend:
    cd frontend && npm run dev

# Build the bundled Linux client.
build-client:
    npm --prefix frontend ci
    npm --prefix frontend run build -- --mode wails
    mkdir -p client/ui
    rsync -a --delete frontend/build/ client/ui/
    cd client && go build -tags bundle -o better-music .

# Package the bundled client for Arch Linux and CachyOS.
package-client-arch: build-client
    bash deploy/arch/build.sh
