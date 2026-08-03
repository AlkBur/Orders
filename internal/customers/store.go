package customers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"Orders/internal/database/search"
	"Orders/internal/entity"
)

type SyncResult struct {
	Inserted int `json:"inserted"`
	Updated  int `json:"updated"`
}

// ListOptions управляет выборкой списка контрагентов.
type ListOptions struct {
	Query   string
	Limit   int
	Offset  int
	OrderBy string
}

type Store struct {
	db *sql.DB
}

// customerSearchColumns — поисковые колонки списка контрагентов.
var customerSearchColumns = []search.MappedColumn{
	{Field: entity.FieldNameName, Expression: "c.name"},
	{Field: entity.FieldNameOrganizationName, Expression: "o.name"},
}

func (s *Store) searchableColumns() []search.MappedColumn {
	return customerSearchColumns
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) New() *Customer {
	return &Customer{Active: true}
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Customer, error) {
	c := &Customer{}
	err := s.db.QueryRowContext(ctx, `
		SELECT c.id, c.uuid, c.organization_id, c.name, c.active, c.created_at, c.updated_at, o.name
		FROM customers c
		JOIN organizations o ON o.id = c.organization_id
		WHERE c.id = ?
	`, id).Scan(
		&c.ID, &c.UUID, &c.OrganizationID, &c.Name, &c.Active, &c.CreatedAt, &c.UpdatedAt, &c.OrganizationName,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) GetByExternal(ctx context.Context, organizationID int64, uuid string) (*Customer, error) {
	c := &Customer{}
	err := s.db.QueryRowContext(ctx, `
		SELECT c.id, c.uuid, c.organization_id, c.name, c.active, c.created_at, c.updated_at, o.name
		FROM customers c
		JOIN organizations o ON o.id = c.organization_id
		WHERE c.organization_id = ? AND c.uuid = ?
	`, organizationID, uuid).Scan(
		&c.ID, &c.UUID, &c.OrganizationID, &c.Name, &c.Active, &c.CreatedAt, &c.UpdatedAt, &c.OrganizationName,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) Count(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM customers`).Scan(&n)
	return n, err
}

// List возвращает контрагентов. visibleFields — поля, отображаемые в списке:
// поиск выполняется только по ним.
func (s *Store) List(ctx context.Context, organizationID int64, opts ListOptions, visibleFields []entity.FieldName) ([]*Customer, error) {
	var rows *sql.Rows
	var err error

	query := `
		SELECT c.id, c.uuid, c.organization_id, c.name, c.active, c.created_at, c.updated_at, o.name
		FROM customers c
		JOIN organizations o ON o.id = c.organization_id
	`
	var conds []string
	var args []interface{}
	if organizationID > 0 {
		conds = append(conds, `c.organization_id = ?`)
		args = append(args, organizationID)
	}

	where, whereArgs := search.BuildWhere(
		search.VisibleColumns(s.searchableColumns(), visibleFields),
		search.NormalizeQuery(opts.Query),
	)
	if where != "" {
		conds = append(conds, where)
		args = append(args, whereArgs...)
	}

	if len(conds) > 0 {
		query += ` WHERE ` + strings.Join(conds, ` AND `)
	}
	query += ` ORDER BY c.name`

	rows, err = s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []*Customer
	for rows.Next() {
		c := &Customer{}
		if err := rows.Scan(
			&c.ID, &c.UUID, &c.OrganizationID, &c.Name, &c.Active, &c.CreatedAt, &c.UpdatedAt, &c.OrganizationName,
		); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if customers == nil {
		customers = []*Customer{}
	}

	return customers, nil
}

// Save inserts or updates a customer.
//
// INSERT (ID == 0):
//   - UUID must already be assigned.
//   - Returns ErrEmptyUUID if UUID is empty.
//   - OrganizationID must be set. Returns ErrOrganizationRequired if not.
//   - Returns ErrOrganizationNotFound if the organization does not exist.
//
// UPDATE (ID > 0):
//   - Updates the existing record by ID.
//   - Returns ErrNotFound if the record does not exist.
func (s *Store) Save(ctx context.Context, c *Customer) error {
	if c.OrganizationID == 0 {
		return ErrOrganizationRequired
	}

	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM organizations WHERE id = ?)`, c.OrganizationID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check organization: %w", err)
	}
	if !exists {
		return ErrOrganizationNotFound
	}

	if c.ID == 0 {
		if c.UUID == "" {
			return ErrEmptyUUID
		}

		result, err := s.db.ExecContext(ctx, `
			INSERT INTO customers (uuid, organization_id, name, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, c.UUID, c.OrganizationID, c.Name, c.Active)
		if err != nil {
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		c.ID = id
		return nil
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE customers
		SET uuid = ?, name = ?, active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, c.UUID, c.Name, c.Active, c.ID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteByID(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM customers WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteByExternal(ctx context.Context, organizationID int64, uuid string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM customers WHERE organization_id = ? AND uuid = ?
	`, organizationID, uuid)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Synchronize(ctx context.Context, organizationID int64, items []Customer) (SyncResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SyncResult{}, err
	}
	defer tx.Rollback()

	seen := make(map[string]bool)
	for _, item := range items {
		if item.UUID == "" {
			return SyncResult{}, fmt.Errorf("customer uuid is required")
		}
		key := fmt.Sprintf("%d:%s", organizationID, item.UUID)
		if seen[key] {
			return SyncResult{}, fmt.Errorf("duplicate customer uuid: %s", item.UUID)
		}
		seen[key] = true
	}

	var result SyncResult

	for _, item := range items {
		active := item.Active

		res, err := tx.ExecContext(ctx, `
			UPDATE customers
			SET name = ?, active = ?, updated_at = CURRENT_TIMESTAMP
			WHERE organization_id = ? AND uuid = ?
		`, item.Name, active, organizationID, item.UUID)
		if err != nil {
			return SyncResult{}, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			result.Updated++
			continue
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO customers (uuid, organization_id, name, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, item.UUID, organizationID, item.Name, active)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return SyncResult{}, fmt.Errorf("duplicate customer uuid in organization: %s", item.UUID)
			}
			return SyncResult{}, err
		}
		result.Inserted++
	}

	return result, tx.Commit()
}
