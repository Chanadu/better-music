package handlers

import (
	"net/http"
	"testing"

	"github.com/Chanadu/better-music/config"
)

func TestMissingAccountDuringSync(test *testing.T) {
	apiHandler := testHandler(test)
	account := createAccount(test, apiHandler, "a@example.com")
	if _, err := apiHandler.Database.Exec("DELETE FROM users WHERE id=$1", account.UserID); err != nil {
		test.Fatal(err)
	}

	status(test, syncCall(test, apiHandler, "artists", "pull", account.Identity, PullLibraryRequest{Limit: 10}), 403)
}

func TestSpotifyLookupErrors(test *testing.T) {
	apiHandler := NewHandler(&config.Config{}, nil)
	identity := createAccount(test, apiHandler, "lookup@test.example").Identity

	mux := http.NewServeMux()
	apiHandler.RegisterAPI(mux)
	for _, kind := range []string{"artists", "albums"} {
		path := "/api/spotify/" + kind + "/"
		status(test, request(apiHandler, mux.ServeHTTP, "GET", path+"%20", "", identity, nil), 400)
		status(test, request(apiHandler, mux.ServeHTTP, "GET", path+"abc", "", identity, nil), 503)
	}

	apiHandler = spotifyHandler(func(httpRequest *http.Request) (*http.Response, error) {
		if httpRequest.URL.String() == spotifyAuthURL {
			return spotifyResponse(200, `{"access_token":"token","expires_in":3600}`), nil
		}

		return spotifyResponse(404, "missing"), nil
	})
	identity = createAccount(test, apiHandler, "missing@test.example").Identity

	mux = http.NewServeMux()
	apiHandler.RegisterAPI(mux)
	for _, kind := range []string{"artists", "albums"} {
		status(test, request(apiHandler, mux.ServeHTTP, "GET", "/api/spotify/"+kind+"/abc", "", identity, nil), 502)
	}
}
