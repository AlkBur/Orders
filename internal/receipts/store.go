package receipts

import (
	"context"
	"database/sql"
	"time"

	"Orders/internal/common"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) New() *Receipt {
	return &Receipt{}
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Document, error) {
	r, err := scanReceipt(s.db.QueryRowContext(ctx, `
		SELECT
			r.id, r.uuid, r.exchange_id, r.number, r.date,
			r.organization_id, COALESCE(o.name, '') AS org_name,
			r.user_id, COALESCE(u.login, '') AS user_login,
			r.customer_id, COALESCE(c.name, '') AS customer_name,
			r.total, r.sent_at, r.status, r.status_color,
			r.created_at, r.updated_at
		FROM receipts r
		LEFT JOIN organizations o ON o.id = r.organization_id
		LEFT JOIN users u ON u.id = r.user_id
		LEFT JOIN customers c ON c.id = r.customer_id
		WHERE r.id = ?
	`, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	items, err := s.listItems(ctx, id)
	if err != nil {
		return nil, err
	}

	return &Document{Receipt: r, Items: items}, nil
}

func (s *Store) Count(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM receipts`).Scan(&n)
	return n, err
}

func (s *Store) List(ctx context.Context) ([]*Receipt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			r.id, r.uuid, r.exchange_id, r.number, r.date,
			r.organization_id, COALESCE(o.name, '') AS org_name,
			r.user_id, COALESCE(u.login, '') AS user_login,
			r.customer_id, COALESCE(c.name, '') AS customer_name,
			r.total, r.sent_at, r.status, r.status_color,
			r.created_at, r.updated_at
		FROM receipts r
		LEFT JOIN organizations o ON o.id = r.organization_id
		LEFT JOIN users u ON u.id = r.user_id
		LEFT JOIN customers c ON c.id = r.customer_id
		ORDER BY r.date DESC, r.id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Receipt
	for rows.Next() {
		r, err := scanReceipt(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if list == nil {
		list = []*Receipt{}
	}
	return list, nil
}

func (s *Store) Save(ctx context.Context, doc *Document) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r := doc.Receipt

	if r.ID == 0 {
		uuid, err := common.GenerateUUID()
		if err != nil {
			return err
		}
		r.ExchangeID = uuid

		now := time.Now()
		r.CreatedAt = now
		r.UpdatedAt = now

		var uuidArg any
		if r.UUID != "" {
			uuidArg = r.UUID
		}

		var sentAtArg any
		if r.SentAt != nil {
			sentAtArg = r.SentAt.Format(time.RFC3339)
		}

		result, err := tx.ExecContext(ctx, `
			INSERT INTO receipts (uuid, exchange_id, number, date,
				organization_id, user_id, customer_id, total, sent_at,
				status, status_color, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuidArg, r.ExchangeID, r.Number, r.Date.Format("2006-01-02"),
			r.OrganizationID, r.UserID, r.CustomerID, r.Total,
			sentAtArg, r.Status, r.StatusColor,
			r.CreatedAt.Format(time.RFC3339), r.UpdatedAt.Format(time.RFC3339))
		if err != nil {
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		r.ID = id
	} else {
		r.UpdatedAt = time.Now()

		var uuidArg any
		if r.UUID != "" {
			uuidArg = r.UUID
		}

		var sentAtArg any
		if r.SentAt != nil {
			sentAtArg = r.SentAt.Format(time.RFC3339)
		}

		_, err := tx.ExecContext(ctx, `
			UPDATE receipts SET
				uuid = ?, number = ?, date = ?,
				organization_id = ?, user_id = ?, customer_id = ?,
				total = ?, sent_at = ?, status = ?, status_color = ?,
				updated_at = ?
			WHERE id = ?
		`, uuidArg, r.Number, r.Date.Format("2006-01-02"),
			r.OrganizationID, r.UserID, r.CustomerID, r.Total,
			sentAtArg, r.Status, r.StatusColor,
			r.UpdatedAt.Format(time.RFC3339), r.ID)
		if err != nil {
			return err
		}
	}

	if err := saveItemsTx(tx, r.ID, doc.Items); err != nil {
		return err
	}

	return tx.Commit()
}

func saveItemsTx(tx *sql.Tx, receiptID int64, items []ReceiptItem) error {
	if _, err := tx.ExecContext(context.Background(),
		`DELETE FROM receipt_items WHERE receipt_id = ?`, receiptID); err != nil {
		return err
	}

	for _, item := range items {
		_, err := tx.ExecContext(context.Background(), `
			INSERT INTO receipt_items
				(receipt_id, line_num, product_id, unit, quantity, price, amount)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, receiptID, item.LineNum, item.ProductID, item.Unit,
			item.Quantity, item.Price, item.Amount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) DeleteByID(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM receipts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Synchronize(ctx context.Context, updates []ReceiptUpdate) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, upd := range updates {
		var currentUUID sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT uuid FROM receipts WHERE exchange_id = ?`, upd.ExchangeID).Scan(&currentUUID)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrExchangeIDNotFound
			}
			return err
		}

		if upd.UUID != nil {
			if currentUUID.Valid && *upd.UUID != currentUUID.String {
				return ErrUUIDAlreadyAssigned
			}
		}

		var sets []string
		var args []any

		if upd.UUID != nil {
			sets = append(sets, "uuid = ?")
			args = append(args, *upd.UUID)
		}
		if upd.Status != nil {
			sets = append(sets, "status = ?")
			args = append(args, *upd.Status)
		}
		if upd.StatusColor != nil {
			sets = append(sets, "status_color = ?")
			args = append(args, *upd.StatusColor)
		}

		if len(sets) == 0 {
			continue
		}

		sets = append(sets, "updated_at = ?")
		args = append(args, time.Now().Format(time.RFC3339))
		args = append(args, upd.ExchangeID)

		query := "UPDATE receipts SET "
		for i, set := range sets {
			if i > 0 {
				query += ", "
			}
			query += set
		}
		query += " WHERE exchange_id = ?"

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) listItems(ctx context.Context, receiptID int64) ([]ReceiptItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			i.id, i.receipt_id, i.line_num, i.product_id,
			COALESCE(p.name, '') AS product_name,
			i.unit, i.quantity, i.price, i.amount
		FROM receipt_items i
		LEFT JOIN products p ON p.id = i.product_id
		WHERE i.receipt_id = ?
		ORDER BY i.line_num
	`, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ReceiptItem
	for rows.Next() {
		var item ReceiptItem
		if err := rows.Scan(
			&item.ID, &item.ReceiptID, &item.LineNum, &item.ProductID,
			&item.ProductName, &item.Unit, &item.Quantity, &item.Price, &item.Amount,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if items == nil {
		items = []ReceiptItem{}
	}
	return items, nil
}

func scanReceipt(row interface {
	Scan(dest ...any) error
}) (*Receipt, error) {
	r := &Receipt{}

	var uuid sql.NullString
	var sentAt sql.NullTime
	var dateStr, createdAt, updatedAt string

	err := row.Scan(
		&r.ID, &uuid, &r.ExchangeID, &r.Number, &dateStr,
		&r.OrganizationID, &r.OrganizationName,
		&r.UserID, &r.UserLogin,
		&r.CustomerID, &r.CustomerName,
		&r.Total, &sentAt, &r.Status, &r.StatusColor,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if uuid.Valid {
		r.UUID = uuid.String
	}
	if sentAt.Valid {
		r.SentAt = &sentAt.Time
	}
	if dateStr != "" {
		r.Date, _ = time.Parse("2006-01-02", dateStr[:10])
	}

	return r, nil
}
