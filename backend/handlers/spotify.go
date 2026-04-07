package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSpotifySearchLimit = 6
	maxSpotifySearchLimit     = 10
	spotifyTokenBuffer        = 30 * time.Second
	spotifyAuthURL            = "https://accounts.spotify.com/api/token"
	spotifyAPIBaseURL         = "https://api.spotify.com/v1"
)

var errSpotifyNotConfigured = errors.New("spotify is not configured")

type SpotifyImage struct {
	URL    string `json:"url" example:"https://i.scdn.co/image/ab67616d00001e02"`
	Width  int    `json:"width" example:"640"`
	Height int    `json:"height" example:"640"`
}

type SpotifyArtistSearchResult struct {
	ID     string         `json:"id" example:"4Z8W4fKeB5YxbusRsdQVPb"`
	Name   string         `json:"name" example:"Radiohead"`
	Images []SpotifyImage `json:"images"`
}

type SpotifyAlbumArtist struct {
	ID   string `json:"id" example:"4Z8W4fKeB5YxbusRsdQVPb"`
	Name string `json:"name" example:"Radiohead"`
}

type SpotifyAlbumSearchResult struct {
	ID          string               `json:"id" example:"6ZG5lRT77aJ3btmArcykra"`
	Name        string               `json:"name" example:"OK Computer"`
	Artists     []SpotifyAlbumArtist `json:"artists"`
	Images      []SpotifyImage       `json:"images"`
	ReleaseDate string               `json:"release_date" example:"1997-05-21"`
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
		return "", errSpotifyNotConfigured
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
		[]byte(fmt.Sprintf("%s:%s", h.Config.Spotify.ClientID, h.Config.Spotify.ClientSecret)),
	)
	req.Header.Set("Authorization", "Basic "+basicAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := h.spotifyHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return "", fmt.Errorf("spotify auth failed (%d): %s", res.StatusCode, strings.TrimSpace(string(body)))
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
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("spotify request failed (%d): %s", res.StatusCode, strings.TrimSpace(string(body)))
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

func writeSpotifyError(w http.ResponseWriter, message string, status int, err error) {
	if err != nil {
		slog.Warn("spotify proxy request failed", "status", status, "error", err)
	}
	writeJSON(w, status, apiError(message))
}

// SearchSpotifyArtists godoc
// @Summary Search Spotify artists
// @Description Search Spotify for artists and return trimmed results for the authenticated user
// @Tags spotify
// @Produce json
// @Security Bearer
// @Param q query string true "Search query"
// @Param limit query int false "Max results (1-10)"
// @Success 200 {array} handlers.SpotifyArtistSearchResult
// @Failure 400 {object} map[string]string "Missing or invalid query parameters"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 502 {object} map[string]string "Spotify request failed"
// @Failure 503 {object} map[string]string "Spotify is not configured"
// @Router /api/spotify/search/artists [get]
func (h *Handler) SearchSpotifyArtists(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/spotify/search/artists", "method", r.Method, "path", r.URL.Path)
	if _, ok := getUserID(w, r); !ok {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusBadRequest, apiError("q is required"))
		return
	}

	limit, err := parseSpotifySearchLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError(err.Error()))
		return
	}

	results, err := h.searchSpotifyArtists(r.Context(), query, limit)
	if err != nil {
		if errors.Is(err, errSpotifyNotConfigured) {
			writeSpotifyError(w, "spotify is not configured", http.StatusServiceUnavailable, err)
			return
		}

		writeSpotifyError(w, "spotify artist search failed", http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}

// SearchSpotifyAlbums godoc
// @Summary Search Spotify albums
// @Description Search Spotify for albums and return trimmed results for the authenticated user
// @Tags spotify
// @Produce json
// @Security Bearer
// @Param q query string true "Search query"
// @Param limit query int false "Max results (1-10)"
// @Success 200 {array} handlers.SpotifyAlbumSearchResult
// @Failure 400 {object} map[string]string "Missing or invalid query parameters"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 502 {object} map[string]string "Spotify request failed"
// @Failure 503 {object} map[string]string "Spotify is not configured"
// @Router /api/spotify/search/albums [get]
func (h *Handler) SearchSpotifyAlbums(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/spotify/search/albums", "method", r.Method, "path", r.URL.Path)
	if _, ok := getUserID(w, r); !ok {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusBadRequest, apiError("q is required"))
		return
	}

	limit, err := parseSpotifySearchLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError(err.Error()))
		return
	}

	results, err := h.searchSpotifyAlbums(r.Context(), query, limit)
	if err != nil {
		if errors.Is(err, errSpotifyNotConfigured) {
			writeSpotifyError(w, "spotify is not configured", http.StatusServiceUnavailable, err)
			return
		}

		writeSpotifyError(w, "spotify album search failed", http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}
