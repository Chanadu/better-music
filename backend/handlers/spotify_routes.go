package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/middleware"
	"github.com/danielgtaylor/huma/v2"
)

type spotifyLookupInput struct {
	ID string `path:"id"`
}

type spotifySearchInput struct {
	Query string `query:"q" required:"true"`
	Limit string `query:"limit" doc:"Integer between 1 and 10. Defaults to 6."`
}

type spotifySearchOutput[T any] struct {
	Body []T `nullable:"false"`
}

func (h *Handler) registerSpotify(api huma.API) {
	huma.Register(api, spotifyOperation(api, "getSpotifyArtist", "/api/spotify/artists/{id}"),
		spotifyLookup("artist", h.getSpotifyArtist))

	huma.Register(api, spotifyOperation(api, "getSpotifyAlbum", "/api/spotify/albums/{id}"),
		spotifyLookup("album", h.getSpotifyAlbum))

	huma.Register(api, spotifyOperation(api, "searchSpotifyArtists", "/api/spotify/search/artists"),
		spotifySearch("artist", h.searchSpotifyArtists))

	huma.Register(api, spotifyOperation(api, "searchSpotifyAlbums", "/api/spotify/search/albums"),
		spotifySearch("album", h.searchSpotifyAlbums))
}

func spotifyOperation(api huma.API, id, path string) huma.Operation {
	responses := authenticatedResponses(api)

	addProblemResponses(api, responses, http.StatusBadRequest, http.StatusBadGateway)
	responses["502"].Headers = map[string]*huma.Header{
		"Retry-After": {
			Description: "Retry delay provided by Spotify when rate limited.",
			Schema:      &huma.Schema{Type: huma.TypeString},
		},
	}

	return huma.Operation{
		OperationID:        id,
		Method:             http.MethodGet,
		Path:               path,
		Responses:          responses,
		SkipValidateParams: true,
	}
}

func spotifyLookup[T any](kind string, lookup func(context.Context, string) (*T, error)) func(context.Context, *spotifyLookupInput) (*jsonOutput[*T], error) {
	return func(ctx context.Context, input *spotifyLookupInput) (*jsonOutput[*T], error) {
		if _, ok := middleware.UserIDFromContext(ctx); !ok {
			return nil, huma.Error401Unauthorized("unauthorized")
		}

		id := strings.TrimSpace(input.ID)
		if id == "" {
			return nil, huma.Error400BadRequest("spotify " + kind + " ID is required")
		}

		result, err := lookup(ctx, id)
		if err != nil {
			return nil, spotifyError(ctx, "spotify "+kind+" lookup failed", err)
		}

		return &jsonOutput[*T]{Body: result}, nil
	}
}

func spotifySearch[T any](kind string, search func(context.Context, string, int) ([]T, error)) func(context.Context, *spotifySearchInput) (*spotifySearchOutput[T], error) {
	return func(ctx context.Context, input *spotifySearchInput) (*spotifySearchOutput[T], error) {
		if _, ok := middleware.UserIDFromContext(ctx); !ok {
			return nil, huma.Error401Unauthorized("unauthorized")
		}

		query := strings.TrimSpace(input.Query)
		if query == "" {
			return nil, huma.Error400BadRequest("q is required")
		}

		limit, err := parseSpotifySearchLimit(input.Limit)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		results, err := search(ctx, query, limit)
		if err != nil {
			return nil, spotifyError(ctx, "spotify "+kind+" search failed", err)
		}

		return &spotifySearchOutput[T]{Body: results}, nil
	}
}

func spotifyError(ctx context.Context, message string, err error) error {
	var upstreamError spotifyUpstreamError
	var statusError huma.StatusError

	switch {
	case errors.As(err, &upstreamError):
		slog.WarnContext(ctx, message, "operation", upstreamError.operation, "upstream_status", upstreamError.status)

		if upstreamError.status == http.StatusTooManyRequests && upstreamError.retryAfter != "" {
			return huma.ErrorWithHeaders(huma.Error502BadGateway(message), http.Header{
				"Retry-After": {upstreamError.retryAfter},
			})
		}

	case errors.As(err, &statusError):
		slog.WarnContext(ctx, statusError.Error())
		return err

	default:
		slog.WarnContext(ctx, message, logger.ErrorDetails(err))
	}

	return huma.Error502BadGateway(message)
}
