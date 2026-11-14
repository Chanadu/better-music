//go:build !bundle || !linux || android || gtk3 || !cgo

package main

import (
	"errors"
	"net/url"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func runBundled(_ *application.App, _ application.WebviewWindowOptions, _ *url.URL) error {
	return errors.New("bundled mode requires a Linux GTK4 build with CGO and the bundle tag; see client/README.md")
}
