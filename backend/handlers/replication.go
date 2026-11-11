package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
	"github.com/danielgtaylor/huma/v2"
)

type LibraryCheckpoint struct {
	Epoch    string `json:"epoch"`
	Sequence string `json:"sequence"`
}

type PullLibraryRequest struct {
	Checkpoint *LibraryCheckpoint `json:"checkpoint"`
	Limit      int                `json:"limit"`
}

type LibraryChange struct {
	Assumed json.RawMessage `json:"assumedMasterState"`
	New     json.RawMessage `json:"newDocumentState"`
}

type PushLibraryRequest struct {
	Epoch string          `json:"epoch"`
	Rows  []LibraryChange `json:"rows"`
}

type LibraryChangeFailure struct {
	ID        string `json:"id"`
	Retryable bool   `json:"retryable"`
	Error     string `json:"error"`
}

type PullLibraryResponse struct {
	Documents  []json.RawMessage `json:"documents" nullable:"false"`
	Checkpoint LibraryCheckpoint `json:"checkpoint"`
}

type PushLibraryErrorResponse struct {
	Error    string                 `json:"error"`
	Failures []LibraryChangeFailure `json:"failures" nullable:"false"`
}

type pushLibraryFailure struct {
	Status   int                    `json:"-"`
	Message  string                 `json:"error"`
	Failures []LibraryChangeFailure `json:"failures"`
}

func (err pushLibraryFailure) GetStatus() int { return err.Status }
func (err pushLibraryFailure) Error() string  { return err.Message }

func (h *Handler) registerSync(api huma.API) {
	schemas := api.OpenAPI().Components.Schemas
	schemas.RegisterTypeAlias(reflect.TypeFor[models.NullableField[string]](), reflect.TypeFor[string]())
	schemas.RegisterTypeAlias(reflect.TypeFor[models.NullableField[int32]](), reflect.TypeFor[int32]())

	document := &huma.Schema{AnyOf: []*huma.Schema{
		schemas.Schema(reflect.TypeFor[models.Artist](), true, ""),
		schemas.Schema(reflect.TypeFor[models.Album](), true, ""),
	}}

	pull := schemas.Schema(reflect.TypeFor[PullLibraryResponse](), true, "")
	schemas.SchemaFromRef(pull.Ref).Properties["documents"].Items = document

	push := schemas.Schema(reflect.TypeFor[PushLibraryRequest](), true, "")
	row := schemas.Schema(reflect.TypeFor[LibraryChange](), true, "")

	rowProperties := schemas.SchemaFromRef(row.Ref).Properties
	rowProperties["newDocumentState"] = document
	rowProperties["assumedMasterState"] = &huma.Schema{AnyOf: append([]*huma.Schema{{Type: "null"}}, document.AnyOf...)}

	schemas.SchemaFromRef(row.Ref).Required = []string{"newDocumentState"}

	pullRequest := schemas.Schema(reflect.TypeFor[PullLibraryRequest](), true, "")
	checkpoint := schemas.SchemaFromRef(pullRequest.Ref).Properties
	checkpoint["checkpoint"] = &huma.Schema{AnyOf: []*huma.Schema{{Type: "null"}, checkpoint["checkpoint"]}}

	registerSyncRoute(api, "pull", pull, pullRequest, h.pullLibrary)
	registerSyncRoute(api, "push", &huma.Schema{Type: huma.TypeArray, Items: document}, push, h.pushLibrary)
}

func registerSyncRoute[I, O any](api huma.API, action string, outputSchema, inputSchema *huma.Schema, handler func(context.Context, string, models.LibraryKind, I) (*jsonOutput[O], error)) {
	responses := authenticatedResponses(api)
	responses["200"] = &huma.Response{Content: map[string]*huma.MediaType{"application/json": {Schema: outputSchema}}}

	addProblemResponses(api, responses, http.StatusBadRequest, http.StatusNotFound, http.StatusConflict,
		http.StatusRequestEntityTooLarge, http.StatusInternalServerError)

	if action == "push" {
		addProblemResponses(api, responses, http.StatusUnprocessableEntity)
		schemas := api.OpenAPI().Components.Schemas
		failures := schemas.Schema(reflect.TypeFor[PushLibraryErrorResponse](), true, "")
		conflict := responses["409"].Content["application/json"]
		conflict.Schema = &huma.Schema{AnyOf: []*huma.Schema{conflict.Schema, failures}}
		responses["422"].Content["application/json"] = &huma.MediaType{Schema: failures}
	}

	registerRawJSON(api, huma.Operation{
		OperationID:     action + "Library",
		Method:          http.MethodPost,
		Path:            "/api/sync/{kind}/" + action,
		MaxBodyBytes:    2 << 20,
		BodyReadTimeout: -1,
		Responses:       responses,
	}, inputSchema, func(ctx context.Context, request *struct {
		Kind    string `path:"kind"`
		RawBody []byte
	}) (*jsonOutput[O], error) {
		userID, ok := middleware.UserIDFromContext(ctx)

		if !ok {
			return nil, huma.Error401Unauthorized("unauthorized")
		}

		var kind models.LibraryKind

		switch request.Kind {
		case "artists":
			kind = models.ArtistKind

		case "albums":
			kind = models.AlbumKind

		default:
			return nil, huma.Error404NotFound("unknown collection")
		}

		var input I
		if err := decodeJSON(bytes.NewReader(request.RawBody), &input); err != nil {
			return nil, huma.Error400BadRequest("invalid or oversized sync request")
		}

		return handler(ctx, userID, kind, input)
	})
}

func syncStorageError(ctx context.Context, userID string, kind models.LibraryKind, operation string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return huma.Error401Unauthorized("account no longer exists")
	}

	logSyncFailure(ctx, userID, kind, operation, err)
	return huma.Error500InternalServerError("library storage is unavailable")
}

func logSyncFailure(ctx context.Context, userID string, kind models.LibraryKind, operation string, err error) {
	slog.ErrorContext(ctx, "library sync failed", "operation", operation, "collection", kind, "user_id", userID, logger.ErrorDetails(err))
}
