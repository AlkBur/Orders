package customers

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

func (s *Store) New() *Customer {
	return &Customer{ID: common.NilUUID, Active: true}
}

func (s *Store) Get(ctx context.Context, organizationID, id string) (*Customer, error) {
	c := &Customer{}
	err := s.db.QueryRowContext(ctx, `
		SELECT organization_id, id, name, active, created_at, updated_at
		FROM customers
		WHERE organization_id = ? AND id = ?
	`, organizationID, id).Scan(
		&c.OrganizationID, &c.ID, &c.Name, &c.Active, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) List(ctx context.Context, organizationID string) ([]*Customer, error) {
	var rows *sql.Rows
	var err error

	if common.IsNilUUID(organizationID) {
		rows, err = s.db.QueryContext(ctx, `
			SELECT organization_id, id, name, active, created_at, updated_at
			FROM customers
			ORDER BY name
		`)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT organization_id, id, name, active, created_at, updated_at
			FROM customers
			WHERE organization_id = ?
			ORDER BY name
		`, organizationID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []*Customer
	for rows.Next() {
		c := &Customer{}
		if err := rows.Scan(
			&c.OrganizationID, &c.ID, &c.Name, &c.Active, &c.CreatedAt, &c.UpdatedAt,
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

func (s *Store) Save(ctx context.Context, c *Customer) error {
	if common.IsNilUUID(c.OrganizationID) {
		return ErrOrganizationRequired
	}

	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM organizations WHERE uuid = ?)`, c.OrganizationID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check organization: %w", err)
	}
	if !exists {
		return ErrOrganizationNotFound
	}

	if common.IsNilUUID(c.ID) {
		id, err := common.GenerateUUID()
		if err != nil {
			return err
		}
		c.ID = id

		_, err = s.db.ExecContext(ctx, `
			INSERT INTO customers (organization_id, id, name, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, c.OrganizationID, c.ID, c.Name, c.Active)
		return err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE customers
		SET name = ?, active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE organization_id = ? AND id = ?
	`, c.Name, c.Active, c.OrganizationID, c.ID)
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
		DELETE FROM customers
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

func (s *Store) Synchronize(ctx context.Context, organizationID string, items []Customer) (SyncResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SyncResult{}, err
	}
	defer tx.Rollback()

	// Validate: no duplicates in request
	seen := make(map[string]bool)
	for _, item := range items {
		if common.IsNilUUID(item.ID) {
			return SyncResult{}, fmt.Errorf("customer id is required")
		}
		key := organizationID + ":" + item.ID
		if seen[key] {
			return SyncResult{}, fmt.Errorf("duplicate customer id: %s", item.ID)
		}
		seen[key] = true
	}

	var result SyncResult

	for _, item := range items {
		active := item.Active

		res, err := tx.ExecContext(ctx, `
			UPDATE customers
			SET name = ?, active = ?, updated_at = CURRENT_TIMESTAMP
			WHERE organization_id = ? AND id = ?
		`, item.Name, active, organizationID, item.ID)
		if err != nil {
			return SyncResult{}, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			result.Updated++
			continue
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO customers (organization_id, id, name, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, organizationID, item.ID, item.Name, active)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return SyncResult{}, fmt.Errorf("duplicate customer id in organization: %s", item.ID)
			}
			return SyncResult{}, err
		}
		result.Inserted++
	}

	return result, tx.Commit()
}
