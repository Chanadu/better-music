package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/internal/testdb"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword"
	"github.com/supertokens/supertokens-golang/supertokens"
)

func TestSuperTokensIntegration(test *testing.T) {
	uri := os.Getenv("TEST_SUPERTOKENS_URI")
	if uri == "" {
		test.Skip("set TEST_SUPERTOKENS_URI and TEST_SUPERTOKENS_API_KEY for real Core checks")
	}
	database := testdb.Open(test)
	coreURL, err := url.Parse(uri)
	if err != nil {
		test.Fatal(err)
	}
	var blockRevocation atomic.Bool
	proxy := httputil.NewSingleHostReverseProxy(coreURL)
	core := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, httpRequest *http.Request) {
		if blockRevocation.Load() && strings.HasSuffix(httpRequest.URL.Path, "/recipe/session/remove") {
			http.Error(writer, "revocation temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		proxy.ServeHTTP(writer, httpRequest)
	}))
	test.Cleanup(core.Close)
	cfg := config.Config{Auth: config.AuthConfig{CoreURI: core.URL, APIKey: os.Getenv("TEST_SUPERTOKENS_API_KEY"), APIDomain: "http://localhost:5173", WebsiteDomain: "http://localhost:5173"}}
	router, service, err := newAPIHandler(&cfg, database)
	if err != nil {
		test.Fatal(err)
	}
	cookies := map[int][]*http.Cookie{}
	request := func(index int, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://localhost:5173"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("rid", "emailpassword")
		req.Header.Set("st-auth-mode", "cookie")
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		for _, cookie := range cookies[index] {
			if cookie.Value != "" {
				req.AddCookie(cookie)
			}
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}
	sdkOK := func(recorder *httptest.ResponseRecorder) {
		test.Helper()
		var body struct {
			Status string `json:"status"`
		}
		_ = json.Unmarshal(recorder.Body.Bytes(), &body)
		if recorder.Code != 200 || body.Status != "OK" {
			test.Fatalf("SDK request: %d %s", recorder.Code, recorder.Body.String())
		}
	}
	suffix := time.Now().UnixNano()
	emails := []string{}
	owners := []string{}
	password := "Local-test-password-123!"
	signIn := func(index int, email, pass string) {
		test.Helper()
		body := fmt.Sprintf(`{"formFields":[{"id":"email","value":%q},{"id":"password","value":%q}]}`, email, pass)
		recorder := request(-1, "POST", "/api/auth/signin", body, nil)
		sdkOK(recorder)
		cookies[index] = recorder.Result().Cookies()
	}
	for index := 0; index < 3; index++ {
		email := fmt.Sprintf("st-test-%d-%d@example.test", suffix, index)
		emails = append(emails, email)
		recorder := request(-1, "POST", "/api/auth/signup", fmt.Sprintf(`{"formFields":[{"id":"email","value":%q},{"id":"password","value":%q}]}`, email, password), nil)
		sdkOK(recorder)
		var id string
		if err = database.QueryRow(`SELECT id FROM users WHERE email=$1`, email).Scan(&id); err != nil {
			test.Fatal(err)
		}
		owners = append(owners, id)
		test.Cleanup(func() {
			if err := supertokens.DeleteUser(id); err != nil {
				test.Error(err)
			}
		})
		signIn(index, email, password)
		_, err = database.Exec(`INSERT INTO library_artists(id,user_id,name,created_at,edited_at,change_seq) VALUES($1,$2,$3,'2020-01-01T00:00:00.000Z',NOW(),1);`, fmt.Sprintf("00000000-0000-4000-8000-%012d", index+1), id, fmt.Sprintf("Artist %s", id))
		if err != nil {
			test.Fatal(err)
		}
		if _, err = database.Exec(`UPDATE users SET library_sync_sequence=1 WHERE id=$1`, id); err != nil {
			test.Fatal(err)
		}
	}
	// Even a valid Core session cannot access an unmapped library.
	if _, err = database.Exec(`UPDATE users SET id='unmapped' WHERE id=$1`, owners[2]); err != nil {
		test.Fatal(err)
	}
	for _, tc := range []struct{ index, status int }{{-1, 401}, {2, 403}, {0, 200}, {1, 200}} {
		recorder := request(tc.index, "GET", "/api/account", "", nil)
		if recorder.Code != tc.status {
			test.Fatalf("account %d: %d %s", tc.index, recorder.Code, recorder.Body.String())
		}
		if tc.status == http.StatusOK {
			var account struct {
				Email string `json:"email"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &account); err != nil || account.Email != emails[tc.index] {
				test.Fatalf("account %d returned %s", tc.index, recorder.Body.String())
			}
		}
	}
	if recorder := request(0, "GET", "/api/account", "", map[string]string{"X-Better-Music-Account": owners[1]}); recorder.Code != http.StatusConflict {
		test.Fatalf("account owner mismatch: %d %s", recorder.Code, recorder.Body.String())
	}

	for _, tc := range []struct{ index, status int }{{-1, 401}, {2, 403}, {0, 200}, {1, 200}} {
		recorder := request(tc.index, "POST", "/api/sync/artists/pull", `{"checkpoint":null,"limit":100}`, nil)
		if recorder.Code != tc.status {
			test.Fatalf("pull %d: %d %s", tc.index, recorder.Code, recorder.Body.String())
		}
		if tc.status == 200 {
			var body struct {
				Documents []struct {
					Name string `json:"name"`
				} `json:"documents"`
			}
			if err = json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				test.Fatal(err)
			}
			if len(body.Documents) != 1 || body.Documents[0].Name != fmt.Sprintf("Artist %s", owners[tc.index]) {
				test.Fatal("wrong owner's library")
			}
		}
	}
	if recorder := request(0, "POST", "/api/sync/artists/pull", `{"checkpoint":null,"limit":100}`, map[string]string{"rid": ""}); recorder.Code == 200 {
		test.Fatal("CSRF bypass")
	}
	if recorder := request(0, "POST", "/api/sync/artists/pull", `{"checkpoint":null,"limit":100}`, map[string]string{"X-Better-Music-Account": fmt.Sprint(owners[1])}); recorder.Code != 409 {
		test.Fatal("accepted changed cookie owner")
	}
	// The expected owner check runs before a replacement cookie is issued.
	recorder := request(-1, "POST", "/api/auth/signin", fmt.Sprintf(`{"formFields":[{"id":"email","value":%q},{"id":"password","value":%q}]}`, emails[1], password), map[string]string{"X-Better-Music-Account": fmt.Sprint(owners[0])})
	if recorder.Code != http.StatusOK ||
		!strings.Contains(recorder.Body.String(), `"status":"WRONG_CREDENTIALS_ERROR"`) ||
		len(recorder.Result().Cookies()) > 0 {
		test.Fatalf("account-switch rejection: %d %s", recorder.Code, recorder.Body.String())
	}
	if recorder := request(0, "POST", "/api/auth/session/refresh", "", nil); recorder.Code != 200 {
		test.Fatalf("refresh: %d", recorder.Code)
	}
	for _, headers := range []map[string]string{nil, {"X-Better-Music-Account": owners[1]}} {
		recorder := request(0, "POST", "/api/auth/signout", "", headers)
		if !strings.Contains(recorder.Body.String(), `"status":"GENERAL_ERROR"`) {
			test.Fatalf("signout accepted a missing or different owner: %d %s", recorder.Code, recorder.Body.String())
		}
	}
	if recorder := request(0, "POST", "/api/auth/session/refresh", "", nil); recorder.Code != 200 {
		test.Fatalf("rejected signout revoked the session: %d", recorder.Code)
	}
	sdkOK(request(0, "POST", "/api/auth/signout", "", map[string]string{"X-Better-Music-Account": owners[0]}))
	if recorder := request(0, "GET", "/api/session", "", nil); recorder.Code != http.StatusUnauthorized {
		test.Fatalf("session lookup accepted a revoked session: %d", recorder.Code)
	}
	if recorder := request(0, "POST", "/api/auth/session/refresh", "", nil); recorder.Code == 200 {
		test.Fatal("revoked session refreshed")
	}
	signIn(0, emails[0], password)

	// Legacy authentication routes and bearer credentials are no longer accepted.
	for _, path := range []string{"/api/auth/register", "/api/auth/login", "/api/auth/refresh", "/api/auth/logout"} {
		if recorder := request(0, "POST", path, `{}`, nil); recorder.Code != 404 {
			test.Fatalf("retired auth route %s returned %d", path, recorder.Code)
		}
	}
	if recorder := request(-1, "GET", "/api/session", "", map[string]string{"Authorization": "Bearer eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoxfQ.old-signature"}); recorder.Code != 401 {
		test.Fatal("bearer authentication still accepted")
	}

	// Administrator recovery preserves the existing SuperTokens account.
	recoveryEmail := fmt.Sprintf("recovery-%d@example.test", suffix)
	sdkOK(request(-1, "POST", "/api/auth/signup", fmt.Sprintf(
		`{"formFields":[{"id":"email","value":%q},{"id":"password","value":%q}]}`, recoveryEmail, password), nil))
	var recoveryID string
	if err = database.QueryRow(`SELECT id FROM users WHERE email = $1`, recoveryEmail).Scan(&recoveryID); err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() { _ = supertokens.DeleteUser(recoveryID) })

	for _, email := range []string{recoveryEmail, "unknown@example.test"} {
		recorder := request(-1, "POST", "/api/auth/user/password/reset/token", fmt.Sprintf(`{"formFields":[{"id":"email","value":%q}]}`, email), nil)
		if recorder.Code == 200 {
			test.Fatal("public email-based password reset remains enabled")
		}
	}
	for _, id := range []string{owners[2], "missing"} {
		if _, err = service.PasswordResetLink(context.Background(), id); err == nil {
			test.Fatal("reset link generated for an unmapped or missing account")
		}
	}
	privateLink, err := service.PasswordResetLink(context.Background(), recoveryID)
	if err != nil {
		test.Fatal(err)
	}
	link, err := url.Parse(privateLink)
	if err != nil {
		test.Fatal(err)
	}
	if link.Path != "/auth/reset-password" {
		test.Fatalf("reset route: %s", link.Path)
	}
	reset := fmt.Sprintf(`{"formFields":[{"id":"password","value":"Reset-password-123!"}],"token":%q}`, link.Query().Get("token"))
	sdkOK(request(-1, "POST", "/api/auth/user/password/reset", reset, nil))
	if recorder := request(-1, "POST", "/api/auth/user/password/reset", reset, nil); strings.Contains(recorder.Body.String(), `"status":"OK"`) {
		test.Fatal("reset token reused")
	}
	signIn(3, recoveryEmail, "Reset-password-123!")
	recorder = request(3, "GET", "/api/session", "", nil)
	var sessionInfo struct {
		UserID    string `json:"user_id"`
		Email     string `json:"email"`
		SyncEpoch string `json:"sync_epoch"`
	}
	if recorder.Code != 200 || json.Unmarshal(recorder.Body.Bytes(), &sessionInfo) != nil ||
		sessionInfo.UserID != recoveryID || sessionInfo.Email != recoveryEmail || sessionInfo.SyncEpoch == "" {
		test.Fatal("recovered session did not return the expected account details")
	}

	signIn(4, recoveryEmail, "Reset-password-123!")
	sdkOK(request(3, "POST", "/api/auth/signout", "", map[string]string{"X-Better-Music-Account": recoveryID}))
	if recorder := request(4, "POST", "/api/auth/session/refresh", "", nil); recorder.Code != 200 {
		test.Fatalf("signout interrupted an independent session: %d", recorder.Code)
	}

	signIn(3, recoveryEmail, "Reset-password-123!")
	privateLink, err = service.PasswordResetLink(context.Background(), recoveryID)
	if err != nil {
		test.Fatal(err)
	}
	link, err = url.Parse(privateLink)
	if err != nil {
		test.Fatal(err)
	}
	reset = fmt.Sprintf(`{"formFields":[{"id":"password","value":"Recovered-password-123!"}],"token":%q}`, link.Query().Get("token"))
	blockRevocation.Store(true)
	sdkOK(request(-1, "POST", "/api/auth/user/password/reset", reset, nil))
	var revocationPending bool
	if err := database.QueryRow(`SELECT session_revocation_pending FROM users WHERE id = $1`, recoveryID).Scan(&revocationPending); err != nil || !revocationPending {
		test.Fatalf("failed revocation was not queued: %v", err)
	}
	blockRevocation.Store(false)
	if err := service.ReconcilePending(context.Background()); err != nil {
		test.Fatal(err)
	}
	if err := database.QueryRow(`SELECT session_revocation_pending FROM users WHERE id = $1`, recoveryID).Scan(&revocationPending); err != nil || revocationPending {
		test.Fatalf("completed revocation remained queued: %v", err)
	}
	for _, index := range []int{3, 4} {
		if recorder := request(index, "POST", "/api/auth/session/refresh", "", nil); recorder.Code != http.StatusUnauthorized {
			test.Fatalf("session %d refreshed after password recovery: %d", index, recorder.Code)
		}
	}

	// Lifecycle actions use Core credentials, not the unused legacy hash.
	// Invalid JSON must not echo credentials in Huma's problem details.
	malformed := request(0, http.MethodPut, "/api/account/email", `{"password":"secret-do-not-echo",`, nil)
	if malformed.Code != http.StatusBadRequest || strings.Contains(malformed.Body.String(), "secret-do-not-echo") {
		test.Fatalf("malformed account request returned HTTP %d or echoed the password", malformed.Code)
	}
	if recorder := request(0, http.MethodPut, "/api/account/email", "{}", map[string]string{"X-Better-Music-Account": owners[1]}); recorder.Code != http.StatusConflict {
		test.Fatalf("account change owner mismatch: %d %s", recorder.Code, recorder.Body.String())
	}
	if recorder := request(0, "PUT", "/api/account/email", `{"email":"new@example.test","password":"wrong"}`, nil); recorder.Code != 400 {
		test.Fatal("wrong password accepted")
	}
	coreOnlyEmail := fmt.Sprintf("core-only-%d@example.test", suffix)
	moved, err := emailpassword.UpdateEmailOrPassword(owners[2], &coreOnlyEmail, nil, nil, nil)
	if err != nil || moved.OK == nil {
		test.Fatalf("stage a Core-only email collision: %v", err)
	}
	if recorder := request(0, "PUT", "/api/account/email", fmt.Sprintf(`{"email":%q,"password":%q}`, coreOnlyEmail, password), nil); recorder.Code != http.StatusConflict {
		test.Fatalf("Core-only email collision: %d %s", recorder.Code, recorder.Body.String())
	}
	var pendingEmailChange bool
	if err = database.QueryRow(`SELECT EXISTS(SELECT 1 FROM auth_account_operations WHERE user_id=$1)`, owners[0]).Scan(&pendingEmailChange); err != nil || pendingEmailChange {
		test.Fatalf("pending email change after collision: %t, query error: %v", pendingEmailChange, err)
	}
	newEmail := fmt.Sprintf("changed-%d@example.test", suffix)
	if recorder := request(0, "PUT", "/api/account/email", fmt.Sprintf(`{"email":%q,"password":%q}`, newEmail, password), nil); recorder.Code != 200 {
		test.Fatalf("email change: %d %s", recorder.Code, recorder.Body.String())
	}
	signIn(0, newEmail, password)
	var storedEmail string
	if err = database.QueryRow(`SELECT email FROM users WHERE id=$1`, owners[0]).Scan(&storedEmail); err != nil || storedEmail != newEmail {
		test.Fatal("email copy stale")
	}
	if recorder := request(0, "PUT", "/api/account/password", fmt.Sprintf(`{"current_password":%q,"new_password":"Updated-password-123!"}`, password), nil); recorder.Code != 200 {
		test.Fatalf("password change: %d %s", recorder.Code, recorder.Body.String())
	}
	signIn(0, newEmail, "Updated-password-123!")
	if recorder := request(0, "POST", "/api/sync/artists/pull", `{"checkpoint":null,"limit":100}`, nil); recorder.Code != 200 {
		test.Fatal("library lost after password change")
	}
	if recorder := request(0, "DELETE", "/api/account", `{"password":"Updated-password-123!"}`, nil); recorder.Code != 200 {
		test.Fatalf("delete: %d %s", recorder.Code, recorder.Body.String())
	}
	var count int
	if err = database.QueryRow(`SELECT count(*) FROM users WHERE id=$1`, owners[0]).Scan(&count); err != nil || count != 0 {
		test.Fatal("application owner not deleted")
	}
	// Replay a persisted email operation after Core changed but SQL did not commit.
	repairEmail := fmt.Sprintf("repaired-%d@example.test", suffix)
	if _, err = database.Exec(`INSERT INTO auth_account_operations(user_id,kind,target_email) VALUES($1,'email',$2)`, owners[1], repairEmail); err != nil {
		test.Fatal(err)
	}
	if _, err = service.PasswordResetLink(context.Background(), owners[1]); err == nil {
		test.Fatal("generated a reset link during an unfinished email change")
	}
	updated, err := emailpassword.UpdateEmailOrPassword(owners[1], &repairEmail, nil, nil, nil)
	if err != nil || updated.OK == nil {
		test.Fatal("failed to stage interrupted Core email change")
	}
	if err = service.ReconcilePending(context.Background()); err != nil {
		test.Fatal(err)
	}
	if err = database.QueryRow(`SELECT email FROM users WHERE id=$1`, owners[1]).Scan(&storedEmail); err != nil || storedEmail != repairEmail {
		test.Fatal("interrupted email change not reconciled")
	}
	signIn(1, repairEmail, password)
	// Invalid password input must not revoke a valid session.
	if recorder := request(1, "PUT", "/api/account/password", fmt.Sprintf(`{"current_password":%q,"new_password":"no-digits-password"}`, password), nil); recorder.Code != 400 {
		test.Fatal("invalid password policy accepted")
	}
	if recorder := request(1, "POST", "/api/auth/session/refresh", "", nil); recorder.Code != 200 {
		test.Fatal("invalid password update revoked session")
	}
	// A persisted deletion blocks access immediately, then cleans Core and SQL.
	if _, err = database.Exec(`UPDATE users SET auth_disabled=true WHERE id=$1`, owners[1]); err != nil {
		test.Fatal(err)
	}
	if _, err = database.Exec(`INSERT INTO auth_account_operations(user_id,kind) VALUES($1,'delete')`, owners[1]); err != nil {
		test.Fatal(err)
	}
	if _, err = service.PasswordResetLink(context.Background(), owners[1]); err == nil {
		test.Fatal("generated a reset link for a disabled account")
	}
	if recorder := request(1, "GET", "/api/session", "", nil); recorder.Code != 403 {
		test.Fatal("pending deletion still has access")
	}

	if err = service.ReconcilePending(context.Background()); err != nil {
		test.Fatal(err)
	}
}
