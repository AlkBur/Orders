package users

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid login or password")
	ErrNotFound           = errors.New("user not found")
	ErrEmptyUUID          = errors.New("uuid is required")
	ErrLastAdministrator  = errors.New("cannot remove the last administrator")
)
