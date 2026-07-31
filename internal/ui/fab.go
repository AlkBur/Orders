package ui

// FAB — платформенный компонент для главного действия на мобильных.
// Отдельная модель от Button: может эволюционировать независимо.
type FAB struct {
	Icon string
	Text string
	URL  string
}

// Temporary solution.
// FAB will move to common LayoutModel when it appears.
type FABProvider interface {
	FAB() *FAB
}

// getFAB returns the FAB of the page data if it implements FABProvider.
func getFAB(data any) *FAB {
	if p, ok := data.(FABProvider); ok {
		return p.FAB()
	}
	return nil
}
