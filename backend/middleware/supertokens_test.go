package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chanadu/better-music/internal/testdb"
)

func TestAuthIdentityOwnership(test *testing.T) {
	db := testdb.Open(test)
	_, err := db.Exec(`
		INSERT INTO users (email, id)
		VALUES ('first@test', 'auth-first'), ('second@test', 'auth-second')
	`)
	if err != nil {
		test.Fatal(err)
	}

	for _, tc := range []struct {
		identity string
		status   int
		owner    string
	}{
		{"", 401, ""}, {"unknown", 403, ""}, {"auth-first", 204, "auth-first"}, {"auth-second", 204, "auth-second"},
	} {
		test.Run(tc.identity, func(test *testing.T) {
			handler := ResolveAuthIdentity(http.HandlerFunc(func(writer http.ResponseWriter, httpRequest *http.Request) {
				id, ok := UserIDFromContext(httpRequest.Context())
				if !ok || id != tc.owner {
					test.Errorf("owner=%s want=%s", id, tc.owner)
				}
				writer.WriteHeader(204)
			}), db, func(*http.Request) string { return tc.identity })
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest("POST", "/api/sync/artists/pull?user_id=42", nil))
			if recorder.Code != tc.status {
				test.Fatalf("status=%d want=%d", recorder.Code, tc.status)
			}
		})
	}
}
