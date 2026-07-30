package pages

import (
	"Orders/internal/customers"
	"Orders/internal/organizations"
	"Orders/internal/receipts"
)

type ReceiptCardPage struct {
	Title     string
	Receipt   *receipts.Receipt
	Items     []receipts.ReceiptItem
	Orgs      []*organizations.Organization
	Customers []*customers.Customer
}
