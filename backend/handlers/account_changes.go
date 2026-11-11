package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/Chanadu/better-music/internal/auth"
	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/middleware"
	"github.com/danielgtaylor/huma/v2"
)

type updateEmailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password" writeOnly:"true"`
}

type updatePasswordRequest struct {
	CurrentPassword string `json:"current_password" writeOnly:"true"`
	NewPassword     string `json:"new_password" writeOnly:"true"`
}

type deleteAccountRequest struct {
	Password string `json:"password" writeOnly:"true"`
}

func (h *Handler) registerAccountChanges(api huma.API) {
	registerAccountChange(api, http.MethodPut, "/api/account/email", "updateAccountEmail",
		func(ctx context.Context, id string, body updateEmailRequest) error {
			return h.changeAccount(ctx, id, auth.ChangeEmail, body.Password, body.Email)
		})

	registerAccountChange(api, http.MethodPut, "/api/account/password", "updateAccountPassword",
		func(ctx context.Context, id string, body updatePasswordRequest) error {
			return h.changeAccount(ctx, id, auth.ChangePassword, body.CurrentPassword, body.NewPassword)
		})

	registerAccountChange(api, http.MethodDelete, "/api/account", "deleteAccount",
		func(ctx context.Context, id string, body deleteAccountRequest) error {
			return h.changeAccount(ctx, id, auth.DeleteAccount, body.Password, "")
		})
}

func registerAccountChange[T any](api huma.API, method, path, operationID string, change func(context.Context, string, T) error) {
	responses := authenticatedResponses(api)
	addProblemResponses(api, responses, http.StatusForbidden, http.StatusConflict)

	schema := api.OpenAPI().Components.Schemas.Schema(reflect.TypeFor[T](), true, "")

	registerRawJSON(api, huma.Operation{
		OperationID:  operationID,
		Method:       method,
		Path:         path,
		MaxBodyBytes: 16384,
		Responses:    responses,
		Errors:       []int{http.StatusBadRequest, http.StatusRequestEntityTooLarge},
	}, schema, func(ctx context.Context, input *struct {
		RawBody []byte
	}) (*jsonOutput[MessageResponse], error) {
		id, ok := middleware.UserIDFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized("unauthorized")
		}

		var body T
		if err := json.NewDecoder(bytes.NewReader(input.RawBody)).Decode(&body); err != nil {
			return nil, huma.Error400BadRequest("invalid account request")
		}

		if err := change(ctx, id, body); err != nil {
			return nil, err
		}

		return &jsonOutput[MessageResponse]{Body: MessageResponse{Message: "account updated"}}, nil
	})
}

func (h *Handler) changeAccount(ctx context.Context, id string, kind auth.AccountOperation, password, value string) error {
	err := h.Auth.Change(ctx, id, kind, password, value)
	if err == nil {
		return nil
	}

	var statusError huma.StatusError
	if errors.As(err, &statusError) {
		return err
	}

	slog.ErrorContext(ctx, "account change failed", "user_id", id, "operation", kind, logger.ErrorDetails(err))
	return huma.Error503ServiceUnavailable("Account change could not finish. Try signing in again; pending changes will be retried.")
}
