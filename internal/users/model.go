package users

import "time"

type User struct {
	ID           int64
	Login        string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// HasPassword возвращает true, если пароль установлен.
func (u *User) HasPassword() bool {
	return u.PasswordHash != ""
}

// SetPassword устанавливает новый пароль.
func (u *User) SetPassword(password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	u.PasswordHash = hash
	return nil
}

// VerifyPassword проверяет пароль.
func (u *User) VerifyPassword(password string) (bool, error) {
	return VerifyPassword(password, u.PasswordHash)
}
