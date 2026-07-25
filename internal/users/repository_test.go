package users

import (
	"database/sql"
	"errors"
	"testing"

	"Orders/internal/testutil"
)

func TestStore_CreateAndFindByLogin(t *testing.T) {
	db := testutil.NewTestDB(t)

	store := NewStore(db)

	user := &User{
		Login:        "admin",
		PasswordHash: "hash",
		Name:         "Administrator",
	}

	if err := store.Create(user); err != nil {
		t.Fatal(err)
	}

	count, err := store.Count()
	if err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf("expected 1 user, got %d", count)
	}

	found, err := store.FindByLogin("admin")
	if err != nil {
		t.Fatal(err)
	}

	if found.Login != user.Login {
		t.Fatal("login mismatch")
	}

	if found.Name != user.Name {
		t.Fatal("name mismatch")
	}
}

func TestStore_FindByLogin_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	store := NewStore(db)

	_, err := store.FindByLogin("unknown")

	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
