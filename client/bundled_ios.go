//go:build bundle && cgo

package main

/*
void pinBetterMusicLoopbackCertificate(const void *der, unsigned long length);
*/
import "C"

import (
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func runBundledWindow(app *application.App, options application.WebviewWindowOptions, _ string, certificateDER []byte) error {
	C.pinBetterMusicLoopbackCertificate(unsafe.Pointer(&certificateDER[0]), C.ulong(len(certificateDER)))
	options.URL = bundleOrigin + "/"
	app.Window.NewWithOptions(options)
	return app.Run()
}
