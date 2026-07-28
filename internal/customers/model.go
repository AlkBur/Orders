package customers

import "time"

type Customer struct {
	OrganizationID   string
	ID               string
	Name             string
	Active           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	OrganizationName string // заполняется через JOIN при чтении
}
