package products

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type SyncResult struct {
	Inserted int `json:"inserted"`
	Updated  int `json:"updated"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) New() *Product {
	return &Product{Active: true}
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Product, error) {
	p := &Product{}
	err := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.uuid, p.organization_id, p.name, p.unit, p.active,
		       p.created_at, p.updated_at, o.name
		FROM products p
		JOIN organizations o ON o.id = p.organization_id
		WHERE p.id = ?
	`, id).Scan(
		&p.ID, &p.UUID, &p.OrganizationID, &p.Name, &p.Unit, &p.Active,
		&p.CreatedAt, &p.UpdatedAt, &p.OrganizationName,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) GetByExternal(ctx context.Context, organizationID int64, uuid string) (*Product, error) {
	p := &Product{}
	err := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.uuid, p.organization_id, p.name, p.unit, p.active,
		       p.created_at, p.updated_at, o.name
		FROM products p
		JOIN organizations o ON o.id = p.organization_id
		WHERE p.organization_id = ? AND p.uuid = ?
	`, organizationID, uuid).Scan(
		&p.ID, &p.UUID, &p.OrganizationID, &p.Name, &p.Unit, &p.Active,
		&p.CreatedAt, &p.UpdatedAt, &p.OrganizationName,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) Count(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM products`).Scan(&n)
	return n, err
}

func (s *Store) List(ctx context.Context, organizationID int64) ([]*Product, error) {
	var rows *sql.Rows
	var err error

	query := `
		SELECT p.id, p.uuid, p.organization_id, p.name, p.unit, p.active,
		       p.created_at, p.updated_at, o.name
		FROM products p
		JOIN organizations o ON o.id = p.organization_id
	`
	args := []interface{}{}
	if organizationID > 0 {
		query += ` WHERE p.organization_id = ?`
		args = append(args, organizationID)
	}
	query += ` ORDER BY p.name`

	rows, err = s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		p := &Product{}
		if err := rows.Scan(
			&p.ID, &p.UUID, &p.OrganizationID, &p.Name, &p.Unit, &p.Active,
			&p.CreatedAt, &p.UpdatedAt, &p.OrganizationName,
		); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if products == nil {
		products = []*Product{}
	}

	return products, nil
}

// Save inserts or updates a product.
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
func (s *Store) Save(ctx context.Context, p *Product) error {
	if p.OrganizationID == 0 {
		return ErrOrganizationRequired
	}

	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM organizations WHERE id = ?)`, p.OrganizationID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check organization: %w", err)
	}
	if !exists {
		return ErrOrganizationNotFound
	}

	if p.ID == 0 {
		if p.UUID == "" {
			return ErrEmptyUUID
		}

		result, err := s.db.ExecContext(ctx, `
			INSERT INTO products (uuid, organization_id, name, unit, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, p.UUID, p.OrganizationID, p.Name, p.Unit, p.Active)
		if err != nil {
			return err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		p.ID = id
		return nil
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE products
		SET uuid = ?, name = ?, unit = ?, active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, p.UUID, p.Name, p.Unit, p.Active, p.ID)
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
	result, err := s.db.ExecContext(ctx, `DELETE FROM products WHERE id = ?`, id)
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
		DELETE FROM products WHERE organization_id = ? AND uuid = ?
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

func (s *Store) Synchronize(ctx context.Context, organizationID int64, items []Product) (SyncResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SyncResult{}, err
	}
	defer tx.Rollback()

	seen := make(map[string]bool)
	for _, item := range items {
		if item.UUID == "" {
			return SyncResult{}, fmt.Errorf("product uuid is required")
		}
		if seen[item.UUID] {
			return SyncResult{}, fmt.Errorf("duplicate product uuid: %s", item.UUID)
		}
		seen[item.UUID] = true
	}

	var result SyncResult

	for _, item := range items {
		res, err := tx.ExecContext(ctx, `
			UPDATE products
			SET name = ?, unit = ?, active = ?, updated_at = CURRENT_TIMESTAMP
			WHERE organization_id = ? AND uuid = ?
		`, item.Name, item.Unit, item.Active, organizationID, item.UUID)
		if err != nil {
			return SyncResult{}, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			result.Updated++
			continue
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO products (uuid, organization_id, name, unit, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, item.UUID, organizationID, item.Name, item.Unit, item.Active)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return SyncResult{}, fmt.Errorf("duplicate product uuid in organization: %s", item.UUID)
			}
			return SyncResult{}, err
		}
		result.Inserted++
	}

	return result, tx.Commit()
}
