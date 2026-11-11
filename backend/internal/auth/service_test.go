package auth

import (
	"errors"
	"testing"

	"github.com/Chanadu/better-music/internal/testdb"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword/epmodels"
	"github.com/supertokens/supertokens-golang/supertokens"
)

func TestSignupRecoversLostCoreResponse(test *testing.T) {
	db := testdb.Open(test)
	const email = "retry@example.test"
	const password = "Password-123!"
	signup := func(string, string, string, supertokens.UserContext) (epmodels.SignUpResponse, error) {
		return epmodels.SignUpResponse{}, errors.New("lost successful Core response")
	}
	signin := func(email, password, tenant string, uc supertokens.UserContext) (epmodels.SignInResponse, error) {
		return epmodels.SignInResponse{OK: &struct{ User epmodels.User }{
			User: epmodels.User{ID: "returned-core-id", Email: email},
		}}, nil
	}
	service := Service{DB: db, raw: epmodels.RecipeInterface{SignUp: &signup, SignIn: &signin}}
	if _, err := service.signup(email, password, "public", nil); err == nil {
		test.Fatal("lost response unexpectedly succeeded")
	}
	result, err := service.signin(email, password, "public", nil)
	if err != nil || result.OK == nil {
		test.Fatalf("sign-in could not finish signup: %v", err)
	}
	var id string
	if err := db.QueryRow(`SELECT id FROM users WHERE email = $1`, email).Scan(&id); err != nil || id != "returned-core-id" {
		test.Fatalf("signup did not save the Core ID: %s %v", id, err)
	}
}

func TestSignupIntentFailureDoesNotLeaveAnAccount(test *testing.T) {
	db := testdb.Open(test)
	if _, err := db.Exec(`ALTER TABLE users ADD CONSTRAINT reject_signup CHECK (id NOT LIKE 'pending-%')`); err != nil {
		test.Fatal(err)
	}
	called := false
	signup := func(string, string, string, supertokens.UserContext) (epmodels.SignUpResponse, error) {
		called = true
		return epmodels.SignUpResponse{}, errors.New("unexpected Core call")
	}
	service := Service{DB: db, raw: epmodels.RecipeInterface{SignUp: &signup}}
	if _, err := service.signup("new@example.test", "Password-123!", "public", nil); err == nil {
		test.Fatal("failed intent was accepted")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&count); err != nil {
		test.Fatal(err)
	}
	if count != 0 || called {
		test.Fatal("failed recovery intent left an account or created a Core identity")
	}
}
