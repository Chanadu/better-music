package handlers

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Chanadu/better-music/models"
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

func TestParseIfMatch(test *testing.T) {
	for _, testCase := range []struct {
		input string
		want  int
		bad   bool
	}{
		{"", 0, false},
		{"1", 1, false},
		{`"42"`, 42, false},
		{"2147483647", 2147483647, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"2147483648", 0, true},
		{"*", 0, true},
		{`W/"1"`, 0, true},
		{"1,2", 0, true},
		{" 1 ", 0, true},
		{`"1`, 0, true},
	} {
		test.Run(testCase.input, func(test *testing.T) {
			httpRequest := httptest.NewRequest("PUT", "/", nil)
			httpRequest.Header.Set("If-Match", testCase.input)
			got, err := parseIfMatch(httpRequest)
			if (err != nil) != testCase.bad || got != testCase.want {
				test.Fatalf("got %d %v", got, err)
			}
		})
	}

	httpRequest := httptest.NewRequest("PUT", "/", nil)
	httpRequest.Header.Add("If-Match", "1")
	httpRequest.Header.Add("If-Match", "2")
	if _, err := parseIfMatch(httpRequest); err == nil {
		test.Error("duplicate header accepted")
	}
}

func TestMutationFields(test *testing.T) {
	for _, testCase := range []struct {
		body   string
		entity models.LibraryEntity
		want   map[string]any
	}{
		{
			`{"name":" Artist ","cover_url":null}`,
			models.ArtistEntity,
			map[string]any{"name": "Artist", "cover_url": nil},
		},
		{
			`{"artist_id":1,"title":" Album ","listened":true,"rating":10,"year":-1,"comment":null}`,
			models.AlbumEntity,
			map[string]any{
				"artist_id": 1,
				"title":     "Album",
				"listened":  true,
				"rating":    10,
				"year":      -1,
				"comment":   nil,
			},
		},
		{`{}`, models.AlbumEntity, map[string]any{}},
	} {
		got, err := decodeMutationFields([]byte(testCase.body), testCase.entity)
		if err != nil || !reflect.DeepEqual(got, testCase.want) {
			test.Fatalf("%s: %v %v", testCase.body, got, err)
		}
	}

	for _, body := range []string{
		`null`,
		`[]`,
		`1`,
		`{`,
		`{} {}`,
		`{"unknown":1}`,
		`{"title":null}`,
		`{"title":" "}`,
		`{"title":1}`,
		`{"artist_id":null}`,
		`{"artist_id":0}`,
		`{"artist_id":1.5}`,
		`{"year":2147483648}`,
		`{"year":-2147483649}`,
		`{"rating":0}`,
		`{"rating":11}`,
		`{"listened":null}`,
		`{"listened":"true"}`,
		`{"comment":true}`,
	} {
		test.Run(body, func(test *testing.T) {
			if _, err := decodeMutationFields([]byte(body), models.AlbumEntity); err == nil {
				test.Fatal("accepted invalid body")
			}
		})
	}

	if _, err := decodeMutationFields([]byte(`{"name":null}`), models.ArtistEntity); err == nil {
		test.Error("null name accepted")
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

	response, err := encodeMutationResponse(mutationResponse{
		status:  200,
		version: 3,
		body:    apiMessage("ok"),
	})
	if err != nil || response.Headers["ETag"] != `"3"` {
		test.Fatalf("%+v %v", response, err)
	}

	if _, err := encodeMutationResponse(mutationResponse{body: make(chan int)}); err == nil {
		test.Error("encoding should fail")
	}

	databaseError := fmt.Errorf("wrapped: %w", &pq.Error{Code: "23505", Constraint: "users_email_key"})
	if !isPostgresError(databaseError, "23505") || !isPostgresConstraintError(databaseError, "23505", "users_email_key") ||
		isPostgresConstraintError(databaseError, "23505", "other") ||
		isPostgresError(errors.New("oops"), "23505") {
		test.Error("postgres error classification")
	}

	empty := ""
	full := "x"
	if !isEmpty(nil) || !isEmpty(&empty) || isEmpty(&full) {
		test.Error("isEmpty")
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
