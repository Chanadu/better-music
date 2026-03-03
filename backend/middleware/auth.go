package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Chanadu/better-music/config"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "userID"

func httpError(w http.ResponseWriter, msg string, status int) {
	http.Error(w, fmt.Sprintf(`{"error":"%s"}`, msg), status)
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			httpError(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			httpError(w, "invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		token, err := parseJWT(tokenString)
		if err != nil || !token.Valid {
			httpError(w, "invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["user_id"] == nil {
			httpError(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		userID := int(claims["user_id"].(float64))
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func parseJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}

		return []byte(config.Conf.JWTSecret), nil
	})

}

func GetUserID(r *http.Request) int {
	return r.Context().Value(userIDKey).(int)
}
