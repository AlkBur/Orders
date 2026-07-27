package pages

import "Orders/internal/organizations"

type OrganizationCardPage struct {
	Title string
	Org   *organizations.Organization
}
