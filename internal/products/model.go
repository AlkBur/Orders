package products

import (
	"fmt"
	"time"

	"Orders/internal/entity"
)

type Product struct {
	OrganizationID   string
	OrganizationName string `readonly:"true" label:"Организация" order:"25" list:"true"`
	ID               string `db:"id" label:"ID" order:"10"`
	Name             string `db:"name" label:"Наименование" order:"20" list:"true" search:"true"`
	Unit             string `db:"unit" label:"Ед. изм" order:"30" list:"true"`
	Active           bool   `db:"active" label:"Активен" order:"40" list:"true"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

var Descriptor = entity.Register[Product]()

func (p Product) DisplayValue(name string) (string, error) {
	switch name {
	case "Name":
		return p.Name, nil
	case "OrganizationName":
		return p.OrganizationName, nil
	case "Unit":
		return p.Unit, nil
	case "Active":
		return formatActive(p.Active), nil
	default:
		return "", fmt.Errorf("unknown display field %q for Product", name)
	}
}

func (p Product) URL() string {
	return "/organizations/" + p.OrganizationID + "/products/" + p.ID
}

func formatActive(v bool) string {
	if v {
		return "Активен"
	}
	return "Неактивен"
}
