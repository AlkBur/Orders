package users

import (
	"fmt"
	"strconv"
	"time"

	"Orders/internal/entity"
)

type User struct {
	ID           int64  `db:"id"`
	UUID         string `db:"uuid" label:"ID" order:"5"`
	Login        string `db:"login" label:"Логин" order:"10" list:"true" search:"true"`
	Email        string `db:"email" label:"Email" order:"15" list:"true"`
	PasswordHash string
	IsAdmin      bool `db:"is_admin" label:"Администратор" order:"20" list:"true"`
	HasPassword  bool `readonly:"true" label:"Пароль установлен" order:"30" list:"true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

var Descriptor = entity.Register[User](
	entity.PrimaryKey("ID"),
	entity.ExternalKey("UUID"),
)

func (u User) DisplayValue(name string) (string, error) {
	switch name {
	case "UUID":
		return u.UUID, nil
	case "Login":
		return u.Login, nil
	case "Email":
		return u.Email, nil
	case "IsAdmin":
		if u.IsAdmin {
			return "Да", nil
		}
		return "Нет", nil
	case "HasPassword":
		if u.HasPassword {
			return "Установлен", nil
		}
		return "Не установлен", nil
	default:
		return "", fmt.Errorf("unknown display field %q for User", name)
	}
}

func (u User) URL() string {
	return "/users/" + strconv.FormatInt(u.ID, 10)
}

func (u *User) HasPasswordSet() bool {
	return u.PasswordHash != ""
}

func (u *User) NeedsPasswordSetup() bool {
	return !u.HasPasswordSet()
}

func (u *User) SetPassword(password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

func (u *User) VerifyPassword(password string) (bool, error) {
	return VerifyPassword(password, u.PasswordHash)
}
