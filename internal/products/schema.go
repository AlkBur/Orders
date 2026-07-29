package products

import "Orders/internal/database"

var Table = database.Must(database.NewTable("products",
	database.Int("id").PrimaryKey().AutoIncrement(),
	database.String("uuid").NotNull(),
	database.Int("organization_id").NotNull().References("organizations", "id").OnDelete("CASCADE"),
	database.String("name").NotNull(),
	database.String("unit").NotNull().Default(""),
	database.Bool("active").NotNull().Default(true),
	database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
	database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
)).AddUniqueConstraint("organization_id", "uuid")
