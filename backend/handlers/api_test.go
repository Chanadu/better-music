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
	contentType := recorder.Header().Get("Content-Type")
	if (contentType != "application/json" && contentType != "application/problem+json") || !json.Valid(recorder.Body.Bytes()) {
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
	mux := http.NewServeMux()
	handler.RegisterAPI(mux)
	recorder := request(handler, mux.ServeHTTP, "GET", "/api/account", "", account.Identity, nil)
	status(test, recorder, 200)
	if decode[AccountResponse](test, recorder).Email != "a@example.com" || strings.Contains(recorder.Body.String(), "password") {
		test.Fatal(recorder.Body)
	}
}
func TestUnauthorized(test *testing.T) {
	handler := NewHandler(&config.Config{}, nil)
	mux := http.NewServeMux()
	handler.RegisterAPI(mux)
	for _, path := range []string{"/api/account", "/api/spotify/artists/abc", "/api/spotify/albums/abc", "/api/spotify/search/artists?q=x", "/api/spotify/search/albums?q=x"} {
		status(test, request(handler, mux.ServeHTTP, "GET", path, "", "", nil), 401)
	}

	for _, route := range []struct{ method, path string }{
		{http.MethodPut, "/api/account/email"},
		{http.MethodPut, "/api/account/password"},
		{http.MethodDelete, "/api/account"},
	} {
		status(test, request(handler, mux.ServeHTTP, route.method, route.path, "{}", "", nil), 401)
	}

	// The route's own error uses Huma's problem details format.
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/account", nil))
	if recorder.Code != http.StatusUnauthorized || recorder.Header().Get("Content-Type") != "application/problem+json" {
		test.Fatalf("unexpected Huma error: %d %s", recorder.Code, recorder.Body)
	}
	problem := decode[struct {
		Detail string `json:"detail"`
	}](test, recorder)
	if problem.Detail != "unauthorized" {
		test.Fatalf("unexpected Huma error detail: %q", problem.Detail)
	}
	for _, path := range []string{"/api/sync/artists/pull", "/api/sync/albums/push"} {
		status(test, request(handler, mux.ServeHTTP, http.MethodPost, path, "{}", "", nil), 401)
	}
}
