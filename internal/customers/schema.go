package customers

import "Orders/internal/database"

var Table = database.Must(database.NewTable("customers",
	database.Int("id").PrimaryKey().AutoIncrement(),
	database.String("uuid").NotNull(),
	database.Int("organization_id").NotNull().References("organizations", "id").OnDelete("CASCADE"),
	database.String("name").NotNull(),
	database.Bool("active").NotNull().Default(true),
	database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
	database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
)).AddUniqueConstraint("organization_id", "uuid")
