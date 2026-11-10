package handlers

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/lib/pq"
)

func TestNormalizeEmail(test *testing.T) {
	for _, testCase := range []struct {
		input, out string
	}{
		{" User@Example.COM ", "user@example.com"},
		{"bad", ""},
		{"Name <user@example.com>", ""},
		{"", ""},
		{"a@b.com", "a@b.com"},
	} {
		got, isValidEmail := normalizeEmail(testCase.input)
		if got != testCase.out || isValidEmail != (testCase.out != "") {
			test.Errorf("%q: %q %v", testCase.input, got, isValidEmail)
		}
	}
}

func TestJSONAndErrors(test *testing.T) {
	responseRecorder := httptest.NewRecorder()
	writeJSON(responseRecorder, 201, apiMessage("created"))
	if responseRecorder.Code != 201 || responseRecorder.Body.String() != `{"message":"created"}`+"\n" {
		test.Fatal(responseRecorder)
	}

	responseRecorder = httptest.NewRecorder()
	writeJSON(responseRecorder, 200, make(chan int))
	if responseRecorder.Code != 500 {
		test.Fatal(responseRecorder)
	}

	databaseError := fmt.Errorf("wrapped: %w", &pq.Error{Code: "23505", Constraint: "users_email_key"})
	if !isPostgresError(databaseError, "23505") || !isPostgresConstraintError(databaseError, "23505", "users_email_key") ||
		isPostgresConstraintError(databaseError, "23505", "other") ||
		isPostgresError(errors.New("oops"), "23505") {
		test.Error("postgres error classification")
	}

}

func TestRefreshTokens(test *testing.T) {
	firstRefreshToken, err := generateRefreshToken()
	if err != nil {
		test.Fatal(err)
	}

	secondRefreshToken, err := generateRefreshToken()
	if err != nil {
		test.Fatal(err)
	}

	if len(firstRefreshToken) != 43 || firstRefreshToken == secondRefreshToken || len(hashRefreshToken(firstRefreshToken)) != 64 ||
		hashRefreshToken(firstRefreshToken) == hashRefreshToken(secondRefreshToken) {
		test.Fatal("invalid refresh tokens")
	}

	if hashRefreshToken("abc") != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		test.Fatal("wrong SHA256")
	}
}
