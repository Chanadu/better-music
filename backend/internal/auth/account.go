package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Chanadu/better-music/models"
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
	if kind == ChangeEmail {
		value = strings.ToLower(strings.TrimSpace(value))

		return s.withEmailLock(ctx, value, func() error {
			return s.changeAccount(ctx, userID, kind, password, value)
		})
	}

	return s.changeAccount(ctx, userID, kind, password, value)
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
		return ErrAccountDisabled
	}

	verified, err := (*s.raw.SignIn)(identity.Email, password, publicTenant, nil)
	if err != nil {
		return err
	}

	if verified.OK == nil || verified.OK.User.ID != userID {
		return ErrCurrentPassword
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
			return fmt.Errorf("%w: enter a different email address", ErrInvalidInput)
		}

		emailInUse, err := models.AuthEmailInUse(ctx, tx, value, userID)
		if err != nil {
			return err
		}

		if emailInUse {
			return ErrEmailExists
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

func changePassword(userID, newPassword string) error {
	recipe, err := emailpassword.GetRecipeInstanceOrThrowError()
	if err != nil {
		return err
	}

	for _, field := range recipe.Config.SignUpFeature.FormFields {
		if field.ID != "password" {
			continue
		}

		if reason := field.Validate(newPassword, publicTenant); reason != nil {
			return fmt.Errorf("%w: %s", ErrInvalidInput, *reason)
		}
	}

	if _, err = session.RevokeAllSessionsForUser(userID, nil); err != nil {
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
