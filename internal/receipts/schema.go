package receipts

import "Orders/internal/database"

var Table = database.Must(database.NewTable("receipts",
	database.Int("id").PrimaryKey().AutoIncrement(),
	database.String("uuid").Unique(),
	database.String("exchange_id").NotNull().Unique(),
	database.String("number").NotNull(),
	database.String("date").NotNull(),
	database.Int("organization_id").NotNull().References("organizations", "id"),
	database.Int("user_id").NotNull().References("users", "id"),
	database.Int("customer_id").NotNull().References("customers", "id"),
	database.Real("total").NotNull().Default(0),
	database.DateTime("sent_at"),
	database.String("status").NotNull().Default(""),
	database.String("status_color").NotNull().Default(""),
	database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
	database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
)).AddUniqueConstraint("organization_id", "number")

var ItemsTable = database.Must(database.NewTable("receipt_items",
	database.Int("id").PrimaryKey().AutoIncrement(),
	database.Int("receipt_id").NotNull().References("receipts", "id").OnDelete("CASCADE"),
	database.Int("line_num").NotNull(),
	database.Int("product_id").NotNull().References("products", "id"),
	database.String("unit").NotNull().Default(""),
	database.Real("quantity").NotNull().Default(1),
	database.Real("price").NotNull().Default(0),
	database.Real("amount").NotNull().Default(0),
))
