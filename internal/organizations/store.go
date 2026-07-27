package organizations

import (
	"context"
	"database/sql"
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
