//go:build !bundle || !cgo || (!ios && (!linux || android || gtk3))

package main

import (
	"errors"
	"net/url"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func runBundled(_ *application.App, _ application.WebviewWindowOptions, _ *url.URL) error {
	return errors.New("bundled mode requires an iOS or Linux GTK4 build with CGO and the bundle tag; see client/README.md")
}
