package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthRequest struct {
	Email    string `json:"email" example:"user@example.com" validate:"required"`
	Password string `json:"password" example:"password123" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"your-refresh-token" validate:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." validate:"required"`
	RefreshToken string `json:"refresh_token" example:"M2vK0r5mMWhM6Lqg0k4cCso1WW1wmX7uCUNB3vV9Q-U" validate:"required"`
	TokenType    string `json:"token_type" example:"Bearer" validate:"required"`
	ExpiresIn    int64  `json:"expires_in" example:"900" validate:"required"`
	UserID       int    `json:"user_id" example:"42" validate:"required"`
}

func (h *Handler) generateJWT(userID int) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.AuthClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(h.Config.AccessTTL)),
		},
	})

	return token.SignedString([]byte(h.Config.JWTSecret))
}

func generateRefreshToken() (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (h *Handler) cleanupOldRefreshTokens() {
	deleted, err := models.CleanupRefreshTokens(h.Database)
	if err != nil {
		slog.Warn("failed to cleanup old refresh tokens", "error", err)
		return
	}

	if deleted > 0 {
		slog.Info("cleaned up old refresh tokens", "deleted", deleted)
	}
}

func (h *Handler) issueTokens(userID int) (*TokenResponse, error) {
	h.cleanupOldRefreshTokens()

	accessToken, err := h.generateJWT(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshExpiresAt := time.Now().Add(h.Config.RefreshTTL)
	err = models.CreateRefreshToken(h.Database, userID, hashRefreshToken(refreshToken), refreshExpiresAt)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(h.Config.AccessTTL / time.Second),
		UserID:       userID,
	}, nil
}

func (h *Handler) AuthRegister(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "POST /api/auth/register", "method", r.Method, "path", r.URL.Path)

	var body AuthRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	email, validEmail := normalizeEmail(body.Email)
	if !validEmail || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, apiError("valid email and password required"))
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to hash password"))
		return
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate refresh token"))
		return
	}

	user, err := models.CreateUserWithRefreshToken(
		h.Database,
		email,
		string(passwordHash),
		hashRefreshToken(refreshToken),
		time.Now().Add(h.Config.RefreshTTL),
	)
	if err != nil {
		if isPostgresConstraintError(err, "23505", "users_email_key") {
			writeJSON(w, http.StatusConflict, apiError("email already in use"))
			return
		}

		writeJSON(w, http.StatusInternalServerError, apiError("failed to create account"))
		return
	}

	accessToken, err := h.generateJWT(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate access token"))
		return
	}

	writeJSON(w, http.StatusCreated, &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(h.Config.AccessTTL / time.Second),
		UserID:       user.ID,
	})
}

func (h *Handler) AuthLogin(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "POST /api/auth/login", "method", r.Method, "path", r.URL.Path)

	var body AuthRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	email, validEmail := normalizeEmail(body.Email)
	if !validEmail {
		writeJSON(w, http.StatusUnauthorized, apiError("invalid email or password"))
		return
	}

	user, err := models.GetUserByEmail(h.Database, email)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError("invalid email or password"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError("invalid email or password"))
		return
	}

	tokens, err := h.issueTokens(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate tokens"))
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

func (h *Handler) AuthRefresh(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "POST /api/auth/refresh", "method", r.Method, "path", r.URL.Path)

	var body RefreshTokenRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	if body.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, apiError("refresh token required"))
		return
	}

	replacementToken, err := generateRefreshToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate refresh token: "+err.Error()))
		return
	}

	userID, err := models.RotateRefreshToken(
		h.Database,
		hashRefreshToken(body.RefreshToken),
		hashRefreshToken(replacementToken),
		time.Now().Add(h.Config.RefreshTTL),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusUnauthorized, apiError("invalid, expired, or revoked refresh token"))
			return
		}

		writeJSON(w, http.StatusInternalServerError, apiError("failed to rotate refresh token: "+err.Error()))
		return
	}

	accessToken, err := h.generateJWT(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate access token: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: replacementToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(h.Config.AccessTTL / time.Second),
		UserID:       userID,
	})
}

func (h *Handler) AuthLogout(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "POST /api/auth/logout", "method", r.Method, "path", r.URL.Path)

	var body RefreshTokenRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	if body.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, apiError("refresh token required"))
		return
	}

	storedToken, err := models.GetRefreshTokenByHash(h.Database, hashRefreshToken(body.RefreshToken))
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, apiMessage("logged out"))
			return
		}

		writeJSON(w, http.StatusInternalServerError, apiError("failed to load refresh token: "+err.Error()))
		return
	}

	err = models.RevokeRefreshToken(h.Database, storedToken.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to revoke refresh token: "+err.Error()))
		return
	}

	h.cleanupOldRefreshTokens()

	writeJSON(w, http.StatusOK, apiMessage("logged out"))
}
