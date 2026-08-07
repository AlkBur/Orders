package pages

// Page — базовая модель страницы, отображаемая через RenderPage или
// RenderAuthPage. Содержит данные, которые использует layout (base.html,
// auth.html): в настоящий момент это Title. Новые поля, требуемые layout
// (Description, PageClass, Breadcrumbs и т.п.), добавляются сюда.
type Page struct {
	Title string
}