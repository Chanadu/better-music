# Start both development servers.
[parallel]
dev: backend frontend

# Start the Go backend development server.
backend:
    cd backend && go run .

# Start the Astro frontend development server.
frontend:
    cd frontend && npm run dev
