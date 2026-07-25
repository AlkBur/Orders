package database

import "database/sql"

type SchemaFunc func(*sql.DB) error

var schemas []SchemaFunc

func RegisterSchema(fn SchemaFunc) {
	schemas = append(schemas, fn)
}

func InitSchema(db *sql.DB) error {
	for _, fn := range schemas {
		if err := fn(db); err != nil {
			return err
		}
	}
	return nil
}
