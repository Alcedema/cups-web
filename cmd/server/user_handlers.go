package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"unicode/utf8"

	"cups-web/internal/auth"
	"cups-web/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type meResponse struct {
	ID                int64  `json:"id"`
	Username          string `json:"username"`
	Role              string `json:"role"`
	Language          string `json:"language"`
	EffectiveLanguage string `json:"effectiveLanguage"`
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.GetSession(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var resp meResponse
	err = appStore.WithTx(r.Context(), true, func(tx *sql.Tx) error {
		user, err := store.GetUserByID(r.Context(), tx, sess.UserID)
		if err != nil {
			return err
		}
		resp = meResponse{
			ID:                user.ID,
			Username:          user.Username,
			Role:              user.Role,
			Language:          user.Language,
			EffectiveLanguage: effectiveLanguage(user.Language),
		}
		if user.Username == guestUsername {
			resp.Language = ""
			resp.EffectiveLanguage = defaultLanguage
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "failed to load profile")
		}
		return
	}
	writeJSON(w, resp)
}

func accountError(w http.ResponseWriter, status int, code, message string) {
	writeJSONStatus(w, status, map[string]string{"code": code, "error": message})
}

// Account writes deliberately require a browser session, even though other
// protected routes accept API keys. The route middleware also validates CSRF.
func accountUser(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	if auth.IsAPIKeyAuth(r.Context()) {
		accountError(w, http.StatusForbidden, "account.browser_required", "browser session required")
		return store.User{}, false
	}
	sess, err := auth.GetSession(r)
	if err != nil {
		accountError(w, http.StatusUnauthorized, "account.unauthorized", "unauthorized")
		return store.User{}, false
	}
	var user store.User
	err = appStore.WithTx(r.Context(), true, func(tx *sql.Tx) error {
		user, err = store.GetUserByID(r.Context(), tx, sess.UserID)
		return err
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			accountError(w, http.StatusUnauthorized, "account.unauthorized", "unauthorized")
		} else {
			accountError(w, http.StatusInternalServerError, "account.load_failed", "failed to load profile")
		}
		return store.User{}, false
	}
	if user.Username == guestUsername {
		accountError(w, http.StatusForbidden, "account.guest_read_only", "guest account cannot be changed")
		return store.User{}, false
	}
	return user, true
}

func decodeAccountBody(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		accountError(w, http.StatusBadRequest, "account.invalid_request", "invalid request")
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		accountError(w, http.StatusBadRequest, "account.invalid_request", "invalid request")
		return false
	}
	return true
}

func updatePreferencesHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := accountUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Language *string `json:"language"`
	}
	if !decodeAccountBody(w, r, &body) {
		return
	}
	if body.Language == nil || (*body.Language != "" && !validLanguage(*body.Language)) {
		accountError(w, http.StatusBadRequest, "account.invalid_language", "unsupported language")
		return
	}
	err := appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(r.Context(), "UPDATE users SET language = ?, updated_at = ? WHERE id = ?", *body.Language, nowRFC3339(), user.ID)
		return err
	})
	if err != nil {
		accountError(w, http.StatusInternalServerError, "account.save_failed", "failed to save preferences")
		return
	}
	writeJSON(w, map[string]string{"language": *body.Language, "effectiveLanguage": effectiveLanguage(*body.Language)})
}

func changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := accountUser(w, r)
	if !ok {
		return
	}
	var body struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !decodeAccountBody(w, r, &body) {
		return
	}
	if !utf8.ValidString(body.NewPassword) || utf8.RuneCountInString(body.NewPassword) < 8 || len(body.NewPassword) > 72 {
		accountError(w, http.StatusBadRequest, "account.password_length", "password must have at least 8 characters and at most 72 UTF-8 bytes")
		return
	}
	key := "password|" + strconv.FormatInt(user.ID, 10)
	if ok, _ := loginAllowed(key); !ok {
		accountError(w, http.StatusTooManyRequests, "account.rate_limited", "too many attempts, please try again later")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.CurrentPassword)) != nil {
		registerLoginFailure(key)
		accountError(w, http.StatusBadRequest, "account.current_password_wrong", "current password is incorrect")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		accountError(w, http.StatusInternalServerError, "account.password_failed", "failed to change password")
		return
	}
	// Compare-and-update prevents a concurrent password change from being lost.
	err = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(r.Context(), "UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ? AND password_hash = ?", string(hash), nowRFC3339(), user.ID, user.PasswordHash)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err == nil && n != 1 {
			return sql.ErrNoRows
		}
		return err
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			accountError(w, http.StatusConflict, "account.password_failed", "password changed concurrently; sign in again")
		} else {
			accountError(w, http.StatusInternalServerError, "account.password_failed", "failed to change password")
		}
		return
	}
	clearLoginFailures(key)
	auth.ClearSession(w, r)
	writeJSON(w, map[string]bool{"ok": true})
}
