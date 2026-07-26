package customers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type SyncResult struct {
	Inserted    int `json:"inserted"`
	Updated     int `json:"updated"`
	Deactivated int `json:"deactivated"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Synchronize(ctx context.Context, snapshots []CustomerSnapshot) (SyncResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SyncResult{}, err
	}
	defer tx.Rollback()

	insertStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO customers (uuid, name, active, created_at, updated_at)
		SELECT ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		WHERE NOT EXISTS (SELECT 1 FROM customers WHERE uuid = ?)
	`)
	if err != nil {
		return SyncResult{}, err
	}
	defer insertStmt.Close()

	updateStmt, err := tx.PrepareContext(ctx, `
		UPDATE customers
		SET name = ?, active = 1, updated_at = CURRENT_TIMESTAMP
		WHERE uuid = ? AND (name != ? OR active = 0)
	`)
	if err != nil {
		return SyncResult{}, err
	}
	defer updateStmt.Close()

	var result SyncResult

	for _, sn := range snapshots {
		res, err := insertStmt.ExecContext(ctx, sn.UUID, sn.Name, sn.UUID)
		if err != nil {
			return SyncResult{}, err
		}
		n, _ := res.RowsAffected()
		result.Inserted += int(n)
	}

	for _, sn := range snapshots {
		res, err := updateStmt.ExecContext(ctx, sn.Name, sn.UUID, sn.Name)
		if err != nil {
			return SyncResult{}, err
		}
		n, _ := res.RowsAffected()
		result.Updated += int(n)
	}

	if len(snapshots) == 0 {
		res, err := tx.ExecContext(ctx, `
			UPDATE customers SET active = 0, updated_at = CURRENT_TIMESTAMP
			WHERE active = 1
		`)
		if err != nil {
			return SyncResult{}, err
		}
		n, _ := res.RowsAffected()
		result.Deactivated = int(n)
	} else {
		placeholders := make([]string, len(snapshots))
		args := make([]any, len(snapshots))
		for i, sn := range snapshots {
			placeholders[i] = "?"
			args[i] = sn.UUID
		}
		query := fmt.Sprintf(
			`UPDATE customers SET active = 0, updated_at = CURRENT_TIMESTAMP WHERE active = 1 AND uuid NOT IN (%s)`,
			strings.Join(placeholders, ","),
		)
		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return SyncResult{}, err
		}
		n, _ := res.RowsAffected()
		result.Deactivated = int(n)
	}

	return result, tx.Commit()
}
