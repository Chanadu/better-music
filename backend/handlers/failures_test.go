package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/middleware"
)

func TestMissingAccountDuringSync(test *testing.T) {
	apiHandler := testHandler(test)
	accountTokens := register(test, apiHandler, "a@example.com")
	if _, err := apiHandler.Database.Exec("DELETE FROM users WHERE id=$1", accountTokens.UserID); err != nil {
		test.Fatal(err)
	}

	status(test, syncCall(test, apiHandler, "artists", "pull", accountTokens.AccessToken, PullLibraryRequest{Limit: 10}), 401)
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
