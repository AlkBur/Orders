package pages

import (
	"Orders/internal/customers"
	"Orders/internal/organizations"
)

type CustomersPage struct {
	Title            string
	Customers        []*customers.Customer
	OrganizationID   string
	ShowOrganization bool
}

type CustomerCardPage struct {
	Title          string
	Customer       *customers.Customer
	Orgs           []*organizations.Organization
	OrganizationID string
	IsNew          bool
}
