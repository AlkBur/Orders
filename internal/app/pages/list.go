package pages

type Column struct {
	Name  string
	Label string
}

type RowAction struct {
	Label   string
	BaseURL string
}

type Row struct {
	Cells []string
	ID    string

	// URL overrides RowAction.BaseURL + ID when the row
	// represents a resource with its own canonical address.
	URL string
}

type ListPage struct {
	Title     string
	Search    string
	Columns   []Column
	Rows      []Row

	NewURL    string
	RowAction RowAction
	SearchURL string

	EmptyText string
}
