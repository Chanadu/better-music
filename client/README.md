# Better Music Wails client

The first Wails client opens a running Better Music instance in a native window. It uses the deployment's HTTPS origin for both the UI and the API.

## Run on Linux

Install the [Wails v3 Linux dependencies](https://wails.io/quick-start/installation/) and run `wails3 doctor`.

From the repository root, replace the example URL with your Better Music deployment:

```sh
cd client
go run . -server https://music.example.com
```

For local development, start the backend and frontend with `just dev` at the repository root. Then run `go run .` from `client/` to open `http://localhost:5173`.

## Verify the webview

Sign in with a test account. Check that the library loads, then quit and reopen the app to check session persistence. After the library has loaded online, try an offline edit and check that it syncs when you reconnect. Offline startup currently depends on the hosted site's service worker and the webview's cache.

To build a Linux executable from `client/`, run `go build -o /tmp/better-music-wails .` and launch it with the same `-server` argument.
