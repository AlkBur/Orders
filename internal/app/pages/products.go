package pages

import (
	"Orders/internal/organizations"
	"Orders/internal/products"
)

type ProductCardPage struct {
	Title          string
	Product        *products.Product
	Orgs           []*organizations.Organization
	OrganizationID string
	IsNew          bool
}
