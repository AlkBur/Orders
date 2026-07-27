package customers

import "Orders/internal/database"

var Table = database.Must(database.NewTable("customers",
	database.String("uuid").PrimaryKey(),
	database.String("name").NotNull(),
	database.Bool("active").NotNull().Default(true),
	database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
	database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
))
