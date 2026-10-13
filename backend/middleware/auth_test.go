package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuth(test *testing.T) {
	future := jwt.NewNumericDate(time.Now().Add(time.Hour))
	past := jwt.NewNumericDate(time.Now().Add(-time.Hour))

	for _, testCase := range []struct {
		name, header, secret string
		userID               int
		expiry               *jwt.NumericDate
		method               jwt.SigningMethod
		want                 int
	}{
		{name: "missing", want: 401}, {
			name:   "basic",
			header: "Basic abc",
			want:   401,
		}, {
			name:   "lowercase",
			header: "bearer abc",
			want:   401,
		}, {
			name:   "malformed",
			header: "Bearer garbage",
			want:   401,
		},
		{
			name:   "valid",
			secret: "secret",
			userID: 42,
			expiry: future,
			method: jwt.SigningMethodHS256,
			want:   204,
		},
		{
			name:   "wrong secret",
			secret: "other",
			userID: 42,
			expiry: future,
			method: jwt.SigningMethodHS256,
			want:   401,
		},
		{
			name:   "expired",
			secret: "secret",
			userID: 42,
			expiry: past,
			method: jwt.SigningMethodHS256,
			want:   401,
		},
		{
			name:   "no expiry",
			secret: "secret",
			userID: 42,
			method: jwt.SigningMethodHS256,
			want:   401,
		},
		{
			name:   "zero user",
			secret: "secret",
			expiry: future,
			method: jwt.SigningMethodHS256,
			want:   401,
		},
		{
			name:   "negative user",
			secret: "secret",
			userID: -1,
			expiry: future,
			method: jwt.SigningMethodHS256,
			want:   401,
		},
		{
			name:   "unsigned",
			userID: 42,
			expiry: future,
			method: jwt.SigningMethodNone,
			want:   401,
		},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			header := testCase.header
			if testCase.method != nil {
				var key any = []byte(testCase.secret)
				if testCase.method == jwt.SigningMethodNone {
					key = jwt.UnsafeAllowNoneSignatureType
				}

				token, err := jwt.NewWithClaims(testCase.method, AuthClaims{
					UserID:           testCase.userID,
					RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: testCase.expiry},
				}).SignedString(key)
				if err != nil {
					test.Fatal(err)
				}

				header = "Bearer " + token
			}

			httpRequest := httptest.NewRequest("GET", "/", nil)
			httpRequest.Header.Set("Authorization", header)
			responseRecorder := httptest.NewRecorder()
			called := false
			Auth(http.HandlerFunc(func(responseWriter http.ResponseWriter, httpRequest *http.Request) {
				called = true
				userID, hasUserID := GetUserID(httpRequest)
				if !hasUserID || userID != 42 {
					test.Errorf("user=%d ok=%v", userID, hasUserID)
				}

				responseWriter.WriteHeader(204)
			}), "secret").ServeHTTP(responseRecorder, httpRequest)
			if responseRecorder.Code != testCase.want || called != (testCase.want == 204) {
				test.Fatalf("status=%d called=%v body=%s", responseRecorder.Code, called, responseRecorder.Body)
			}

			if testCase.want == 401 && responseRecorder.Header().Get("Content-Type") != "application/json" {
				test.Error("missing JSON content type")
			}
		})
	}

	if _, hasUserID := GetUserID(httptest.NewRequest("GET", "/", nil)); hasUserID {
		test.Error("unauthenticated context accepted")
	}
}
