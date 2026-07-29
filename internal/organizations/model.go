package organizations

import (
	"strconv"
	"time"

	"Orders/internal/entity"
)

type Organization struct {
	ID        int64  `db:"id"`
	UUID      string `db:"uuid" label:"ID" order:"10"`
	Name      string `db:"name" label:"Наименование" order:"20" list:"true"`
	APIKey    string
	Active    bool   `db:"active" label:"Статус" order:"30" list:"true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

var Descriptor = entity.Register[Organization](
	entity.PrimaryKey("ID"),
	entity.ExternalKey("UUID"),
)

func (o Organization) DisplayValue(name string) (string, error) {
	switch name {
	case "UUID":
		return o.UUID, nil
	case "Name":
		return o.Name, nil
	case "Active":
		if o.Active {
			return "Активен", nil
		}
		return "Неактивен", nil
	default:
		return "", nil
	}
}

func (o Organization) URL() string {
	return "/organizations/" + strconv.FormatInt(o.ID, 10)
}
