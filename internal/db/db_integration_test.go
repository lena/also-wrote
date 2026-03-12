//go:build integration

package db

import (
	"fmt"
	"os"
	"testing"
	"time"
)

var testDB *DB

func TestMain(m *testing.M) {
	conn := os.Getenv("TEST_DATABASE_URL")
	if conn == "" {
		conn = "postgres://localhost/also_wrote_test?sslmode=disable"
	}
	var err error
	testDB, err = Open(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration tests: open test DB: %v\n", err)
		fmt.Fprintf(os.Stderr, "create a test database with: createdb also_wrote_test\n")
		fmt.Fprintf(os.Stderr, "then run: go test -tags=integration ./internal/db/...\n")
		os.Exit(1)
	}
	defer testDB.Close()
	os.Exit(m.Run())
}

func TestIntegration_UserAndFavorites(t *testing.T) {
	email := fmt.Sprintf("test-%d@test.local", time.Now().UnixNano())
	db := testDB

	// CreateUser
	u, err := db.CreateUser(email)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.Email != email || u.ID == 0 {
		t.Errorf("CreateUser: got id=%v email=%q", u.ID, u.Email)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM users WHERE email = $1", email)
	})

	// UserByID
	got, err := db.UserByID(u.ID)
	if err != nil {
		t.Fatalf("UserByID: %v", err)
	}
	if got == nil || got.Email != email {
		t.Errorf("UserByID: got %+v", got)
	}

	// UserByEmail
	got, err = db.UserByEmail(email)
	if err != nil || got == nil || got.ID != u.ID {
		t.Errorf("UserByEmail: err=%v got=%+v", err, got)
	}

	// GetOrCreateUser (existing)
	got, err = db.GetOrCreateUser(email)
	if err != nil || got == nil || got.ID != u.ID {
		t.Errorf("GetOrCreateUser existing: err=%v got=%+v", err, got)
	}

	// AddFavoriteWriter
	if err := db.AddFavoriteWriter(u.ID, 12345); err != nil {
		t.Fatalf("AddFavoriteWriter: %v", err)
	}
	if err := db.AddFavoriteWriter(u.ID, 12345); err != nil { // idempotent
		t.Fatalf("AddFavoriteWriter duplicate: %v", err)
	}
	if err := db.AddFavoriteWriter(u.ID, 67890); err != nil {
		t.Fatalf("AddFavoriteWriter second: %v", err)
	}

	// IsFavoriteWriter
	ok, err := db.IsFavoriteWriter(u.ID, 12345)
	if err != nil || !ok {
		t.Errorf("IsFavoriteWriter(12345): ok=%v err=%v", ok, err)
	}
	ok, err = db.IsFavoriteWriter(u.ID, 99999)
	if err != nil || ok {
		t.Errorf("IsFavoriteWriter(99999): ok=%v err=%v", ok, err)
	}

	// FavoriteWriterPersonIDs
	ids, err := db.FavoriteWriterPersonIDs(u.ID)
	if err != nil {
		t.Fatalf("FavoriteWriterPersonIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("FavoriteWriterPersonIDs: got %v", ids)
	}

	// RemoveFavoriteWriter
	if err := db.RemoveFavoriteWriter(u.ID, 12345); err != nil {
		t.Fatalf("RemoveFavoriteWriter: %v", err)
	}
	ids, _ = db.FavoriteWriterPersonIDs(u.ID)
	if len(ids) != 1 || ids[0] != 67890 {
		t.Errorf("after remove: FavoriteWriterPersonIDs = %v", ids)
	}
}

func TestIntegration_GetOrCreateUser_creates(t *testing.T) {
	email := fmt.Sprintf("test-%d@test.local", time.Now().UnixNano())
	db := testDB

	u, err := db.GetOrCreateUser(email)
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}
	if u == nil || u.Email != email {
		t.Errorf("GetOrCreateUser: got %+v", u)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM users WHERE email = $1", email)
	})

	// second call returns same user
	u2, err := db.GetOrCreateUser(email)
	if err != nil || u2 == nil || u2.ID != u.ID {
		t.Errorf("GetOrCreateUser second: err=%v got=%+v", err, u2)
	}
}

func TestIntegration_UserByID_notFound(t *testing.T) {
	got, err := testDB.UserByID(999999999)
	if err != nil {
		t.Fatalf("UserByID: %v", err)
	}
	if got != nil {
		t.Errorf("UserByID(999999999): got %+v, want nil", got)
	}
}
