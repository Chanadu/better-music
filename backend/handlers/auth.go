package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type requestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func generateJWT(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	return token.SignedString([]byte(config.Conf.JWTSecret))
}

func AuthRegister(w http.ResponseWriter, r *http.Request) {
	var body requestBody

	slog.Info("Registering user with email: " + body.Email)

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON"+err.Error()))
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

	user, err := models.CreateUser(body.Email, string(passwordHash))
	if err != nil {
		writeJSON(w, http.StatusConflict, apiError("email already in use, err: "+err.Error()))
		return
	}

	token, err := generateJWT(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate token"))
	}

	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

func AuthLogin(w http.ResponseWriter, r *http.Request) {
	var body requestBody

	slog.Info("Logging in user with email: " + body.Email)

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON"))
		return
	}

	user, err := models.GetUserByEmail(body.Email)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError("invalid email or password"))
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError("invalid email or password"))
		return
	}

	token, err := generateJWT(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to generate token"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
