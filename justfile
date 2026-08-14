# Start both development servers.
[parallel]
dev: backend frontend

# Start the Go backend development server.
backend:
    cd backend && go run .

# Start the Astro frontend development server.
frontend:
    cd frontend && npm run dev

# Regenerate the backend OpenAPI document and frontend API types.
api:
    cd backend && go run github.com/swaggo/swag/cmd/swag@v1.16.6 init
    cd frontend && npm run api:types

# Verify that frontend API types match the checked-in OpenAPI document.
api-check:
    cd frontend && npm run api:types:check
