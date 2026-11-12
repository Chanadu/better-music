package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {

	fmt.Println("test")
	server := flag.String("server", "http://localhost:5173", "Better Music server URL")

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
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Better Music",
		URL:    target.String(),
		Width:  1100,
		Height: 760,
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
