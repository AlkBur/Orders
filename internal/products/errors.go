package products

import "errors"

var (
	ErrOrganizationRequired = errors.New("organization_id is required")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrNotFound             = errors.New("product not found")
)
