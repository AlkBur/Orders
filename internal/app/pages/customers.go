package pages

import (
	"Orders/internal/customers"
	"Orders/internal/organizations"
)

type CustomerCardPage struct {
	Title          string
	Customer       *customers.Customer
	Orgs           []*organizations.Organization
	OrganizationID int64
}
