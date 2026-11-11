package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/internal/testdb"
	"github.com/Chanadu/better-music/middleware"
)

func testHandler(test *testing.T) *Handler {
	test.Helper()
	return NewHandler(&config.Config{}, testdb.Open(test))
}

type testAccount struct {
	UserID   string
	Identity string
}

func createAccount(test *testing.T, handler *Handler, email string) testAccount {
	test.Helper()
	if handler.Database == nil {
		handler.Database = testdb.Open(test)
	}
	account := testAccount{Identity: "fixture-" + email}
	if err := handler.Database.QueryRow(`INSERT INTO users(email,id) VALUES($1,$2) RETURNING id`, email, account.Identity).Scan(&account.UserID); err != nil {
		test.Fatal(err)
	}
	return account
}

func serveIdentity(handler *Handler, endpoint http.Handler, writer http.ResponseWriter, httpRequest *http.Request, identity string) {
	middleware.ResolveAuthIdentity(endpoint, handler.Database, func(*http.Request) string { return identity }).ServeHTTP(writer, httpRequest)
}

func request(handler *Handler, endpoint http.HandlerFunc, method, path, body, identity string, headers map[string]string) *httptest.ResponseRecorder {
	httpRequest := httptest.NewRequest(method, path, strings.NewReader(body))
	parts := strings.Split(strings.Split(path, "?")[0], "/")
	if len(parts) > 3 {
		httpRequest.SetPathValue("id", parts[3])
	}
	for name, value := range headers {
		httpRequest.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	serveIdentity(handler, endpoint, recorder, httpRequest, identity)
	return recorder
}
func status(test *testing.T, recorder *httptest.ResponseRecorder, want int) {
	test.Helper()
	if recorder.Code != want {
		test.Fatalf("status %d, want %d: %s", recorder.Code, want, recorder.Body)
	}
	if recorder.Header().Get("Content-Type") != "application/json" || !json.Valid(recorder.Body.Bytes()) {
		test.Fatalf("invalid JSON response: %s", recorder.Body)
	}
}
func decode[T any](test *testing.T, recorder *httptest.ResponseRecorder) T {
	test.Helper()
	var result T
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		test.Fatal(err)
	}
	return result
}

func TestAccountRead(test *testing.T) {
	handler := testHandler(test)
	account := createAccount(test, handler, "a@example.com")
	recorder := request(handler, handler.GetAccount, "GET", "/api/account", "", account.Identity, nil)
	status(test, recorder, 200)
	if decode[AccountResponse](test, recorder).Email != "a@example.com" || strings.Contains(recorder.Body.String(), "password") {
		test.Fatal(recorder.Body)
	}
}
func TestUnauthorized(test *testing.T) {
	handler := NewHandler(&config.Config{}, nil)
	for _, endpoint := range []http.HandlerFunc{handler.GetAccount, handler.ChangeSuperTokensAccount, handler.PullLibrary, handler.PushLibrary, handler.SearchSpotifyArtists, handler.SearchSpotifyAlbums, handler.GetSpotifyArtist, handler.GetSpotifyAlbum} {
		status(test, request(handler, endpoint, "GET", "/", "{}", "", nil), 401)
	}
}
