package sessions

import "Orders/internal/database"

var Table = database.Must(database.NewTable("sessions",
	database.String("id").PrimaryKey(),
	database.Int("user_id").References("users", "id").OnDelete("SET NULL"),
	database.String("flash_type").NotNull().Default(""),
	database.String("flash_message").NotNull().Default(""),
	database.String("values_json").NotNull().Default("{}"),
	database.String("user_agent").NotNull().Default(""),
	database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
	database.DateTime("last_seen_at").NotNull().Default("CURRENT_TIMESTAMP"),
	database.DateTime("expires_at").NotNull(),
))
