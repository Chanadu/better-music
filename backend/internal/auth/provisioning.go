package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/models"
	"github.com/danielgtaylor/huma/v2"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword/epmodels"
	"github.com/supertokens/supertokens-golang/supertokens"
)

func (s *Service) withEmailLock(ctx context.Context, email string, fn func() error) error {
	conn, err := s.DB.Conn(ctx)

	if err != nil {
		return err
	}

	defer conn.Close()
	if err = models.LockAuthEmail(ctx, conn, email); err != nil {
		return err
	}

	defer func() {
		if err := models.UnlockAuthEmail(context.Background(), conn, email); err != nil {
			slog.ErrorContext(ctx, "release auth email lock failed", logger.ErrorDetails(err))
		}
	}()

	return fn()
}

func (s *Service) createIdentity(email, password, tenant string, uc supertokens.UserContext) (epmodels.User, error) {
	created, err := (*s.raw.SignUp)(email, password, tenant, uc)

	if err != nil {
		return epmodels.User{}, err
	}

	if created.OK != nil {
		return created.OK.User, nil
	}

	if created.EmailAlreadyExistsError == nil {
		return epmodels.User{}, errors.New("identity creation failed")
	}

	verified, err := (*s.raw.SignIn)(email, password, tenant, uc)
	if err != nil {
		return epmodels.User{}, err
	}

	if verified.OK == nil {
		return epmodels.User{}, huma.Error409Conflict("email already in use")
	}

	return verified.OK.User, nil
}

func (s *Service) signup(email, password, tenant string, uc supertokens.UserContext) (epmodels.SignUpResponse, error) {
	if tenant != publicTenant {
		return epmodels.SignUpResponse{}, errors.New("unsupported tenant")
	}

	email = strings.ToLower(strings.TrimSpace(email))

	ctx := context.Background()

	var response epmodels.SignUpResponse
	err := s.withEmailLock(ctx, email, func() error {
		userID, err := models.ReserveSignupAccount(ctx, s.DB, email)

		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error409Conflict("email already in use")
		}

		if err != nil {
			return err
		}

		user, err := s.createIdentity(email, password, tenant, uc)
		if err != nil {
			return err
		}

		if err = models.CompleteSignup(ctx, s.DB, userID, user.ID); err != nil {
			return err
		}

		response.OK = &struct{ User epmodels.User }{User: user}

		return nil
	})

	var statusError huma.StatusError
	if errors.As(err, &statusError) && statusError.GetStatus() == http.StatusConflict {
		return epmodels.SignUpResponse{EmailAlreadyExistsError: &struct{}{}}, nil
	}

	return response, err
}
