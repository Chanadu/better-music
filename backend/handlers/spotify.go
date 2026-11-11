package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const (
	defaultSpotifySearchLimit = 6
	maxSpotifySearchLimit     = 10
	spotifyTokenBuffer        = 30 * time.Second
	spotifyAuthURL            = "https://accounts.spotify.com/api/token"
	spotifyAPIBaseURL         = "https://api.spotify.com/v1"
)

type SpotifyImage struct {
	URL    string `json:"url" example:"https://i.scdn.co/image/ab67616d00001e02"`
	Width  int    `json:"width" example:"640"`
	Height int    `json:"height" example:"640"`
}

type SpotifyArtistSearchResult struct {
	ID     string         `json:"id" example:"4Z8W4fKeB5YxbusRsdQVPb"`
	Name   string         `json:"name" example:"Radiohead"`
	Images []SpotifyImage `json:"images" nullable:"true"`
}

type SpotifyAlbumArtist struct {
	ID   string `json:"id" example:"4Z8W4fKeB5YxbusRsdQVPb"`
	Name string `json:"name" example:"Radiohead"`
}

type SpotifyAlbumSearchResult struct {
	ID          string               `json:"id" example:"6ZG5lRT77aJ3btmArcykra"`
	Name        string               `json:"name" example:"OK Computer"`
	Artists     []SpotifyAlbumArtist `json:"artists" nullable:"true"`
	Images      []SpotifyImage       `json:"images" nullable:"true"`
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

type spotifyTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type spotifySearchArtistsResponse struct {
	Artists struct {
		Items []SpotifyArtistSearchResult `json:"items"`
	} `json:"artists"`
}

type spotifySearchAlbumsResponse struct {
	Albums struct {
		Items []SpotifyAlbumSearchResult `json:"items"`
	} `json:"albums"`
}

func (h *Handler) spotifyConfigured() bool {
	return strings.TrimSpace(h.Config.Spotify.ClientID) != "" && strings.TrimSpace(h.Config.Spotify.ClientSecret) != ""
}

func (h *Handler) clearSpotifyToken() {
	h.spotifyMutex.Lock()
	defer h.spotifyMutex.Unlock()

	h.spotifyAccessToken = ""
	h.spotifyTokenExpiresAt = time.Time{}
}

func (h *Handler) getSpotifyAccessToken(ctx context.Context) (string, error) {
	if !h.spotifyConfigured() {
		return "", huma.Error503ServiceUnavailable("spotify is not configured")
	}

	h.spotifyMutex.Lock()
	defer h.spotifyMutex.Unlock()

	if h.spotifyAccessToken != "" && time.Until(h.spotifyTokenExpiresAt) > spotifyTokenBuffer {
		return h.spotifyAccessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, spotifyAuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}

	basicAuth := base64.StdEncoding.EncodeToString(
		fmt.Appendf(nil, "%s:%s", h.Config.Spotify.ClientID, h.Config.Spotify.ClientSecret),
	)
	req.Header.Set("Authorization", "Basic "+basicAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := h.spotifyHTTPClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", spotifyUpstreamError{operation: "token request", status: res.StatusCode, retryAfter: res.Header.Get("Retry-After")}
	}

	var payload spotifyTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", err
	}

	if payload.AccessToken == "" || payload.ExpiresIn <= 0 {
		return "", errors.New("spotify auth returned an invalid token response")
	}

	h.spotifyAccessToken = payload.AccessToken
	h.spotifyTokenExpiresAt = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	return h.spotifyAccessToken, nil
}

func (h *Handler) executeSpotifyJSONRequest(ctx context.Context, endpoint string, target any) error {
	token, err := h.getSpotifyAccessToken(ctx)
	if err != nil {
		return err
	}

	doRequest := func(accessToken string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		return h.spotifyHTTPClient.Do(req)
	}

	res, err := doRequest(token)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusUnauthorized {
		res.Body.Close()
		h.clearSpotifyToken()

		token, err = h.getSpotifyAccessToken(ctx)
		if err != nil {
			return err
		}

		res, err = doRequest(token)
		if err != nil {
			return err
		}
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return spotifyUpstreamError{operation: "API request", status: res.StatusCode, retryAfter: res.Header.Get("Retry-After")}
	}

	return json.NewDecoder(res.Body).Decode(target)
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
	values := url.Values{}
	values.Set("q", query)
	values.Set("type", "artist")
	values.Set("limit", strconv.Itoa(limit))

	var payload spotifySearchArtistsResponse
	if err := h.executeSpotifyJSONRequest(ctx, spotifyAPIBaseURL+"/search?"+values.Encode(), &payload); err != nil {
		return nil, err
	}

	return payload.Artists.Items, nil
}

func (h *Handler) getSpotifyArtist(ctx context.Context, spotifyID string) (*SpotifyArtistSearchResult, error) {
	var artist SpotifyArtistSearchResult
	if err := h.executeSpotifyJSONRequest(
		ctx,
		spotifyAPIBaseURL+"/artists/"+url.PathEscape(spotifyID),
		&artist,
	); err != nil {
		return nil, err
	}

	return &artist, nil
}

func (h *Handler) searchSpotifyAlbums(ctx context.Context, query string, limit int) ([]SpotifyAlbumSearchResult, error) {
	values := url.Values{}
	values.Set("q", query)
	values.Set("type", "album")
	values.Set("limit", strconv.Itoa(limit))

	var payload spotifySearchAlbumsResponse
	if err := h.executeSpotifyJSONRequest(ctx, spotifyAPIBaseURL+"/search?"+values.Encode(), &payload); err != nil {
		return nil, err
	}

	return payload.Albums.Items, nil
}

func (h *Handler) getSpotifyAlbum(ctx context.Context, spotifyID string) (*SpotifyAlbumSearchResult, error) {
	var album SpotifyAlbumSearchResult
	if err := h.executeSpotifyJSONRequest(
		ctx,
		spotifyAPIBaseURL+"/albums/"+url.PathEscape(spotifyID),
		&album,
	); err != nil {
		return nil, err
	}

	return &album, nil
}
