package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/models"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"
)

type contextKey string

const userIDKey contextKey = "userID"

func httpError(writer http.ResponseWriter, httpRequest *http.Request, msg string, status int) {
	if status < 500 {
		slog.DebugContext(httpRequest.Context(), "auth middleware rejected request", "status", status, "reason", msg, "method", httpRequest.Method)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	_ = json.NewEncoder(writer).Encode(map[string]string{"error": msg})
}

func GetUserID(httpRequest *http.Request) (string, bool) {
	userID, ok := httpRequest.Context().Value(userIDKey).(string)

	if !ok || userID == "" {
		return "", false
	}

	return userID, true
}

func ResolveAuthIdentity(next http.Handler, database *sql.DB, identity func(*http.Request) string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, httpRequest *http.Request) {
		authID := identity(httpRequest)
		if authID == "" {
			httpError(writer, httpRequest, "missing session", http.StatusUnauthorized)
			return
		}

		err := models.CheckActiveAuthAccount(httpRequest.Context(), database, authID)
		if errors.Is(err, sql.ErrNoRows) {
			httpError(writer, httpRequest, "identity is not mapped to an application account", http.StatusForbidden)
			return
		}

		if err != nil {
			slog.ErrorContext(httpRequest.Context(), "resolve auth identity failed", "user_id", authID, logger.ErrorDetails(err))
			httpError(writer, httpRequest, "account lookup unavailable", http.StatusServiceUnavailable)
			return
		}

		if expected := httpRequest.Header.Get("X-Better-Music-Account"); expected != "" && expected != authID {
			httpError(writer, httpRequest, "account changed; sign in to the original account", http.StatusConflict)
			return
		}

		next.ServeHTTP(writer, httpRequest.WithContext(context.WithValue(httpRequest.Context(), userIDKey, authID)))
	})
}

func SuperTokensAuth(next http.Handler, database *sql.DB, options *sessmodels.VerifySessionOptions) http.Handler {
	mapped := ResolveAuthIdentity(next, database, func(httpRequest *http.Request) string {
		authSession := session.GetSessionFromRequestContext(httpRequest.Context())

		if authSession == nil {
			return ""
		}

		return authSession.GetUserID()
	})

	return session.VerifySession(options, mapped.ServeHTTP)
}
