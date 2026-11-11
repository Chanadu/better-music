package auth

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/logger"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword/epmodels"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"
	"github.com/supertokens/supertokens-golang/supertokens"
)

const publicTenant = "public"

type Service struct {
	DB  *sql.DB
	raw epmodels.RecipeInterface
}

func Init(authConfig config.AuthConfig, database *sql.DB) (*Service, error) {
	service := &Service{DB: database}
	base, websiteBase := "/api/auth", "/auth"
	telemetry := false
	csrf := "VIA_CUSTOM_HEADER"

	err := supertokens.Init(supertokens.TypeInput{
		Supertokens: &supertokens.ConnectionInfo{
			ConnectionURI: authConfig.CoreURI,
			APIKey:        authConfig.APIKey,
		},
		AppInfo: supertokens.AppInfo{
			AppName:         "Better Music",
			APIDomain:       authConfig.APIDomain,
			WebsiteDomain:   authConfig.WebsiteDomain,
			APIBasePath:     &base,
			WebsiteBasePath: &websiteBase,
		},
		Telemetry: &telemetry,
		OnSuperTokensAPIError: func(err error, httpRequest *http.Request, writer http.ResponseWriter) {
			slog.ErrorContext(httpRequest.Context(), "SuperTokens API failed", "method", httpRequest.Method, logger.ErrorDetails(err))
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		},
		RecipeList: []supertokens.Recipe{
			session.Init(&sessmodels.TypeInput{
				AntiCsrf:               &csrf,
				GetTokenTransferMethod: cookieTransferMethod,
				Override:               &sessmodels.OverrideStruct{APIs: overrideSessionAPIs},
			}),
			emailpassword.Init(&epmodels.TypeInput{
				Override: &epmodels.OverrideStruct{
					Functions: service.overrideEmailPasswordFunctions,
					APIs:      overrideEmailPasswordAPIs,
				},
			}),
		},
	})

	return service, err
}

func cookieTransferMethod(*http.Request, bool, supertokens.UserContext) sessmodels.TokenTransferMethod {
	return sessmodels.CookieTransferMethod
}

func overrideSessionAPIs(api sessmodels.APIInterface) sessmodels.APIInterface {
	signOut := *api.SignOutPOST

	checkOwner := func(session sessmodels.SessionContainer, options sessmodels.APIOptions, ctx supertokens.UserContext) (sessmodels.SignOutPOSTResponse, error) {
		expected := options.Req.Header.Get("X-Better-Music-Account")

		if expected == "" || expected != session.GetUserID() {
			return sessmodels.SignOutPOSTResponse{
				GeneralError: &supertokens.GeneralErrorResponse{Message: "Account changed. Sign in to the original account."},
			}, nil
		}

		return signOut(session, options, ctx)
	}

	api.SignOutPOST = &checkOwner
	return api
}

func (s *Service) overrideEmailPasswordFunctions(original epmodels.RecipeInterface) epmodels.RecipeInterface {
	s.raw = original

	signup, signin, reset := s.signup, s.signin, s.resetPasswordAndRevokeSessions
	original.SignUp = &signup
	original.SignIn = &signin
	original.ResetPasswordUsingToken = &reset

	return original
}

func overrideEmailPasswordAPIs(api epmodels.APIInterface) epmodels.APIInterface {
	api.EmailExistsGET = nil
	api.GeneratePasswordResetTokenPOST = nil

	return api
}
