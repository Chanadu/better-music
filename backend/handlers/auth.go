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

// AuthRequest represents the request body for auth endpoints
type AuthRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"password123"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"your-refresh-token"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"M2vK0r5mMWhM6Lqg0k4cCso1WW1wmX7uCUNB3vV9Q-U"`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"900"`
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
	}, nil
}

// AuthRegister godoc
// @Summary Register a new user
// @Description Create a new user account and receive access and refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body AuthRequest true "User credentials"
// @Success 201 {object} TokenResponse
// @Failure 400 {object} map[string]string "Invalid JSON or missing fields"
// @Failure 409 {object} map[string]string "Email already in use"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/auth/register [post]
func (h *Handler) AuthRegister(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "POST /api/auth/register", "method", r.Method, "path", r.URL.Path)
	var body AuthRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	if body.Email == "" || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, apiError("email and password required"))
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to hash password"))
		return
	}

	user, err := models.CreateUser(h.Database, body.Email, string(passwordHash))
	if err != nil {
		writeJSON(w, http.StatusConflict, apiError("email already in use"))
		return
	}

	tokens, err := h.issueTokens(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate tokens"))
		return
	}

	writeJSON(w, http.StatusCreated, tokens)
}

// AuthLogin godoc
// @Summary Login user
// @Description Authenticate user with email and password to receive access and refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body AuthRequest true "User credentials"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} map[string]string "Invalid JSON"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/auth/login [post]
func (h *Handler) AuthLogin(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "POST /api/auth/login", "method", r.Method, "path", r.URL.Path)
	var body AuthRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	user, err := models.GetUserByEmail(h.Database, body.Email)
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

// AuthRefresh godoc
// @Summary Refresh auth tokens
// @Description Exchange a valid refresh token for a new access token and refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} map[string]string "Invalid JSON or missing token"
// @Failure 401 {object} map[string]string "Invalid or expired refresh token"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/auth/refresh [post]
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

	storedToken, err := models.GetRefreshTokenByHash(h.Database, hashRefreshToken(body.RefreshToken))
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusUnauthorized, apiError("invalid refresh token"))
			return
		}

		writeJSON(w, http.StatusInternalServerError, apiError("failed to load refresh token: "+err.Error()))
		return
	}

	if storedToken.RevokedAt.Valid || time.Now().After(storedToken.ExpiresAt) {
		_ = models.RevokeRefreshToken(h.Database, storedToken.ID)

		writeJSON(w, http.StatusUnauthorized, apiError("refresh token expired or revoked"))
		return
	}

	err = models.RevokeRefreshToken(h.Database, storedToken.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to rotate refresh token: "+err.Error()))
		return
	}

	tokens, err := h.issueTokens(storedToken.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate tokens: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// AuthLogout godoc
// @Summary Logout user
// @Description Revoke a refresh token so it can no longer be used to mint new access tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid JSON or missing token"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/auth/logout [post]
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
			writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
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

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}
