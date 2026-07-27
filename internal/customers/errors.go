package customers

import "errors"

var (
	ErrOrganizationRequired = errors.New("organization_id is required")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrOrganizationImmutable = errors.New("organization_id cannot be changed")
	ErrNotFound             = errors.New("customer not found")
)
