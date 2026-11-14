//go:build bundle && linux && !android && !gtk3 && cgo

package main

/*
#cgo pkg-config: webkitgtk-6.0
#include <webkit/webkit.h>
#include <stdlib.h>

static void configure_webkit(const char *certificate, const char *cookie_path) {
	WebKitNetworkSession *session = webkit_network_session_get_default();
	GTlsCertificate *cert = g_tls_certificate_new_from_pem(certificate, -1, NULL);
	webkit_network_session_allow_tls_certificate_for_host(session, cert, "localhost");
	g_object_unref(cert);
	webkit_cookie_manager_set_persistent_storage(
		webkit_network_session_get_cookie_manager(session), cookie_path,
		WEBKIT_COOKIE_PERSISTENT_STORAGE_SQLITE);
}
*/
import "C"

import (
	"encoding/pem"
	"path/filepath"
	"sync"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func runBundledWindow(app *application.App, options application.WebviewWindowOptions, profile string, certificateDER []byte) error {
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}))

	options.URL = "about:blank"
	window := app.Window.NewWithOptions(options)
	var configure sync.Once
	window.OnWindowEvent(events.Linux.WindowLoadFinished, func(_ *application.WindowEvent) {
		go configure.Do(func() {
			application.InvokeSync(func() {
				certificate := C.CString(certPEM)
				defer C.free(unsafe.Pointer(certificate))
				cookiePath := C.CString(filepath.Join(profile, "cookies.sqlite"))
				defer C.free(unsafe.Pointer(cookiePath))
				C.configure_webkit(certificate, cookiePath)
			})
			window.SetURL(bundleOrigin + "/")
		})
	})
	return app.Run()
}
