package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Chanadu/better-music/models"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type AccountResponse struct {
	Email string `json:"email" validate:"required"`
}

type UpdateEmailRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" validate:"required"`
}

func (h *Handler) authenticatedUser(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	userID, ok := getUserID(w, r)
	if !ok {
		return nil, false
	}

	user, err := models.GetUserByID(h.Database, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to load account"))
		return nil, false
	}

	return user, true
}

func validPassword(user *models.User, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}

// GetAccount godoc
// @Summary Get the current account
// @Tags account
// @Produce json
// @Security Bearer
// @Success 200 {object} AccountResponse
// @Failure 401 {object} ApiErrorResponse
// @Failure 500 {object} ApiErrorResponse
// @Router /api/account [get]
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	user, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, AccountResponse{Email: user.Email})
}

// UpdateAccountEmail godoc
// @Summary Change the current account email
// @Tags account
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body UpdateEmailRequest true "New email and current password"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ApiErrorResponse
// @Failure 401 {object} ApiErrorResponse
// @Failure 409 {object} ApiErrorResponse
// @Failure 500 {object} ApiErrorResponse
// @Router /api/account/email [put]
func (h *Handler) UpdateAccountEmail(w http.ResponseWriter, r *http.Request) {
	var body UpdateEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON"))
		return
	}

	body.Email = strings.TrimSpace(body.Email)
	if body.Email == "" || body.Password == "" || !strings.Contains(body.Email, "@") {
		writeJSON(w, http.StatusBadRequest, apiError("valid email and current password required"))
		return
	}

	user, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	if !validPassword(user, body.Password) {
		writeJSON(w, http.StatusUnauthorized, apiError("current password is incorrect"))
		return
	}
	if err := models.UpdateUserEmail(h.Database, user.ID, body.Email); err != nil {
		var pqError *pq.Error
		if errors.As(err, &pqError) && pqError.Code == "23505" {
			writeJSON(w, http.StatusConflict, apiError("email already in use"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("failed to update email"))
		return
	}

	writeJSON(w, http.StatusOK, apiMessage("email updated"))
}

// UpdateAccountPassword godoc
// @Summary Change the current account password
// @Tags account
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body UpdatePasswordRequest true "Current and new passwords"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ApiErrorResponse
// @Failure 401 {object} ApiErrorResponse
// @Failure 500 {object} ApiErrorResponse
// @Router /api/account/password [put]
func (h *Handler) UpdateAccountPassword(w http.ResponseWriter, r *http.Request) {
	var body UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON"))
		return
	}
	if body.CurrentPassword == "" || len(body.NewPassword) < 4 {
		writeJSON(w, http.StatusBadRequest, apiError("current password and a new password of at least 4 characters required"))
		return
	}

	user, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	if !validPassword(user, body.CurrentPassword) {
		writeJSON(w, http.StatusUnauthorized, apiError("current password is incorrect"))
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to hash password"))
		return
	}
	if err = models.UpdateUserPassword(h.Database, user.ID, string(passwordHash)); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to update password"))
		return
	}
	writeJSON(w, http.StatusOK, apiMessage("password updated"))
}

// DeleteAccount godoc
// @Summary Permanently delete the current account
// @Tags account
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body DeleteAccountRequest true "Current password"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ApiErrorResponse
// @Failure 401 {object} ApiErrorResponse
// @Failure 500 {object} ApiErrorResponse
// @Router /api/account [delete]
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	var body DeleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, apiError("current password required"))
		return
	}

	user, ok := h.authenticatedUser(w, r)
	if !ok {
		return
	}
	if !validPassword(user, body.Password) {
		writeJSON(w, http.StatusUnauthorized, apiError("current password is incorrect"))
		return
	}

	if err := models.DeleteUser(h.Database, user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to delete account"))
		return
	}

	writeJSON(w, http.StatusOK, apiMessage("account deleted"))
}
