package users

type User struct {
	ID           int64
	Login        string
	PasswordHash string

	Name     string
	Disabled bool
}
