package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/Chanadu/better-music/models"
	"github.com/danielgtaylor/huma/v2"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword"
	"github.com/supertokens/supertokens-golang/recipe/session"
)

type AccountOperation string

const (
	ChangeEmail    AccountOperation = "email"
	ChangePassword AccountOperation = "password"
	DeleteAccount  AccountOperation = "delete"
)

func (s *Service) Change(ctx context.Context, userID string, kind AccountOperation, password, value string) error {
	if password == "" {
		return huma.Error400BadRequest("current password required")
	}

	if kind == ChangePassword {
		if err := validateNewPassword(value); err != nil {
			return err
		}
	}

	if kind != ChangeEmail {
		return s.changeAccount(ctx, userID, kind, password, value)
	}

	email, valid := normalizeEmail(value)
	if !valid {
		return huma.Error400BadRequest("valid email required")
	}

	return s.withEmailLock(ctx, email, func() error {
		return s.changeAccount(ctx, userID, kind, password, email)
	})
}

func (s *Service) changeAccount(ctx context.Context, userID string, kind AccountOperation, password, value string) error {
	if err := s.Reconcile(ctx, userID); err != nil {
		return err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := models.LockActiveAuthAccount(ctx, tx, userID); err != nil {
		return err
	}

	identity, err := emailpassword.GetUserByID(userID)
	if err != nil {
		return err
	}

	if identity == nil {
		return huma.Error403Forbidden("account is disabled")
	}

	verified, err := (*s.raw.SignIn)(identity.Email, password, publicTenant, nil)
	if err != nil {
		return err
	}

	if verified.OK == nil || verified.OK.User.ID != userID {
		return huma.Error400BadRequest("current password is incorrect")
	}

	var targetEmail *string
	switch kind {
	case ChangePassword:
		if err := changePassword(userID, value); err != nil {
			return err
		}

		return tx.Commit()

	case ChangeEmail:
		if value == identity.Email {
			return huma.Error400BadRequest("invalid account input: enter a different email address")
		}

		emailInUse, err := models.AuthEmailInUse(ctx, tx, value, userID)
		if err != nil {
			return err
		}

		if emailInUse {
			return huma.Error409Conflict("email already in use")
		}

		targetEmail = &value

	case DeleteAccount:
		if err = models.DisableAuthAccount(ctx, tx, userID); err != nil {
			return err
		}

	default:
		return errors.New("unknown account operation")
	}

	if err = models.RecordAuthAccountOperation(ctx, tx, userID, string(kind), targetEmail); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return s.Reconcile(ctx, userID)
}

func validateNewPassword(newPassword string) error {
	recipe, err := emailpassword.GetRecipeInstanceOrThrowError()
	if err != nil {
		return err
	}

	for _, field := range recipe.Config.SignUpFeature.FormFields {
		if field.ID != "password" {
			continue
		}

		if reason := field.Validate(newPassword, publicTenant); reason != nil {
			return huma.Error400BadRequest("invalid account input: " + *reason)
		}
	}

	return nil
}

func changePassword(userID, newPassword string) error {
	if _, err := session.RevokeAllSessionsForUser(userID, nil); err != nil {
		return err
	}

	result, err := emailpassword.UpdateEmailOrPassword(userID, nil, &newPassword, nil, nil)
	if err != nil {
		return err
	}

	if result.PasswordPolicyViolatedError != nil {
		return errors.New(result.PasswordPolicyViolatedError.FailureReason)
	}

	if result.OK == nil {
		return errors.New("password could not be changed")
	}

	return nil
}

func normalizeEmail(email string) (string, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)

	if err != nil || address.Address != email {
		return "", false
	}

	return email, true
}
