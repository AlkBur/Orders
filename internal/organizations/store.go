package organizations

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUUIDRequired = errors.New("uuid is required")
	ErrNameRequired = errors.New("name is required")
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(ctx context.Context) ([]*Organization, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			uuid,
			name,
			active,
			created_at,
			updated_at
		FROM organizations
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		o := &Organization{}
		if err := rows.Scan(
			&o.UUID,
			&o.Name,
			&o.Active,
			&o.CreatedAt,
			&o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if orgs == nil {
		orgs = []*Organization{}
	}

	return orgs, nil
}

func (s *Store) New() *Organization {
	return &Organization{
		Active: true,
	}
}

func (s *Store) Save(ctx context.Context, org *Organization) error {
	if strings.TrimSpace(org.UUID) == "" {
		return ErrUUIDRequired
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE organizations
		SET name = ?, active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE uuid = ?
	`, org.Name, org.Active, org.UUID)
	if err != nil {
		return err
	}

	n, _ := result.RowsAffected()
	if n > 0 {
		return nil
	}

	if org.APIKey == "" {
		key, err := generateAPIKey()
		if err != nil {
			return err
		}
		org.APIKey = key
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO organizations (uuid, name, api_key, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, org.UUID, org.Name, org.APIKey, org.Active)
	return err
}

func (s *Store) GetByUUID(ctx context.Context, uuid string) (*Organization, error) {
	o := &Organization{}

	err := s.db.QueryRowContext(ctx, `
		SELECT uuid, name, api_key, active, created_at, updated_at
		FROM organizations
		WHERE uuid = ?
	`, uuid).Scan(
		&o.UUID,
		&o.Name,
		&o.APIKey,
		&o.Active,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ord_" + fmt.Sprintf("%x", b), nil
}
