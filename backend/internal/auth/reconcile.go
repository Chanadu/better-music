package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Chanadu/better-music/models"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/supertokens"
)

func (s *Service) Reconcile(ctx context.Context, userID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	disabled, revocationPending, err := models.LockAuthAccount(ctx, tx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if revocationPending {
		if _, err := session.RevokeAllSessionsForUser(userID, nil); err != nil {
			return err
		}
		if err := models.ClearSessionRevocation(ctx, tx, userID); err != nil {
			return err
		}
	}

	kind, targetEmail, err := models.PendingAuthAccountOperation(ctx, tx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		if disabled {
			return ErrAccountDisabled
		}
		if revocationPending {
			return tx.Commit()
		}
		return nil
	}

	if err != nil {
		return err
	}

	if AccountOperation(kind) == DeleteAccount {
		if err = supertokens.DeleteUser(userID); err != nil {
			return err
		}

		if err = models.DeleteAuthAccount(ctx, tx, userID); err != nil {
			return err
		}

		return tx.Commit()
	}

	result, err := emailpassword.UpdateEmailOrPassword(userID, &targetEmail.String, nil, nil, nil)
	if err != nil {
		return err
	}

	var operationErr error

	switch {
	case result.EmailAlreadyExistsError != nil:
		operationErr = ErrEmailExists

	case result.OK == nil:
		return errors.New("pending email update could not complete")

	default:
		if _, err = session.RevokeAllSessionsForUser(userID, nil); err != nil {
			return err
		}

		if err = models.UpdateAuthAccountEmail(ctx, tx, userID, targetEmail.String); err != nil {
			return err
		}
	}

	if err = models.ClearAuthAccountOperation(ctx, tx, userID); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return operationErr
}

func (s *Service) ReconcilePending(ctx context.Context) error {
	ids, err := models.PendingAuthAccountIDs(ctx, s.DB)

	if err != nil {
		return err
	}

	var failures []error

	for _, id := range ids {
		if err := s.Reconcile(ctx, id); err != nil {
			failures = append(failures, fmt.Errorf("account %s: %w", id, err))
		}
	}

	return errors.Join(failures...)
}
