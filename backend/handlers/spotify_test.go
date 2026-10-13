package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/middleware"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (transport transportFunc) RoundTrip(httpRequest *http.Request) (*http.Response, error) {
	return transport(httpRequest)
}

func spotifyResponse(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func spotifyHandler(transport transportFunc) *Handler {
	apiHandler := NewHandler(&config.Config{
		JWTSecret: "secret",
		AccessTTL: time.Hour,
		Spotify:   config.SpotifyConfig{ClientID: "id", ClientSecret: "secret"},
	}, nil)
	apiHandler.spotifyHTTPClient = &http.Client{Transport: transport}

	return apiHandler
}

func TestSpotifySearchLimits(test *testing.T) {
	for _, testCase := range []struct {
		raw  string
		want int
		bad  bool
	}{
		{"", 6, false},
		{" ", 6, false},
		{"1", 1, false},
		{"10", 10, false},
		{"0", 0, true},
		{"11", 0, true},
		{"bad", 0, true},
		{"1.5", 0, true},
	} {
		got, err := parseSpotifySearchLimit(testCase.raw)
		if got != testCase.want || (err != nil) != testCase.bad {
			test.Errorf("%q got %d %v", testCase.raw, got, err)
		}
	}
}

func TestSpotifyTokenCache(test *testing.T) {
	var calls atomic.Int32
	apiHandler := spotifyHandler(func(httpRequest *http.Request) (*http.Response, error) {
		calls.Add(1)
		clientID, secret, hasBasicAuth := httpRequest.BasicAuth()
		if !hasBasicAuth || clientID != "id" || secret != "secret" || httpRequest.Method != "POST" ||
			httpRequest.URL.String() != spotifyAuthURL {
			test.Error("invalid auth request")
		}

		body, _ := io.ReadAll(httpRequest.Body)
		if string(body) != "grant_type=client_credentials" {
			test.Error(string(body))
		}

		return spotifyResponse(200, `{"access_token":"token","expires_in":3600}`), nil
	})
	var waitGroup sync.WaitGroup

	for attemptIndex := 0; attemptIndex < 10; attemptIndex++ {
		waitGroup.Go(func() {
			token, err := apiHandler.getSpotifyAccessToken(context.Background())
			if err != nil || token != "token" {
				test.Errorf("%q %v", token, err)
			}
		})
	}

	waitGroup.Wait()
	if calls.Load() != 1 {
		test.Fatal("token fetched more than once")
	}

	apiHandler.spotifyTokenExpiresAt = time.Now().Add(5 * time.Second)
	if _, err := apiHandler.getSpotifyAccessToken(context.Background()); err != nil {
		test.Fatal(err)
	}

	if calls.Load() != 2 {
		test.Error("near-expiry token not refreshed")
	}

	apiHandler.clearSpotifyToken()
	if apiHandler.spotifyAccessToken != "" || !apiHandler.spotifyTokenExpiresAt.IsZero() {
		test.Error("cache not cleared")
	}
}

func TestSpotifyTokenErrors(test *testing.T) {
	for _, testCase := range []struct {
		name string
		code int
		body string
		err  error
	}{
		{"http", 401, "denied", nil},
		{"json", 200, "{", nil},
		{"missing", 200, `{"expires_in":3600}`, nil},
		{
			"expiry",
			200,
			`{"access_token":"x","expires_in":0}`,
			nil,
		},
		{"transport", 0, "", errors.New("offline")},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			apiHandler := spotifyHandler(func(*http.Request) (*http.Response, error) {
				if testCase.err != nil {
					return nil, testCase.err
				}

				return spotifyResponse(testCase.code, testCase.body), nil
			})
			if _, err := apiHandler.getSpotifyAccessToken(context.Background()); err == nil {
				test.Fatal("expected error")
			}

			if apiHandler.spotifyAccessToken != "" {
				test.Error("invalid token cached")
			}
		})
	}

	apiHandler := NewHandler(&config.Config{}, nil)
	if _, err := apiHandler.getSpotifyAccessToken(context.Background()); !errors.Is(err, errSpotifyNotConfigured) {
		test.Fatal(err)
	}
}

func TestSpotifyEndpoints(test *testing.T) {
	for _, kind := range []string{"artists", "albums"} {
		test.Run(kind, func(test *testing.T) {
			apiHandler := spotifyHandler(func(httpRequest *http.Request) (*http.Response, error) {
				if httpRequest.URL.String() == spotifyAuthURL {
					return spotifyResponse(200, `{"access_token":"token","expires_in":3600}`), nil
				}

				if httpRequest.Header.Get("Authorization") != "Bearer token" || httpRequest.Method != "GET" {
					test.Error("missing bearer token")
				}

				item := `{"id":"abc","name":"Music","images":[]}`
				if strings.HasSuffix(httpRequest.URL.Path, "/search") {
					queryParameters := httpRequest.URL.Query()
					if queryParameters.Get("q") != "AC/DC & friends" || queryParameters.Get("limit") != "10" ||
						queryParameters.Get("type") != strings.TrimSuffix(kind, "s") {
						test.Errorf("query %v", queryParameters)
					}

					return spotifyResponse(200, `{"`+kind+`":{"items":[`+item+`]}}`), nil
				}

				if httpRequest.URL.Path != "/v1/"+kind+"/abc" {
					test.Error(httpRequest.URL)
				}

				return spotifyResponse(200, item), nil
			})
			token, err := apiHandler.generateJWT(1)
			if err != nil {
				test.Fatal(err)
			}

			search, lookup := apiHandler.SearchSpotifyArtists, apiHandler.GetSpotifyArtist
			if kind == "albums" {
				search, lookup = apiHandler.SearchSpotifyAlbums, apiHandler.GetSpotifyAlbum
			}

			responseRecorder := request(apiHandler, search, "GET", "/api/spotify?q=AC%2FDC+%26+friends&limit=10", "", token, nil)
			status(test, responseRecorder, 200)
			if !strings.Contains(responseRecorder.Body.String(), `"id":"abc"`) {
				test.Fatal(responseRecorder.Body)
			}

			httpRequest := newSpotifyRequest("abc", token)
			lookupResponse := httptest.NewRecorder()
			middleware.Auth(http.HandlerFunc(lookup), apiHandler.Config.JWTSecret).ServeHTTP(lookupResponse, httpRequest)
			status(test, lookupResponse, 200)

			for _, path := range []string{"/api/spotify", "/api/spotify?q=music&limit=11"} {
				status(test, request(apiHandler, search, "GET", path, "", token, nil), 400)
			}

			apiHandler.Config.Spotify = config.SpotifyConfig{}
			status(test, request(apiHandler, search, "GET", "/api/spotify?q=music", "", token, nil), 503)
		})
	}
}

func TestSpotifyRetryAndFailures(test *testing.T) {
	for _, code := range []int{200, 401, 429, 500} {
		test.Run(http.StatusText(code), func(test *testing.T) {
			auth, api := 0, 0
			apiHandler := spotifyHandler(func(httpRequest *http.Request) (*http.Response, error) {
				if httpRequest.URL.String() == spotifyAuthURL {
					auth++

					return spotifyResponse(200, `{"access_token":"fresh","expires_in":3600}`), nil
				}

				api++
				if api == 1 {
					return spotifyResponse(401, "expired"), nil
				}

				return spotifyResponse(code, `{"id":"abc"}`), nil
			})
			var target map[string]any
			err := apiHandler.executeSpotifyJSONRequest(context.Background(), spotifyAPIBaseURL+"/artists/abc", &target)
			if (err != nil) != (code != 200) || auth != 2 || api != 2 {
				test.Fatalf("err=%v auth=%d api=%d", err, auth, api)
			}
		})
	}

	for _, testCase := range []struct {
		body string
		err  error
	}{{"{", nil}, {"", errors.New("network")}} {
		apiHandler := spotifyHandler(func(httpRequest *http.Request) (*http.Response, error) {
			if httpRequest.URL.String() == spotifyAuthURL {
				return spotifyResponse(200, `{"access_token":"token","expires_in":3600}`), nil
			}

			if testCase.err != nil {
				return nil, testCase.err
			}

			return spotifyResponse(200, testCase.body), nil
		})
		token, _ := apiHandler.generateJWT(1)
		status(test, request(apiHandler, apiHandler.SearchSpotifyArtists, "GET", "/api/spotify?q=x", "", token, nil), 502)

		status(test, request(apiHandler, apiHandler.SearchSpotifyAlbums, "GET", "/api/spotify?q=x", "", token, nil), 502)
	}
}

func newSpotifyRequest(spotifyID, token string) *http.Request {
	httpRequest := httptest.NewRequest("GET", "/api/spotify/artists/"+spotifyID, nil)
	httpRequest.SetPathValue("id", spotifyID)
	httpRequest.Header.Set("Authorization", "Bearer "+token)

	return httpRequest
}
