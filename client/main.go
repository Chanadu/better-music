package main

import (
	"flag"
	"log"
	"net"
	"net/url"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {

	server := flag.String("server", defaultServerURL, "Better Music server URL")
	inspect := flag.Bool("inspect", false, "Open the webview inspector at startup")
	bundle := flag.Bool("bundled", false, "Serve the embedded frontend and forward API requests to -server")

	flag.Parse()

	target, err := url.Parse(*server)
	if err != nil || target.Hostname() == "" || target.User != nil {
		log.Fatal("provide a valid Better Music URL with -server")
	}
	if target.Scheme != "https" {
		host := target.Hostname()
		if target.Scheme != "http" || (host != "localhost" && !net.ParseIP(host).IsLoopback()) {
			log.Fatal("use HTTPS for a remote Better Music server")
		}
	}

	app := application.New(application.Options{Name: "Better Music"})
	options := application.WebviewWindowOptions{
		Title:                  "Better Music",
		URL:                    target.String(),
		Width:                  1100,
		Height:                 760,
		DevToolsEnabled:        *inspect,
		OpenInspectorOnStartup: *inspect,
	}
	if *bundle {
		if err := runBundled(app, options, target); err != nil {
			log.Fatal(err)
		}
		return
	}
	app.Window.NewWithOptions(options)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
