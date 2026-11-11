package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/models"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword/epmodels"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/supertokens"
)

func (s *Service) PasswordResetLink(ctx context.Context, userID string) (string, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}

	defer tx.Rollback()

	identity, disabled, accountChangePending, err := models.LockPasswordRecoveryAccount(ctx, tx, userID)
	if err != nil {
		return "", err
	}

	if disabled {
		return "", ErrAccountDisabled
	}

	if !identity.Valid || accountChangePending {
		return "", errors.New("account must be mapped and pending account changes completed before generating a reset link")
	}

	result, err := emailpassword.CreateResetPasswordLink(publicTenant, identity.String)
	if err != nil {
		return "", err
	}

	if result.OK == nil {
		return "", errors.New("Core identity unavailable; reset link was not generated")
	}

	return result.OK.Link, nil
}

func (s *Service) resetPasswordAndRevokeSessions(token, password, tenant string, ctx supertokens.UserContext) (epmodels.ResetPasswordUsingTokenResponse, error) {
	result, err := (*s.raw.ResetPasswordUsingToken)(token, password, tenant, ctx)
	if err != nil || result.OK == nil {
		return result, err
	}

	if result.OK.UserId == nil {
		return epmodels.ResetPasswordUsingTokenResponse{}, errors.New("Core did not return the password reset user ID")
	}

	userID := *result.OK.UserId
	revocationCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := models.MarkSessionRevocationPending(revocationCtx, s.DB, userID); err != nil {
		slog.WarnContext(revocationCtx, "record pending session revocation failed", "user_id", userID, logger.ErrorDetails(err))
		if _, revokeErr := session.RevokeAllSessionsForUser(userID, nil, ctx); revokeErr != nil {
			return epmodels.ResetPasswordUsingTokenResponse{}, fmt.Errorf("password reset succeeded but session revocation failed: %w", errors.Join(err, revokeErr))
		}

		return result, nil
	}

	if err := s.Reconcile(revocationCtx, userID); err != nil {
		slog.WarnContext(revocationCtx, "session revocation will be retried", "user_id", userID, logger.ErrorDetails(err))
	}

	return result, nil
}
