package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"cups-web/internal/auth"
	"cups-web/internal/middleware"
	"cups-web/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func accountFixture(t *testing.T) (store.User, string) {
	t.Helper()
	old := appStore
	oldLanguage := defaultLanguage
	filename := filepath.Join(t.TempDir(), "test.db")
	var err error
	appStore, err = store.Open(context.Background(), filename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { appStore.Close(); appStore = old; defaultLanguage = oldLanguage })
	if err := auth.SetupSecureCookie(appStore.DB); err != nil {
		t.Fatal(err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("original-password"), bcrypt.MinCost)
	var user store.User
	err = appStore.WithTx(context.Background(), false, func(tx *sql.Tx) error {
		var e error
		user, e = store.CreateUser(context.Background(), tx, store.CreateUserInput{Username: "alice", PasswordHash: string(hash), Role: store.RoleAdmin})
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	return user, filename
}

func accountRequest(user store.User, handler http.HandlerFunc, payload string, csrf, apiKey bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPut, "/api/me", strings.NewReader(payload))
	r = r.WithContext(auth.WithSession(r.Context(), auth.Session{UserID: user.ID, Username: user.Username, Role: user.Role}))
	if apiKey {
		r = r.WithContext(auth.WithAPIKeyAuth(r.Context()))
	}
	if csrf {
		r.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-token"})
		r.Header.Set("X-CSRF-Token", "test-token")
	}
	w := httptest.NewRecorder()
	middleware.RequireSession(middleware.ValidateCSRF(handler)).ServeHTTP(w, r)
	return w
}

func TestAccountPreferencesAndPersistence(t *testing.T) {
	user, filename := accountFixture(t)
	defaultLanguage = "zh-CN"
	for _, tt := range []struct {
		body      string
		csrf, key bool
		status    int
	}{
		{`{"language":"en"}`, false, false, 403},
		{`{"language":"en"}`, true, true, 403},
		{`{}`, true, false, 400},
		{`{"language":null}`, true, false, 400},
		{`{"language":"de"}`, true, false, 400},
		{`{"language":"en","id":999}`, true, false, 400},
		{`{"language":"en"} {}`, true, false, 400},
		{`{"language":"en"}`, true, false, 200},
	} {
		if w := accountRequest(user, updatePreferencesHandler, tt.body, tt.csrf, tt.key); w.Code != tt.status {
			t.Fatalf("%s: %d %s", tt.body, w.Code, w.Body.String())
		}
	}
	appStore.Close()
	var err error
	appStore, err = store.Open(context.Background(), filename)
	if err != nil {
		t.Fatal(err)
	}
	var language, role, hash string
	err = appStore.DB.QueryRow("SELECT language,role,password_hash FROM users WHERE id=?", user.ID).Scan(&language, &role, &hash)
	if err != nil || language != "en" || role != user.Role || hash != user.PasswordHash {
		t.Fatalf("lost account fields: %s %s %v", language, role, err)
	}
	r := httptest.NewRequest("GET", "/api/me", nil)
	r = r.WithContext(auth.WithSession(r.Context(), auth.Session{UserID: user.ID}))
	w := httptest.NewRecorder()
	MeHandler(w, r)
	var me meResponse
	json.Unmarshal(w.Body.Bytes(), &me)
	if me.Language != "en" || me.EffectiveLanguage != "en" {
		t.Fatalf("%+v", me)
	}
	if w := accountRequest(user, updatePreferencesHandler, `{"language":""}`, true, false); w.Code != 200 || !strings.Contains(w.Body.String(), `"effectiveLanguage":"zh-CN"`) {
		t.Fatal(w.Body.String())
	}
}

func TestAccountPasswordValidation(t *testing.T) {
	user, _ := accountFixture(t)
	for _, tt := range []struct {
		current, next string
		csrf, key     bool
		status        int
	}{
		{"original-password", "new-password", false, false, 403},
		{"original-password", "new-password", true, true, 403},
		{"wrong-password", "new-password", true, false, 400},
		{"original-password", "short", true, false, 400},
		{"original-password", strings.Repeat("a", 73), true, false, 400},
		{"original-password", strings.Repeat("字", 25), true, false, 400},
		{"original-password", strings.Repeat("字", 24), true, false, 200},
	} {
		payload, _ := json.Marshal(map[string]string{"currentPassword": tt.current, "newPassword": tt.next})
		w := accountRequest(user, changePasswordHandler, string(payload), tt.csrf, tt.key)
		if w.Code != tt.status {
			t.Fatalf("status %d, want %d: %s", w.Code, tt.status, w.Body.String())
		}
		if tt.status == 200 {
			var hash, role, language string
			appStore.DB.QueryRow("SELECT password_hash,role,language FROM users WHERE id=?", user.ID).Scan(&hash, &role, &language)
			if bcrypt.CompareHashAndPassword([]byte(hash), []byte(tt.next)) != nil || role != user.Role || language != user.Language {
				t.Fatal("password update corrupted account")
			}
			cleared := false
			for _, c := range w.Result().Cookies() {
				if c.Name == "session" && c.MaxAge < 0 {
					cleared = true
				}
			}
			if !cleared {
				t.Fatal("session was not cleared")
			}
		}
	}
}

func TestGuestAndUnauthenticatedAccountWrites(t *testing.T) {
	user, _ := accountFixture(t)
	appStore.DB.Exec("UPDATE users SET username='guest' WHERE id=?", user.ID)
	for _, h := range []http.HandlerFunc{updatePreferencesHandler, changePasswordHandler} {
		w := accountRequest(user, h, `{}`, true, false)
		if w.Code != 403 {
			t.Fatalf("guest: %d", w.Code)
		}
		r := httptest.NewRequest("PUT", "/api/me", bytes.NewBufferString(`{}`))
		w = httptest.NewRecorder()
		middleware.RequireSession(middleware.ValidateCSRF(h)).ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("unauthenticated: %d", w.Code)
		}
	}
}

func TestLanguageDefaults(t *testing.T) {
	old := defaultLanguage
	t.Cleanup(func() { defaultLanguage = old })
	for _, tt := range []struct{ value, want string }{{"", "en"}, {"en", "en"}, {"zh-CN", "zh-CN"}, {"invalid", "en"}} {
		t.Setenv("DEFAULT_LANGUAGE", tt.value)
		configureLanguage()
		if defaultLanguage != tt.want || effectiveLanguage("") != tt.want || effectiveLanguage("zh-CN") != "zh-CN" {
			t.Fatalf("default %s", tt.value)
		}
	}
}
