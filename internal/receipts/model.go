package receipts

import (
	"fmt"
	"strconv"
	"time"

	"Orders/internal/entity"
)

type Receipt struct {
	ID               int64      `db:"id" order:"1"`
	UUID             string     `db:"uuid" order:"2"`
	ExchangeID       string     `db:"exchange_id" order:"3"`
	Number           string     `db:"number" label:"Номер" order:"5" list:"true"`
	Date             time.Time  `db:"date" label:"Дата" order:"10" list:"true"`
	OrganizationID   int64      `db:"organization_id" order:"15"`
	OrganizationName string     `readonly:"true" label:"Организация" order:"20" list:"true"`
	UserID           int64      `db:"user_id" order:"22"`
	UserLogin        string     `readonly:"true" label:"Пользователь" order:"25"`
	CustomerID       int64      `db:"customer_id" order:"30"`
	CustomerName     string     `readonly:"true" label:"Клиент" order:"35" list:"true"`
	Total            float64    `db:"total" label:"Сумма" order:"40" list:"true"`
	SentAt           *time.Time `db:"sent_at" order:"42"`
	Status           string     `db:"status" label:"Статус" order:"45" list:"true"`
	StatusColor      string     `db:"status_color" order:"46"`
	CreatedAt        time.Time `order:"98"`
	UpdatedAt        time.Time `order:"99"`
}

type ReceiptItem struct {
	ID          int64   `db:"id" order:"1"`
	ReceiptID   int64   `db:"receipt_id" order:"2"`
	LineNum     int     `db:"line_num" label:"№" order:"5" list:"true"`
	ProductID   int64   `db:"product_id" order:"7"`
	ProductName string  `readonly:"true" label:"Товар" order:"10" list:"true"`
	Unit        string  `db:"unit" label:"Е. изм" order:"15" list:"true"`
	Quantity    float64 `db:"quantity" label:"Количество" order:"20" list:"true"`
	Price       float64 `db:"price" label:"Цена" order:"25" list:"true"`
	Amount      float64 `db:"amount" label:"Сумма" order:"30" list:"true"`
}

type Document struct {
	Receipt *Receipt
	Items   []ReceiptItem
}

type ReceiptUpdate struct {
	ExchangeID  string
	UUID        *string
	Status      *string
	StatusColor *string
}

var Descriptor = entity.Register[Receipt](
	entity.PrimaryKey("ID"),
	entity.ExternalKey("OrganizationID", "ExchangeID"),
)

func (r Receipt) DisplayValue(name string) (string, error) {
	switch name {
	case "UUID":
		return r.UUID, nil
	case "ExchangeID":
		return r.ExchangeID, nil
	case "Number":
		return r.Number, nil
	case "Date":
		return r.Date.Format("2006-01-02"), nil
	case "OrganizationName":
		return r.OrganizationName, nil
	case "UserLogin":
		return r.UserLogin, nil
	case "CustomerName":
		return r.CustomerName, nil
	case "Total":
		return formatFloat(r.Total), nil
	case "Status":
		return displayStatus(r), nil
	case "StatusColor":
		return r.StatusColor, nil
	default:
		return "", fmt.Errorf("unknown display field %q for Receipt", name)
	}
}

func (r Receipt) URL() string {
	return "/receipts/" + strconv.FormatInt(r.ID, 10)
}

func displayStatus(r Receipt) string {
	if r.Status != "" {
		return r.Status
	}
	if r.SentAt != nil {
		return "Отправлен"
	}
	return "Не отправлен"
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}
