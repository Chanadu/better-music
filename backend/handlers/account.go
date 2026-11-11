package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
	"github.com/danielgtaylor/huma/v2"
)

type AccountResponse struct {
	Email string `json:"email"`
}

func (h *Handler) registerAccount(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "getAccount",
		Method:      http.MethodGet,
		Path:        "/api/account",
		Summary:     "Get the signed-in account",
		Responses:   authenticatedResponses(api),
	}, h.getAccount)

	h.registerAccountChanges(api)
}

func (h *Handler) getAccount(ctx context.Context, _ *struct{}) (*jsonOutput[AccountResponse], error) {
	id, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("unauthorized")
	}

	email, err := models.ActiveAccountEmail(ctx, h.Database, id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(ctx, "load account email failed", "user_id", id, logger.ErrorDetails(err))
		}

		return nil, huma.Error503ServiceUnavailable("account unavailable")
	}

	return &jsonOutput[AccountResponse]{Body: AccountResponse{Email: email}}, nil
}
