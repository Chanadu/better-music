package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/models"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword/epmodels"
	"github.com/supertokens/supertokens-golang/supertokens"
)

func (s *Service) signin(email, password, tenant string, uc supertokens.UserContext) (epmodels.SignInResponse, error) {
	response, err := (*s.raw.SignIn)(email, password, tenant, uc)

	if err != nil || response.OK == nil {
		return response, err
	}

	userID := response.OK.User.ID
	if request := supertokens.GetRequestFromUserContext(uc); request != nil {
		if expected := request.Header.Get("X-Better-Music-Account"); expected != "" && expected != userID {
			return epmodels.SignInResponse{WrongCredentialsError: &struct{}{}}, nil
		}
	}

	ctx := context.Background()
	err = models.CheckActiveAuthAccount(ctx, s.DB, userID)

	if errors.Is(err, sql.ErrNoRows) {
		email = strings.ToLower(email)
		err = s.withEmailLock(ctx, email, func() error {
			if err := models.CheckActiveAuthAccount(ctx, s.DB, userID); !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			id, err := models.PendingSignupAccount(ctx, s.DB, email)
			if err != nil {
				return err
			}

			return models.CompleteSignup(ctx, s.DB, id, userID)
		})
	}

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(ctx, "resolve application account during sign-in failed", "user_id", userID, logger.ErrorDetails(err))
		}
		return epmodels.SignInResponse{WrongCredentialsError: &struct{}{}}, nil
	}

	if err = s.Reconcile(ctx, userID); err != nil {
		return epmodels.SignInResponse{}, err
	}

	return response, nil
}
