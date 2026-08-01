package ui

// MenuItem — пункт меню приложения (AppMenu) в шапке.
type MenuItem struct {
	ID        string
	Label     string
	Icon      string
	URL       string
	Danger    bool
	Separator bool
}
