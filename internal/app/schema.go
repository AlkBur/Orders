package app

import (
	"context"
	"database/sql"

	"Orders/internal/customers"
	"Orders/internal/database"
	"Orders/internal/organizations"
	"Orders/internal/products"
	"Orders/internal/receipts"
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
	if err := s.Register(receipts.Table); err != nil {
		panic(err)
	}
	if err := s.Register(receipts.ItemsTable); err != nil {
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
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS products`); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, products.Table.CreateSQL()); err != nil {
				return err
			}
			return nil
		},
	})

	s.AddMigration(database.Migration{
		Version: 4,
		Name:    "Unified schema: internal ID (INTEGER PK) + external UUID for all dictionaries",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			for _, name := range []string{"sessions", "customers", "products", "organizations", "users"} {
				if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS `+name); err != nil {
					return err
				}
			}
			tables := []database.Table{
				users.Table,
				sessions.Table,
				customers.Table,
				products.Table,
				organizations.Table,
			}
			for _, t := range tables {
				if _, err := tx.ExecContext(ctx, t.CreateSQL()); err != nil {
					return err
				}
			}
			return nil
		},
	})

	s.AddMigration(database.Migration{
		Version: 5,
		Name:    "Add receipts tables",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, receipts.Table.CreateSQLIfNotExists()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, receipts.ItemsTable.CreateSQLIfNotExists()); err != nil {
				return err
			}
			return nil
		},
	})
}
