package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Chanadu/better-music/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthRequest represents the request body for auth endpoints
type AuthRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"password123"`
}

type TokenResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

func (h *Handler) generateJWT(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	return token.SignedString([]byte(h.Config.JWTSecret))
}

// AuthRegister godoc
// @Summary Register a new user
// @Description Create a new user account and receive a JWT token
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
		writeJSON(w, http.StatusConflict, apiError("email already in use, err: "+err.Error()))
		return
	}

	token, err := h.generateJWT(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate token"))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

// AuthLogin godoc
// @Summary Login user
// @Description Authenticate user with email and password to receive a JWT token
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

	token, err := h.generateJWT(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate token"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
