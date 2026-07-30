package pages

import (
	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
)

type ReceiptCardPage struct {
	Title          string
	Receipt        *receipts.Receipt
	Items          []receipts.ReceiptItem
	Orgs           []*organizations.Organization
	Customers      []*customers.Customer
	Products       []*products.Product
	CustomerID     int64
	CustomerName   string
	OrganizationID int64
	Errors         map[string]string
	ErrorsJSON     string
	ItemsJSON      string
}
