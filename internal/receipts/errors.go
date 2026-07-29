package receipts

import "errors"

var (
	ErrNotFound             = errors.New("receipt not found")
	ErrReceiptReadOnly      = errors.New("receipt has already been sent and cannot be modified")
	ErrUUIDAlreadyAssigned  = errors.New("uuid already assigned to this receipt")
	ErrExchangeIDNotFound   = errors.New("receipt with given exchange_id not found")
	ErrEmptyUUID            = errors.New("uuid is required")
)
