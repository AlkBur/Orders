package pages

import "Orders/internal/users"

type UserCardPage struct {
	Title string
	User  *users.User
}
