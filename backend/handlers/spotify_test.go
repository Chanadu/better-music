package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chanadu/better-music/config"
	"github.com/danielgtaylor/huma/v2"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (transport transportFunc) RoundTrip(httpRequest *http.Request) (*http.Response, error) {
	return transport(httpRequest)
}

func spotifyResponse(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func spotifyHandler(transport transportFunc) *Handler {
	apiHandler := NewHandler(&config.Config{
		Spotify: config.SpotifyConfig{ClientID: "id", ClientSecret: "secret"},
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

func TestSpotifyMissingArrays(test *testing.T) {
	handler := spotifyHandler(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == spotifyAuthURL {
			return spotifyResponse(200, `{"access_token":"token","expires_in":3600}`), nil
		}
		if strings.HasSuffix(request.URL.Path, "/search") {
			return spotifyResponse(200, `{"albums":{"items":null}}`), nil
		}
		return spotifyResponse(200, `{"id":"abc","images":null}`), nil
	})
	ctx := context.Background()

	artists, err := handler.searchSpotifyArtists(ctx, "music", 6)
	if err != nil || artists == nil || len(artists) != 0 {
		test.Fatalf("artist search = %v, %v; want empty array", artists, err)
	}
	albums, err := handler.searchSpotifyAlbums(ctx, "music", 6)
	if err != nil || albums == nil || len(albums) != 0 {
		test.Fatalf("album search = %v, %v; want empty array", albums, err)
	}
	artist, err := handler.getSpotifyArtist(ctx, "abc")
	if err != nil || artist.Images == nil || len(artist.Images) != 0 {
		test.Fatalf("artist = %+v, %v; want empty images array", artist, err)
	}
	album, err := handler.getSpotifyAlbum(ctx, "abc")
	if err != nil || album.Images == nil || album.Artists == nil || len(album.Images) != 0 || len(album.Artists) != 0 {
		test.Fatalf("album = %+v, %v; want empty images and artists arrays", album, err)
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
			if err != nil || token.AccessToken != "token" {
				test.Errorf("unexpected token result: %v", err)
			}
		})
	}

	waitGroup.Wait()
	if calls.Load() != 1 {
		test.Fatal("token fetched more than once")
	}

	apiHandler.spotifyToken.Expiry = time.Now().Add(5 * time.Second)
	if _, err := apiHandler.getSpotifyAccessToken(context.Background()); err != nil {
		test.Fatal(err)
	}

	if calls.Load() != 2 {
		test.Error("near-expiry token not refreshed")
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

			if apiHandler.spotifyToken != nil {
				test.Error("invalid token cached")
			}
		})
	}

	apiHandler := NewHandler(&config.Config{}, nil)
	var statusError huma.StatusError
	if _, err := apiHandler.getSpotifyAccessToken(context.Background()); !errors.As(err, &statusError) || statusError.GetStatus() != http.StatusServiceUnavailable {
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
			identity := createAccount(test, apiHandler, "spotify@test.example").Identity

			mux := http.NewServeMux()
			apiHandler.RegisterAPI(mux)
			search := "/api/spotify/search/" + kind
			lookup := "/api/spotify/" + kind + "/abc"
			responseRecorder := request(apiHandler, mux.ServeHTTP, "GET", search+"?q=AC%2FDC+%26+friends&limit=10", "", identity, nil)
			status(test, responseRecorder, 200)
			if !strings.Contains(responseRecorder.Body.String(), `"id":"abc"`) {
				test.Fatal(responseRecorder.Body)
			}

			status(test, request(apiHandler, mux.ServeHTTP, "GET", lookup, "", identity, nil), 200)

			for _, query := range []string{"", "?q=music&limit=11", "?q=music&limit=1.5"} {
				status(test, request(apiHandler, mux.ServeHTTP, "GET", search+query, "", identity, nil), 400)
			}

			apiHandler.Config.Spotify = config.SpotifyConfig{}
			status(test, request(apiHandler, mux.ServeHTTP, "GET", search+"?q=music", "", identity, nil), 503)
		})
	}
}

func TestSpotifyRetryAndFailures(test *testing.T) {
	for _, code := range []int{200, 401, 404, 429, 500} {
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
			_, err := apiHandler.getSpotifyArtist(context.Background(), "abc")
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
		identity := createAccount(test, apiHandler, "failed@test.example").Identity
		mux := http.NewServeMux()
		apiHandler.RegisterAPI(mux)
		for _, kind := range []string{"artists", "albums"} {
			status(test, request(apiHandler, mux.ServeHTTP, "GET", "/api/spotify/search/"+kind+"?q=x", "", identity, nil), 502)
		}
	}
}

func TestSpotifyRateLimitHeader(test *testing.T) {
	handler := spotifyHandler(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == spotifyAuthURL {
			return spotifyResponse(200, `{"access_token":"token","expires_in":3600}`), nil
		}
		response := spotifyResponse(429, "rate limited")
		response.Header = http.Header{"Retry-After": {"15"}}
		return response, nil
	})
	identity := createAccount(test, handler, "limited@test.example").Identity
	mux := http.NewServeMux()
	handler.RegisterAPI(mux)
	response := request(handler, mux.ServeHTTP, "GET", "/api/spotify/artists/abc", "", identity, nil)
	status(test, response, http.StatusBadGateway)
	if got := response.Header().Get("Retry-After"); got != "15" {
		test.Fatalf("Retry-After = %q, want 15", got)
	}
}
