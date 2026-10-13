package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
)

func TestMutationFailureRollsBack(test *testing.T) {
	apiHandler := testHandler(test)
	accountTokens := register(test, apiHandler, "owner@example.com")
	responseRecorder := request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", `{"name":"Artist"}`, accountTokens.AccessToken, nil)
	status(test, responseRecorder, 201)

	artist := decode[models.Artist](test, responseRecorder)
	body := fmt.Sprintf(`{"artist_id":%d,"title":"Album","listened_at":"not-a-date"}`, artist.ID)
	headers := map[string]string{"Idempotency-Key": "bad-date"}
	responseRecorder = request(apiHandler, apiHandler.CreateAlbum, "POST", "/api/albums", body, accountTokens.AccessToken, headers)
	status(test, responseRecorder, 400)

	replay := request(apiHandler, apiHandler.CreateAlbum, "POST", "/api/albums", body, accountTokens.AccessToken, headers)
	status(test, replay, 400)
	if replay.Body.String() != responseRecorder.Body.String() {
		test.Fatal("database validation error not replayed")
	}

	var count int
	if err := apiHandler.Database.QueryRow("SELECT count(*) FROM albums").Scan(&count); err != nil || count != 0 {
		test.Fatalf("failed insert leaked: count=%d %v", count, err)
	}

	if _, err := apiHandler.Database.Exec(`CREATE FUNCTION reject_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected storage failure'; END $$; CREATE TRIGGER reject_mutation BEFORE INSERT ON processed_mutations FOR EACH ROW EXECUTE FUNCTION reject_mutation()`); err != nil {
		test.Fatal(err)
	}

	status(
		test,
		request(
			apiHandler,
			apiHandler.CreateArtist,
			"POST",
			"/api/artists",
			`{"name":"Rollback"}`,
			accountTokens.AccessToken,
			map[string]string{"Idempotency-Key": "storage-failure"},
		),
		500,
	)
	var exists bool
	if err := apiHandler.Database.QueryRow("SELECT EXISTS(SELECT 1 FROM artists WHERE name='Rollback')").Scan(&exists); err != nil || exists {
		test.Fatalf("mutation not rolled back: %v %v", exists, err)
	}
}

func TestMissingAccountAndDuplicateHeaders(test *testing.T) {
	apiHandler := testHandler(test)
	accountTokens := register(test, apiHandler, "a@example.com")
	httpRequest := httptest.NewRequest("POST", "/api/artists", strings.NewReader(`{"name":"Artist"}`))

	httpRequest.Header.Set("Authorization", "Bearer "+accountTokens.AccessToken)
	httpRequest.Header.Add("Idempotency-Key", "first")
	httpRequest.Header.Add("Idempotency-Key", "second")

	responseRecorder := httptest.NewRecorder()
	middleware.Auth(http.HandlerFunc(apiHandler.CreateArtist), apiHandler.Config.JWTSecret).ServeHTTP(responseRecorder, httpRequest)
	status(test, responseRecorder, 400)

	if _, err := apiHandler.Database.Exec("DELETE FROM users WHERE id=$1", accountTokens.UserID); err != nil {
		test.Fatal(err)
	}

	status(test, request(apiHandler, apiHandler.CreateArtist, "POST", "/api/artists", `{"name":"Artist"}`, accountTokens.AccessToken, nil), 401)
}

func TestSpotifyLookupErrors(test *testing.T) {
	apiHandler := NewHandler(&config.Config{JWTSecret: "secret", AccessTTL: time.Hour}, nil)

	token, err := apiHandler.generateJWT(1)
	if err != nil {
		test.Fatal(err)
	}

	for _, endpointHandler := range []http.HandlerFunc{apiHandler.GetSpotifyArtist, apiHandler.GetSpotifyAlbum} {
		httpRequest := newSpotifyRequest("", token)
		responseRecorder := httptest.NewRecorder()
		middleware.Auth(endpointHandler, apiHandler.Config.JWTSecret).ServeHTTP(responseRecorder, httpRequest)
		status(test, responseRecorder, 400)

		httpRequest = newSpotifyRequest("abc", token)
		responseRecorder = httptest.NewRecorder()
		middleware.Auth(endpointHandler, apiHandler.Config.JWTSecret).ServeHTTP(responseRecorder, httpRequest)
		status(test, responseRecorder, 503)
	}

	apiHandler = spotifyHandler(func(httpRequest *http.Request) (*http.Response, error) {
		if httpRequest.URL.String() == spotifyAuthURL {
			return spotifyResponse(200, `{"access_token":"token","expires_in":3600}`), nil
		}

		return spotifyResponse(404, "missing"), nil
	})
	token, err = apiHandler.generateJWT(1)
	if err != nil {
		test.Fatal(err)
	}

	for _, endpointHandler := range []http.HandlerFunc{apiHandler.GetSpotifyArtist, apiHandler.GetSpotifyAlbum} {
		httpRequest := newSpotifyRequest("abc", token)
		responseRecorder := httptest.NewRecorder()
		middleware.Auth(endpointHandler, apiHandler.Config.JWTSecret).ServeHTTP(responseRecorder, httpRequest)
		status(test, responseRecorder, 502)
	}
}
