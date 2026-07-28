package products

import "Orders/internal/database"

var Table = func() database.Table {
	return database.Must(database.NewTable("products",
		database.String("organization_id").NotNull(),
		database.String("id").NotNull(),
		database.String("name").NotNull(),
		database.String("unit").NotNull().Default(""),
		database.Bool("active").NotNull().Default(true),
		database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
		database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
	)).SetPrimaryKey("organization_id", "id")
}()
