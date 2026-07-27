package users

import "Orders/internal/database"

var Table = database.Must(database.NewTable("users",
	database.Int("id").PrimaryKey().AutoIncrement(),
	database.String("login").NotNull().Unique(),
	database.String("password_hash").NotNull().Default(""),
	database.Bool("is_admin").NotNull().Default(false),
	database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
	database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
))
