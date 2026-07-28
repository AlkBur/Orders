package organizations

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"

	"Orders/internal/common"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) New() *Organization {
	return &Organization{
		UUID:   common.NilUUID,
		Active: true,
	}
}

func (s *Store) Get(ctx context.Context, id string) (*Organization, error) {
	o := &Organization{}

	err := s.db.QueryRowContext(ctx, `
		SELECT uuid, name, api_key, active, created_at, updated_at
		FROM organizations
		WHERE uuid = ?
	`, id).Scan(
		&o.UUID,
		&o.Name,
		&o.APIKey,
		&o.Active,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return o, nil
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

func (s *Store) Save(ctx context.Context, org *Organization) error {
	if common.IsNilUUID(org.UUID) {
		id, err := common.GenerateUUID()
		if err != nil {
			return err
		}
		org.UUID = id

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

	result, err := s.db.ExecContext(ctx, `
		UPDATE organizations
		SET name = ?, active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE uuid = ?
	`, org.Name, org.Active, org.UUID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM organizations WHERE uuid = ?
	`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) LoadAPIKeys(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT uuid, api_key
		FROM organizations
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make(map[string]string)
	for rows.Next() {
		var uuid, apiKey string
		if err := rows.Scan(&uuid, &apiKey); err != nil {
			return nil, err
		}
		keys[uuid] = apiKey
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ord_" + fmt.Sprintf("%x", b), nil
}
