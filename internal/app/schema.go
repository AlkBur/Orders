package app

import (
	"Orders/internal/customers"
	"Orders/internal/database"
	"Orders/internal/organizations"
	"Orders/internal/sessions"
	"Orders/internal/users"
)

func NewSchema() *database.Schema {
	s := database.NewSchema()

	if err := s.Register(users.Table); err != nil {
		panic(err)
	}
	if err := s.Register(sessions.Table); err != nil {
		panic(err)
	}
	if err := s.Register(customers.Table); err != nil {
		panic(err)
	}
	if err := s.Register(organizations.Table); err != nil {
		panic(err)
	}

	return s
}
