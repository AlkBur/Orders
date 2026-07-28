package products

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"Orders/internal/common"
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
	return &Product{ID: common.NilUUID, OrganizationID: common.NilUUID, Active: true}
}

func (s *Store) Get(ctx context.Context, organizationID, id string) (*Product, error) {
	p := &Product{}
	err := s.db.QueryRowContext(ctx, `
		SELECT p.organization_id, p.id, p.name, p.unit, p.active,
		       p.created_at, p.updated_at, o.name
		FROM products p
		JOIN organizations o ON o.uuid = p.organization_id
		WHERE p.organization_id = ? AND p.id = ?
	`, organizationID, id).Scan(
		&p.OrganizationID, &p.ID, &p.Name, &p.Unit, &p.Active, &p.CreatedAt, &p.UpdatedAt, &p.OrganizationName,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) List(ctx context.Context, organizationID string) ([]*Product, error) {
	var rows *sql.Rows
	var err error

	query := `
		SELECT p.organization_id, p.id, p.name, p.unit, p.active,
		       p.created_at, p.updated_at, o.name
		FROM products p
		JOIN organizations o ON o.uuid = p.organization_id
	`
	args := []interface{}{}
	if !common.IsNilUUID(organizationID) {
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
			&p.OrganizationID, &p.ID, &p.Name, &p.Unit, &p.Active, &p.CreatedAt, &p.UpdatedAt, &p.OrganizationName,
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

func (s *Store) Save(ctx context.Context, p *Product) error {
	if common.IsNilUUID(p.OrganizationID) {
		return ErrOrganizationRequired
	}

	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM organizations WHERE uuid = ?)`, p.OrganizationID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check organization: %w", err)
	}
	if !exists {
		return ErrOrganizationNotFound
	}

	if common.IsNilUUID(p.ID) {
		id, err := common.GenerateUUID()
		if err != nil {
			return err
		}
		p.ID = id

		_, err = s.db.ExecContext(ctx, `
			INSERT INTO products (organization_id, id, name, unit, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, p.OrganizationID, p.ID, p.Name, p.Unit, p.Active)
		return err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE products
		SET name = ?, unit = ?, active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE organization_id = ? AND id = ?
	`, p.Name, p.Unit, p.Active, p.OrganizationID, p.ID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, organizationID, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM products
		WHERE organization_id = ? AND id = ?
	`, organizationID, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Synchronize(ctx context.Context, organizationID string, items []Product) (SyncResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SyncResult{}, err
	}
	defer tx.Rollback()

	seen := make(map[string]bool)
	for _, item := range items {
		if item.ID == "" || common.IsNilUUID(item.ID) {
			return SyncResult{}, fmt.Errorf("product id is required")
		}
		key := organizationID + ":" + item.ID
		if seen[key] {
			return SyncResult{}, fmt.Errorf("duplicate product id: %s", item.ID)
		}
		seen[key] = true
	}

	var result SyncResult

	for _, item := range items {
		res, err := tx.ExecContext(ctx, `
			UPDATE products
			SET name = ?, unit = ?, active = ?, updated_at = CURRENT_TIMESTAMP
			WHERE organization_id = ? AND id = ?
		`, item.Name, item.Unit, item.Active, organizationID, item.ID)
		if err != nil {
			return SyncResult{}, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			result.Updated++
			continue
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO products (organization_id, id, name, unit, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, organizationID, item.ID, item.Name, item.Unit, item.Active)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return SyncResult{}, fmt.Errorf("duplicate product id in organization: %s", item.ID)
			}
			return SyncResult{}, err
		}
		result.Inserted++
	}

	return result, tx.Commit()
}
