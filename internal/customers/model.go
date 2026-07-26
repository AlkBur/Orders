package customers

import "time"

type CustomerSnapshot struct {
	UUID string
	Name string
}

type Customer struct {
	UUID      string
	Name      string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
