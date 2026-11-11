package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	defaultSpotifySearchLimit = 6
	maxSpotifySearchLimit     = 10
	spotifyTokenBuffer        = 30 * time.Second
	spotifyAuthURL            = "https://accounts.spotify.com/api/token"
)

type SpotifyImage struct {
	URL    string `json:"url" example:"https://i.scdn.co/image/ab67616d00001e02"`
	Width  int    `json:"width" example:"640"`
	Height int    `json:"height" example:"640"`
}

type SpotifyArtistSearchResult struct {
	ID     string         `json:"id" example:"4Z8W4fKeB5YxbusRsdQVPb"`
	Name   string         `json:"name" example:"Radiohead"`
	Images []SpotifyImage `json:"images" nullable:"false"`
}

type SpotifyAlbumArtist struct {
	ID   string `json:"id" example:"4Z8W4fKeB5YxbusRsdQVPb"`
	Name string `json:"name" example:"Radiohead"`
}

type SpotifyAlbumSearchResult struct {
	ID          string               `json:"id" example:"6ZG5lRT77aJ3btmArcykra"`
	Name        string               `json:"name" example:"OK Computer"`
	Artists     []SpotifyAlbumArtist `json:"artists" nullable:"false"`
	Images      []SpotifyImage       `json:"images" nullable:"false"`
	ReleaseDate string               `json:"release_date" example:"1997-05-21"`
}

type spotifyUpstreamError struct {
	operation  string
	status     int
	retryAfter string
}

func (err spotifyUpstreamError) Error() string {
	return fmt.Sprintf("Spotify %s returned HTTP %d", err.operation, err.status)
}

func (h *Handler) getSpotifyAccessToken(ctx context.Context) (*oauth2.Token, error) {
	if strings.TrimSpace(h.Config.Spotify.ClientID) == "" || strings.TrimSpace(h.Config.Spotify.ClientSecret) == "" {
		return nil, huma.Error503ServiceUnavailable("spotify is not configured")
	}

	h.spotifyMutex.Lock()
	defer h.spotifyMutex.Unlock()

	if h.spotifyToken != nil && time.Until(h.spotifyToken.Expiry) > spotifyTokenBuffer {
		return h.spotifyToken, nil
	}

	credentials := clientcredentials.Config{
		ClientID:     h.Config.Spotify.ClientID,
		ClientSecret: h.Config.Spotify.ClientSecret,
		TokenURL:     spotifyAuthURL,
		AuthStyle:    oauth2.AuthStyleInHeader,
	}

	token, err := credentials.Token(context.WithValue(ctx, oauth2.HTTPClient, h.spotifyHTTPClient))
	if err != nil {
		var upstream *oauth2.RetrieveError
		if errors.As(err, &upstream) && upstream.Response.StatusCode != http.StatusOK {
			return nil, spotifyResponseError("token request", upstream.Response)
		}

		return nil, errors.New("spotify token request failed")
	}

	if !token.Expiry.After(time.Now()) {
		return nil, errors.New("spotify auth returned an invalid token expiry")
	}

	h.spotifyToken = token
	return token, nil
}

func spotifyResponseError(operation string, response *http.Response) spotifyUpstreamError {
	return spotifyUpstreamError{
		operation:  operation,
		status:     response.StatusCode,
		retryAfter: response.Header.Get("Retry-After"),
	}
}

type spotifyTransport struct {
	handler *Handler
}

func (transport spotifyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	h := transport.handler
	base := h.spotifyHTTPClient.Transport

	if base == nil {
		base = http.DefaultTransport
	}

	for retry := true; ; retry = false {
		token, err := h.getSpotifyAccessToken(request.Context())
		if err != nil {
			return nil, err
		}

		authenticated := request.Clone(request.Context())
		authenticated.Header.Set("Authorization", "Bearer "+token.AccessToken)

		response, err := base.RoundTrip(authenticated)
		if err != nil {
			return nil, err
		}

		if response.StatusCode == http.StatusOK {
			return response, nil
		}

		response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized || !retry {
			return nil, spotifyResponseError("API request", response)
		}

		h.spotifyMutex.Lock()
		if h.spotifyToken == token {
			h.spotifyToken = nil
		}

		h.spotifyMutex.Unlock()
	}
}

func parseSpotifySearchLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultSpotifySearchLimit, nil
	}

	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("limit must be an integer")
	}

	if limit < 1 || limit > maxSpotifySearchLimit {
		return 0, fmt.Errorf("limit must be between 1 and %d", maxSpotifySearchLimit)
	}

	return limit, nil
}

func (h *Handler) searchSpotifyArtists(ctx context.Context, query string, limit int) ([]SpotifyArtistSearchResult, error) {
	results, err := h.spotifyClient.Search(ctx, query, spotify.SearchTypeArtist, spotify.Limit(limit))
	if err != nil {
		return nil, err
	}
	if results.Artists == nil {
		return []SpotifyArtistSearchResult{}, nil
	}

	return mapSpotifyResults(results.Artists.Artists, spotifyArtistResult), nil
}

func (h *Handler) getSpotifyArtist(ctx context.Context, spotifyID string) (*SpotifyArtistSearchResult, error) {
	artist, err := h.spotifyClient.GetArtist(ctx, spotify.ID(url.PathEscape(spotifyID)))
	if err != nil {
		return nil, err
	}

	result := spotifyArtistResult(*artist)
	return &result, nil
}

func (h *Handler) searchSpotifyAlbums(ctx context.Context, query string, limit int) ([]SpotifyAlbumSearchResult, error) {
	results, err := h.spotifyClient.Search(ctx, query, spotify.SearchTypeAlbum, spotify.Limit(limit))
	if err != nil {
		return nil, err
	}
	if results.Albums == nil {
		return []SpotifyAlbumSearchResult{}, nil
	}

	return mapSpotifyResults(results.Albums.Albums, spotifyAlbumResult), nil
}

func (h *Handler) getSpotifyAlbum(ctx context.Context, spotifyID string) (*SpotifyAlbumSearchResult, error) {
	album, err := h.spotifyClient.GetAlbum(ctx, spotify.ID(url.PathEscape(spotifyID)))
	if err != nil {
		return nil, err
	}

	result := spotifyAlbumResult(album.SimpleAlbum)
	return &result, nil
}

func spotifyArtistResult(artist spotify.FullArtist) SpotifyArtistSearchResult {
	return SpotifyArtistSearchResult{
		ID: string(artist.ID), Name: artist.Name, Images: mapSpotifyResults(artist.Images, spotifyImage),
	}
}

func spotifyAlbumResult(album spotify.SimpleAlbum) SpotifyAlbumSearchResult {
	return SpotifyAlbumSearchResult{
		ID: string(album.ID), Name: album.Name, ReleaseDate: album.ReleaseDate,
		Images: mapSpotifyResults(album.Images, spotifyImage),
		Artists: mapSpotifyResults(album.Artists, func(artist spotify.SimpleArtist) SpotifyAlbumArtist {
			return SpotifyAlbumArtist{ID: string(artist.ID), Name: artist.Name}
		}),
	}
}

func spotifyImage(image spotify.Image) SpotifyImage {
	return SpotifyImage{URL: image.URL, Width: int(image.Width), Height: int(image.Height)}
}

func mapSpotifyResults[In, Out any](items []In, convert func(In) Out) []Out {
	result := make([]Out, len(items))
	for index, item := range items {
		result[index] = convert(item)
	}
	return result
}
