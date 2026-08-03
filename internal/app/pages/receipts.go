package pages

import (
	"Orders/internal/receipts"
	"Orders/internal/ui"
)

type ReceiptOrganizationOption struct {
	ID   int64
	Name string
}

type ReceiptCardPage struct {
	Header         ui.HeaderData
	Title          string
	FormAction     string
	Card           ui.CardData
	Receipt        *receipts.Receipt
	Items          []receipts.ReceiptItem
	Orgs           []ReceiptOrganizationOption
	CustomersJSON  string
	ProductsJSON   string
	CustomerID     int64
	CustomerName   string
	OrganizationID int64
	Errors         map[string]string
	ErrorsJSON     string
	ItemsJSON      string
}
