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
		IsAdmin:      true,
	}

	if err := store.Create(user); err != nil {
		t.Fatal(err)
	}

	if user.ID == 0 {
		t.Fatal("expected ID to be assigned")
	}

	found, err := store.FindByLogin("admin")
	if err != nil {
		t.Fatal(err)
	}

	if found.ID != user.ID {
		t.Fatal("id mismatch")
	}

	if found.Login != user.Login {
		t.Fatal("login mismatch")
	}

	if found.PasswordHash != user.PasswordHash {
		t.Fatal("password hash mismatch")
	}

	if found.IsAdmin != user.IsAdmin {
		t.Fatal("admin flag mismatch")
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

func TestStore_FindAdmin(t *testing.T) {
	db := testutil.NewTestDB(t)
	store := NewStore(db)

	admin := &User{
		Login:   "admin",
		IsAdmin: true,
	}

	if err := store.Create(admin); err != nil {
		t.Fatal(err)
	}

	found, err := store.FindAdmin()
	if err != nil {
		t.Fatal(err)
	}

	if found.ID != admin.ID {
		t.Fatal("id mismatch")
	}

	if found.Login != admin.Login {
		t.Fatal("login mismatch")
	}

	if !found.IsAdmin {
		t.Fatal("expected administrator")
	}
}

func TestStore_FindAdmin_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	store := NewStore(db)

	_, err := store.FindAdmin()

	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
