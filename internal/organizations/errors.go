package organizations

import "errors"

var (
	ErrNotFound  = errors.New("organization not found")
	ErrEmptyUUID = errors.New("uuid is required")
)
