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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:ui
var frontend embed.FS

const (
	bundleHost   = "localhost:43179"
	bundleOrigin = "https://" + bundleHost
)

func runBundled(app *application.App, options application.WebviewWindowOptions, upstream *url.URL) error {
	if upstream.Scheme != "https" || (upstream.Path != "" && upstream.Path != "/") ||
		upstream.RawQuery != "" || upstream.Fragment != "" || upstream.ForceQuery {
		return errors.New("bundled mode requires -server to be the HTTPS origin of your Better Music deployment")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:43179")
	if err != nil {
		return fmt.Errorf("start local Better Music listener: %w", err)
	}
	defer listener.Close()

	config, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	profile := filepath.Join(config, "better-music", "bundled")
	if err := os.MkdirAll(profile, 0700); err != nil {
		return err
	}
	// Cookies and IndexedDB share the local origin across launches. Bind it to
	// one deployment so an accidental server change cannot forward its cookies.
	serverFile := filepath.Join(profile, "server")
	serverOrigin := "https://" + strings.ToLower(upstream.Host)
	stored, err := os.ReadFile(serverFile)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := os.WriteFile(serverFile, []byte(serverOrigin), 0600); err != nil {
			return err
		}
	case err != nil:
		return err
	case string(stored) != serverOrigin:
		return fmt.Errorf("bundled mode is already configured for %s", stored)
	}

	cert, certPEM, err := localhostCertificate()
	if err != nil {
		return err
	}
	handler, err := bundleHandler(upstream)
	if err != nil {
		return err
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		TLSConfig:         &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	defer server.Close()
	go func() {
		if err := server.ServeTLS(listener, "", ""); !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "Better Music local server: %v\n", err)
			app.Quit()
		}
	}()

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

func bundleHandler(upstream *url.URL) (http.Handler, error) {
	assets, err := fs.Sub(frontend, "ui")
	if err != nil {
		return nil, err
	}
	fallback, err := fs.ReadFile(assets, "200.html")
	if err != nil {
		return nil, err
	}
	target := *upstream
	target.Path = ""
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(&target)
		},
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 10 * time.Second
	proxy.Transport = transport

	static := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != bundleHost {
			http.Error(w, "Unexpected host", http.StatusBadRequest)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			// A different local port is a different origin and must not be
			// able to use this proxy with the app's localhost cookies.
			origin := r.Header.Get("Origin")
			if (origin != "" && origin != bundleOrigin) ||
				(r.Method != http.MethodGet && r.Method != http.MethodHead && origin != bundleOrigin) {
				http.Error(w, "Unexpected origin", http.StatusForbidden)
				return
			}
			fetchSite := r.Header.Get("Sec-Fetch-Site")
			if fetchSite != "" && fetchSite != "same-origin" {
				http.Error(w, "Unexpected request site", http.StatusForbidden)
				return
			}
			proxy.ServeHTTP(w, r)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if info, err := fs.Stat(assets, path); err == nil && !info.IsDir() {
			static.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(path, "_app/") || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodGet {
			_, _ = w.Write(fallback)
		}
	}), nil
}

func localhostCertificate() (tls.Certificate, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	now := time.Now()
	template := &x509.Certificate{
		DNSNames:    []string{"localhost"},
		NotBefore:   now.Add(-time.Minute),
		NotAfter:    now.AddDate(1, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key},
		string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), nil
}
