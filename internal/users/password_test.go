package users

import "testing"

func TestHashPassword(t *testing.T) {
	password := "secret123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	if hash == "" {
		t.Fatal("expected hash")
	}

	if hash == password {
		t.Fatal("password was not hashed")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "secret123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected valid password")
	}
}

func TestVerifyPassword_Invalid(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatal(err)
	}

	if ok {
		t.Fatal("expected invalid password")
	}
}
