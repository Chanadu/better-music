package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestGetUserID(test *testing.T) {
	for _, value := range []any{nil, 0, -1, "", "auth-42"} {
		httpRequest := httptest.NewRequest("GET", "/", nil)
		if value != nil {
			httpRequest = httpRequest.WithContext(context.WithValue(httpRequest.Context(), userIDKey, value))
		}
		id, ok := GetUserID(httpRequest)
		if value == "auth-42" {
			if !ok || id != "auth-42" {
				test.Fatal("owner context was not read")
			}
		} else if ok {
			test.Fatalf("accepted invalid owner %v", value)
		}
	}
}
