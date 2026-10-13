package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/internal/testdb"
	"github.com/Chanadu/better-music/middleware"
)

func testHandler(test *testing.T) *Handler {
	test.Helper()

	return NewHandler(&config.Config{
		JWTSecret:  "test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	}, testdb.Open(test))
}

func request(apiHandler *Handler, endpointHandler http.HandlerFunc, method, path, body, token string, headers map[string]string) *httptest.ResponseRecorder {
	httpRequest := httptest.NewRequest(method, path, strings.NewReader(body))
	parts := strings.Split(strings.Split(path, "?")[0], "/")
	if len(parts) > 3 {
		httpRequest.SetPathValue("id", parts[3])
	}

	for headerName, headerValue := range headers {
		httpRequest.Header.Set(headerName, headerValue)
	}

	responseRecorder := httptest.NewRecorder()
	if token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+token)
		middleware.Auth(endpointHandler, apiHandler.Config.JWTSecret).ServeHTTP(responseRecorder, httpRequest)
	} else {
		endpointHandler(responseRecorder, httpRequest)
	}

	return responseRecorder
}

func status(test *testing.T, responseRecorder *httptest.ResponseRecorder, want int) {
	test.Helper()
	if responseRecorder.Code != want {
		test.Fatalf("status %d, want %d: %s", responseRecorder.Code, want, responseRecorder.Body)
	}

	if responseRecorder.Header().Get("Content-Type") != "application/json" || !json.Valid(responseRecorder.Body.Bytes()) {
		test.Fatalf("invalid JSON response: %s", responseRecorder.Body)
	}
}

func decode[T any](test *testing.T, responseRecorder *httptest.ResponseRecorder) T {
	test.Helper()
	var value T
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &value); err != nil {
		test.Fatal(err)
	}

	return value
}

func register(test *testing.T, apiHandler *Handler, email string) TokenResponse {
	test.Helper()
	responseRecorder := request(apiHandler, apiHandler.AuthRegister, "POST", "/api/auth/register", fmt.Sprintf(`{"email":%q,"password":"password"}`, email), "", nil)
	status(test, responseRecorder, 201)

	tokens := decode[TokenResponse](test, responseRecorder)
	if tokens.AccessToken == "" || tokens.RefreshToken == "" || tokens.TokenType != "Bearer" || tokens.ExpiresIn != 900 ||
		tokens.UserID <= 0 {
		test.Fatalf("invalid tokens: %+v", tokens)
	}

	return tokens
}

func TestAuthenticationLifecycle(test *testing.T) {
	apiHandler := testHandler(test)
	tokens := register(test, apiHandler, " USER@Example.com ")
	status(test, request(apiHandler, apiHandler.AuthRegister, "POST", "/", `{"email":"user@example.com","password":"password"}`, "", nil), 409)

	for _, body := range []string{
		`{"email":"user@example.com","password":"wrong"}`,
		`{"email":"missing@example.com","password":"password"}`,
		`{"email":"invalid","password":"password"}`,
	} {
		status(test, request(apiHandler, apiHandler.AuthLogin, "POST", "/", body, "", nil), 401)
	}

	responseRecorder := request(apiHandler, apiHandler.AuthLogin, "POST", "/", `{"email":" USER@example.com ","password":"password"}`, "", nil)
	status(test, responseRecorder, 200)

	login := decode[TokenResponse](test, responseRecorder)
	responseRecorder = request(apiHandler, apiHandler.AuthRefresh, "POST", "/", fmt.Sprintf(`{"refresh_token":%q}`, tokens.RefreshToken), "", nil)
	status(test, responseRecorder, 200)

	rotated := decode[TokenResponse](test, responseRecorder)
	if rotated.RefreshToken == tokens.RefreshToken {
		test.Error("refresh token not rotated")
	}

	status(test, request(apiHandler, apiHandler.AuthRefresh, "POST", "/", fmt.Sprintf(`{"refresh_token":%q}`, tokens.RefreshToken), "", nil), 401)

	for attemptIndex := 0; attemptIndex < 2; attemptIndex++ {
		status(test, request(apiHandler, apiHandler.AuthLogout, "POST", "/", fmt.Sprintf(`{"refresh_token":%q}`, rotated.RefreshToken), "", nil), 200)
	}

	status(test, request(apiHandler, apiHandler.AuthRefresh, "POST", "/", fmt.Sprintf(`{"refresh_token":%q}`, rotated.RefreshToken), "", nil), 401)

	status(test, request(apiHandler, apiHandler.AuthLogout, "POST", "/", `{"refresh_token":"unknown"}`, "", nil), 200)
	if _, err := apiHandler.Database.Exec("UPDATE refresh_tokens SET expires_at=NOW()-INTERVAL '1 hour' WHERE token_hash=$1", hashRefreshToken(login.RefreshToken)); err != nil {
		test.Fatal(err)
	}

	status(test, request(apiHandler, apiHandler.AuthRefresh, "POST", "/", fmt.Sprintf(`{"refresh_token":%q}`, login.RefreshToken), "", nil), 401)
}

func TestAccountLifecycle(test *testing.T) {
	apiHandler := testHandler(test)
	accountTokens := register(test, apiHandler, "a@example.com")
	register(test, apiHandler, "b@example.com")
	responseRecorder := request(apiHandler, apiHandler.GetAccount, "GET", "/api/account", "", accountTokens.AccessToken, nil)
	status(test, responseRecorder, 200)
	if decode[AccountResponse](test, responseRecorder).Email != "a@example.com" ||
		strings.Contains(responseRecorder.Body.String(), "password") {
		test.Fatal(responseRecorder.Body)
	}

	status(
		test,
		request(apiHandler, apiHandler.UpdateAccountEmail, "PUT", "/", `{"email":"new@example.com","password":"wrong"}`, accountTokens.AccessToken, nil),
		401,
	)

	status(
		test,
		request(apiHandler, apiHandler.UpdateAccountEmail, "PUT", "/", `{"email":"b@example.com","password":"password"}`, accountTokens.AccessToken, nil),
		409,
	)

	status(
		test,
		request(apiHandler, apiHandler.UpdateAccountEmail, "PUT", "/", `{"email":" NEW@example.com ","password":"password"}`, accountTokens.AccessToken, nil),
		200,
	)

	status(test, request(apiHandler, apiHandler.AuthRefresh, "POST", "/", fmt.Sprintf(`{"refresh_token":%q}`, accountTokens.RefreshToken), "", nil), 401)

	status(
		test,
		request(apiHandler, apiHandler.UpdateAccountPassword, "PUT", "/", `{"current_password":"wrong","new_password":"updated"}`, accountTokens.AccessToken, nil),
		401,
	)

	status(
		test,
		request(apiHandler, apiHandler.UpdateAccountPassword, "PUT", "/", `{"current_password":"password","new_password":"updated"}`, accountTokens.AccessToken, nil),
		200,
	)

	status(test, request(apiHandler, apiHandler.AuthLogin, "POST", "/", `{"email":"new@example.com","password":"password"}`, "", nil), 401)

	status(test, request(apiHandler, apiHandler.AuthLogin, "POST", "/", `{"email":"new@example.com","password":"updated"}`, "", nil), 200)

	status(
		test,
		request(
			apiHandler,
			apiHandler.CreateArtist,
			"POST",
			"/api/artists",
			`{"name":"Artist"}`,
			accountTokens.AccessToken,
			map[string]string{"Idempotency-Key": "account-delete"},
		),
		201,
	)

	status(test, request(apiHandler, apiHandler.CreateAlbum, "POST", "/api/albums", `{"artist_id":1,"title":"Album"}`, accountTokens.AccessToken, nil), 201)

	status(test, request(apiHandler, apiHandler.DeleteAccount, "DELETE", "/", `{"password":"wrong"}`, accountTokens.AccessToken, nil), 401)

	status(test, request(apiHandler, apiHandler.DeleteAccount, "DELETE", "/", `{"password":"updated"}`, accountTokens.AccessToken, nil), 200)

	for _, table := range []string{
		"users",
		"artists",
		"albums",
		"refresh_tokens",
		"processed_mutations",
	} {
		column := "user_id"
		if table == "users" {
			column = "id"
		}

		var count int
		if err := apiHandler.Database.QueryRow("SELECT count(*) FROM "+table+" WHERE "+column+"=$1", accountTokens.UserID).Scan(&count); err != nil || count != 0 {
			test.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
}

func TestValidationAndUnauthorized(test *testing.T) {
	apiHandler := NewHandler(&config.Config{JWTSecret: "secret"}, nil)

	for _, endpointHandler := range []http.HandlerFunc{
		apiHandler.AuthRegister,
		apiHandler.AuthLogin,
		apiHandler.AuthRefresh,
		apiHandler.AuthLogout,
		apiHandler.UpdateAccountEmail,
		apiHandler.UpdateAccountPassword,
		apiHandler.DeleteAccount,
	} {
		status(test, request(apiHandler, endpointHandler, "POST", "/", "{", "", nil), 400)
	}

	for _, endpointHandler := range []http.HandlerFunc{
		apiHandler.AuthRegister,
		apiHandler.AuthRefresh,
		apiHandler.AuthLogout,
		apiHandler.UpdateAccountEmail,
		apiHandler.UpdateAccountPassword,
		apiHandler.DeleteAccount,
	} {
		status(test, request(apiHandler, endpointHandler, "POST", "/", "{}", "", nil), 400)
	}

	for _, endpointHandler := range []http.HandlerFunc{
		apiHandler.GetAccount,
		apiHandler.GetArtists,
		apiHandler.GetArtist,
		apiHandler.GetArtistAlbums,
		apiHandler.CreateArtist,
		apiHandler.UpdateArtist,
		apiHandler.DeleteArtist,
		apiHandler.GetAlbums,
		apiHandler.GetAlbum,
		apiHandler.CreateAlbum,
		apiHandler.UpdateAlbum,
		apiHandler.DeleteAlbum,
		apiHandler.SearchSpotifyArtists,
		apiHandler.SearchSpotifyAlbums,
		apiHandler.GetSpotifyArtist,
		apiHandler.GetSpotifyAlbum,
	} {
		status(test, request(apiHandler, endpointHandler, "GET", "/", "{}", "", nil), 401)
	}
}

func TestConcurrentRefresh(test *testing.T) {
	apiHandler := testHandler(test)
	accountTokens := register(test, apiHandler, "concurrent@example.com")
	var waitGroup sync.WaitGroup
	codes := make(chan int, 2)

	for attemptIndex := 0; attemptIndex < 2; attemptIndex++ {
		waitGroup.Go(func() {
			codes <- request(apiHandler, apiHandler.AuthRefresh, "POST", "/", fmt.Sprintf(`{"refresh_token":%q}`, accountTokens.RefreshToken), "", nil).Code
		})
	}

	waitGroup.Wait()
	close(codes)
	counts := map[int]int{}

	for code := range codes {
		counts[code]++
	}

	if counts[200] != 1 || counts[401] != 1 {
		test.Fatalf("rotation results: %v", counts)
	}
}
