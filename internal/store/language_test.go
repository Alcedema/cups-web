package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestLanguageMigrationPreservesExistingAccount(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatal(err)
	}
	// The pre-language schema, with real persisted account fields and session keys.
	_, err = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, role TEXT NOT NULL, protected INTEGER NOT NULL DEFAULT 0, contact_name TEXT, phone TEXT, email TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
 INSERT INTO users VALUES (1,'existing','preserved-bcrypt-hash','admin',1,'Contact','123','user@example.invalid','2026-01-01','2026-01-01');
 CREATE TABLE settings (key TEXT PRIMARY KEY,value TEXT NOT NULL);
 INSERT INTO settings VALUES ('session_hash_key','preserve-key');`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	for i := 0; i < 2; i++ {
		s, err := Open(context.Background(), filename)
		if err != nil {
			t.Fatal(err)
		}
		err = s.WithTx(context.Background(), true, func(tx *sql.Tx) error {
			user, e := GetUserByID(context.Background(), tx, 1)
			if e != nil {
				return e
			}
			if user.Language != "" || user.PasswordHash != "preserved-bcrypt-hash" || user.Role != RoleAdmin || !user.Protected || user.Email != "user@example.invalid" {
				t.Fatalf("migration changed account: %+v", user)
			}
			value, e := GetSettingString(context.Background(), tx, "session_hash_key", "")
			if value != "preserve-key" {
				t.Fatal("session key changed")
			}
			return e
		})
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
	}
}
