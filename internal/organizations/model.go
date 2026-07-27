package organizations

import "time"

type Organization struct {
	UUID      string
	Name      string
	APIKey    string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
