package customers

import (
	"fmt"
	"strconv"
	"time"

	"Orders/internal/entity"
)

type Customer struct {
	ID               int64  `db:"id" order:"2"`
	UUID             string `db:"uuid" label:"ID" order:"5"`
	OrganizationID   int64  `db:"organization_id" order:"3"`
	OrganizationName string `readonly:"true" label:"Организация" order:"15" list:"true"`
	Name             string `db:"name" label:"Наименование" order:"20" list:"true" search:"true"`
	Active           bool   `db:"active" label:"Активен" order:"30" list:"true"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

var Descriptor = entity.Register[Customer](
	entity.PrimaryKey("ID"),
	entity.ExternalKey("OrganizationID", "UUID"),
)

func (c Customer) DisplayValue(name string) (string, error) {
	switch name {
	case "UUID":
		return c.UUID, nil
	case "OrganizationName":
		return c.OrganizationName, nil
	case "Name":
		return c.Name, nil
	case "Active":
		if c.Active {
			return "Активен", nil
		}
		return "Неактивен", nil
	default:
		return "", fmt.Errorf("unknown display field %q for Customer", name)
	}
}

func (c Customer) URL() string {
	return "/organizations/" + strconv.FormatInt(c.OrganizationID, 10) + "/customers/" + strconv.FormatInt(c.ID, 10)
}
