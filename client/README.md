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

## Run the bundled client on Linux

This mode requires GTK4, WebKitGTK 6.0, and CGO. From the repository root:

```sh
just build-client
./client/better-music -bundled -server https://music.example.com
```

To build without `just`:

```sh
npm --prefix frontend ci
npm --prefix frontend run build -- --mode wails
mkdir -p client/ui
rsync -a --delete frontend/build/ client/ui/
cd client
go build -tags bundle -o better-music .
./better-music -bundled -server https://music.example.com
```

Replace the example URL with the HTTPS origin of your deployed Better Music instance. The app serves its embedded UI at `https://localhost:43179` and forwards `/api/` to that deployment. The listener binds to loopback, and the webview trusts the generated certificate for localhost for that launch. Leave port 43179 available so the webview can find the same local library and session on every launch.

Sign in once while online to populate the bundled client's separate local library. Then try making an edit with the Pi unreachable, quitting and relaunching, and confirming the edit syncs when the Pi is reachable again. The bundled profile is tied to the first deployment it connects to. Export any pending edits before changing profiles.

## Install on Arch Linux or CachyOS

Run `just package-client-arch` from the repository root on x86-64 Arch Linux or CachyOS. The build uses `makepkg` and the tools in `base-devel`. Install the generated package:

```sh
sudo pacman -U client/packages/better-music-0.0.1-1-x86_64.pkg.tar.zst
```

Open **Better Music** from your application menu or run `better-music` in a terminal. The launcher defaults to the project's Pi deployment. To use a different Better Music deployment, set `BETTER_MUSIC_SERVER` to its HTTPS origin before the first launch.

## Verify hosted mode

Sign in with a test account. Check that the library loads, then quit and reopen the app to check session persistence. Offline startup in hosted mode depends on the site's service worker and the webview's cache.

To build a Linux executable from `client/`, run `go build -o /tmp/better-music-wails .` and launch it with the same `-server` argument.

## Install the hosted iPhone client with SideStore

On GitHub, open **Actions → Better Music iOS IPA → Run workflow**. After the build succeeds, download the **Better-Music-iOS** artifact and extract `Better-Music.ipa` from it.

With LocalDevVPN connected, import the IPA into SideStore on the iPhone. SideStore signs it with your Apple Account. Switch back to Tailscale before opening Better Music so the app can reach the Pi.

The IPA opens the hosted Better Music site. Verify sign-in, navigation, and scrolling on the iPhone.
