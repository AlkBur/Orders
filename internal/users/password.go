package users

import (
	"crypto/subtle"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword создает bcrypt-хэш пароля.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// VerifyBootstrapPassword проверяет введённый пароль
// на соответствие initial password из конфигурации.
func VerifyBootstrapPassword(password, initialPassword string) bool {
	return subtle.ConstantTimeCompare(
		[]byte(password),
		[]byte(initialPassword),
	) == 1
}

// VerifyPassword проверяет соответствие пароля bcrypt-хэшу.
func VerifyPassword(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(password),
	)

	if err == nil {
		return true, nil
	}

	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}

	return false, err
}
