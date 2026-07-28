package app

import (
	"context"
	"database/sql"

	"Orders/internal/customers"
	"Orders/internal/database"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/sessions"
	"Orders/internal/users"
)

func NewSchema() *database.Schema {
	s := database.NewSchema()

	if err := s.Register(users.Table); err != nil {
		panic(err)
	}
	if err := s.Register(sessions.Table); err != nil {
		panic(err)
	}
	if err := s.Register(customers.Table); err != nil {
		panic(err)
	}
	if err := s.Register(organizations.Table); err != nil {
		panic(err)
	}
	if err := s.Register(products.Table); err != nil {
		panic(err)
	}

	registerMigrations(s)

	return s
}

func registerMigrations(s *database.Schema) {
	s.AddMigration(database.Migration{
		Version: 2,
		Name:    "Redesign customers: composite PK (organization_id, id)",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			// Временное решение для MVP: удаление таблицы customers.
			// Данные восстанавливаются из 1С через API синхронизации.
			// В следующих версиях миграции должны сохранять данные пользователей.
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS customers`); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, customers.Table.CreateSQL()); err != nil {
				return err
			}
			return nil
		},
	})

	s.AddMigration(database.Migration{
		Version: 3,
		Name:    "Add products table",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, products.Table.CreateSQL()); err != nil {
				return err
			}
			return nil
		},
	})
}

