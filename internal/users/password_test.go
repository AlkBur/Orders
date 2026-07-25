package users

import "testing"

func TestHashPassword(t *testing.T) {
	password := "admin123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	if hash == "" {
		t.Fatal("hash is empty")
	}

	if hash == password {
		t.Fatal("hash equals password")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "admin123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	if err := VerifyPassword(hash, password); err != nil {
		t.Fatalf("expected valid password: %v", err)
	}
}

func TestVerifyPassword_Invalid(t *testing.T) {
	hash, err := HashPassword("admin123")
	if err != nil {
		t.Fatal(err)
	}

	if err := VerifyPassword(hash, "wrong-password"); err == nil {
		t.Fatal("expected password verification to fail")
	}
}
