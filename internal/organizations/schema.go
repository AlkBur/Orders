package organizations

import "Orders/internal/database"

var Table = database.Must(database.NewTable("organizations",
	database.String("uuid").PrimaryKey(),
	database.String("name").NotNull(),
	database.String("api_key").NotNull().Unique(),
	database.Bool("active").NotNull().Default(true),
	database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
	database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
))
