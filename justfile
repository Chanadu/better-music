# Start both development servers.
[parallel]
dev: backend frontend

# Start the Go backend development server.
backend:
    cd backend && go run .

# Start the SvelteKit frontend development server.
frontend:
    cd frontend && npm run dev

# Run the iPhone live preview through the laptop's Tailscale IPv4 address.
live-preview:
    #!/usr/bin/env bash
    set -euo pipefail
    setsid npm --prefix frontend run dev -- --host 127.0.0.1 --port 5193 --strictPort &
    vite_pid=$!
    forward_pid=""
    trap 'kill -- -"$vite_pid" 2>/dev/null || true; if [ -n "$forward_pid" ]; then kill "$forward_pid" 2>/dev/null || true; fi; wait || true' EXIT
    trap 'exit 130' INT TERM
    ready=false
    for attempt in {1..30}; do
        kill -0 "$vite_pid"
        if curl --noproxy '*' --silent --fail http://127.0.0.1:5193/preview/redesign >/dev/null; then
            ready=true
            break
        fi
        sleep 1
    done
    if [ "$ready" != true ]; then
        echo "Vite did not start on port 5193." >&2
        exit 1
    fi
    tailscale serve --tcp=5194 tcp://127.0.0.1:5193 &
    forward_pid=$!
    wait -n "$vite_pid" "$forward_pid"

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
