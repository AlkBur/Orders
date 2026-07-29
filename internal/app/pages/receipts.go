package pages

import "Orders/internal/receipts"

type ReceiptListPage struct {
	Title     string
	Columns   []Column
	Rows      []Row
	EmptyText string
}

type ReceiptCardPage struct {
	Title   string
	Receipt *receipts.Receipt
	Items   []receipts.ReceiptItem
}
