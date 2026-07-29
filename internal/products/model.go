package products

import (
	"fmt"
	"strconv"
	"time"

	"Orders/internal/entity"
)

type Product struct {
	ID               int64  `db:"id" order:"2"`
	UUID             string `db:"uuid" label:"ID" order:"5"`
	OrganizationID   int64  `db:"organization_id" order:"3"`
	OrganizationName string `readonly:"true" label:"Организация" order:"15" list:"true"`
	Name             string `db:"name" label:"Наименование" order:"20" list:"true" search:"true"`
	Unit             string `db:"unit" label:"Ед. изм" order:"30" list:"true"`
	Active           bool   `db:"active" label:"Активен" order:"40" list:"true"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

var Descriptor = entity.Register[Product](
	entity.PrimaryKey("ID"),
	entity.ExternalKey("OrganizationID", "UUID"),
)

func (p Product) DisplayValue(name string) (string, error) {
	switch name {
	case "UUID":
		return p.UUID, nil
	case "OrganizationName":
		return p.OrganizationName, nil
	case "Name":
		return p.Name, nil
	case "Unit":
		return p.Unit, nil
	case "Active":
		return formatActive(p.Active), nil
	default:
		return "", fmt.Errorf("unknown display field %q for Product", name)
	}
}

func (p Product) URL() string {
	return "/organizations/" + strconv.FormatInt(p.OrganizationID, 10) + "/products/" + strconv.FormatInt(p.ID, 10)
}

func formatActive(v bool) string {
	if v {
		return "Активен"
	}
	return "Неактивен"
}
