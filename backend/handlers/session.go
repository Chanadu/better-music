package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
	"github.com/danielgtaylor/huma/v2"
)

type SessionResponse struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	SyncEpoch string `json:"sync_epoch"`
}

func (h *Handler) registerSession(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "getSession",
		Method:      "GET",
		Path:        "/api/session",
		Responses:   authenticatedResponses(api),
	}, h.sessionInfo)
}

func (h *Handler) sessionInfo(ctx context.Context, _ *struct{}) (*jsonOutput[SessionResponse], error) {
	id, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("unauthorized")
	}
	email, epoch, err := models.ActiveAccountSession(ctx, h.Database, id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(ctx, "load account session failed", "user_id", id, logger.ErrorDetails(err))
		}
		return nil, huma.Error503ServiceUnavailable("account unavailable")
	}
	return &jsonOutput[SessionResponse]{Body: SessionResponse{UserID: id, Email: email, SyncEpoch: epoch}}, nil
}
