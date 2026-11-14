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

## Install the iPhone client with SideStore

On GitHub, open **Actions → Better Music iOS IPA → Run workflow**. After the build succeeds, download the **Better-Music-iOS** artifact and extract `Better-Music.ipa` from it.

With LocalDevVPN connected, import the IPA into SideStore on the iPhone. SideStore signs it with your Apple Account. Disconnect LocalDevVPN after installation.

The IPA embeds the Better Music frontend and forwards API requests to the Pi. After upgrading from the hosted IPA, sign in while online so the bundled client can populate its local library. Its local HTTPS origin has separate webview storage. Sync any pending edits in the hosted version before upgrading.

### Update with Shortcuts

Successful builds on `main` publish the IPA to the [iOS preview release](https://github.com/Chanadu/better-music/releases/tag/ios-preview). Once the first build publishes it, create a shortcut named **Update Better Music** with these actions:

1. Connect **LocalDevVPN** using **Set VPN**.
2. Add **Wait** and set it to five seconds.
3. Add **Open URLs** with this URL:

   ```text
   sidestore://install?url=https%3A%2F%2Fgithub.com%2FChanadu%2Fbetter-music%2Freleases%2Fdownload%2Fios-preview%2FBetter-Music.ipa
   ```

Run the shortcut while on Wi-Fi and follow any prompts in SideStore. Keep LocalDevVPN connected until installation finishes. The daily refresh shortcut renews the installed app's signature.
