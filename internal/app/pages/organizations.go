package pages

import "Orders/internal/organizations"

type OrganizationsPage struct {
	Title string
	Orgs  []*organizations.Organization
}
