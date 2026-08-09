package ui

import (
	"html/template"
	"strings"

	"Orders/internal/ui/display"
)

// Funcs returns the shared template functions of the platform.
// Every page template is parsed with these functions.
//
// basePath — базовый префикс публикации приложения (например /office).
// Шаблонная функция url превращает внутренний путь (/static/css/app.css)
// в полностью квалифицированный URL (/office/static/css/app.css).
// В шаблонах необходимо использовать {{ url "/..." }} вместо хардкода,
// чтобы ссылки не ломались при размещении приложения не в корне домена.
func Funcs(basePath string) template.FuncMap {
	return template.FuncMap{
		"icon":         icon,
		"getFAB":       getFAB,
		"formatNumber": func(v int64) string { return display.FormatNumber(v) },
		"url":          func(path string) string { return joinBasePath(basePath, path) },
	}
}

// joinBasePath добавляет basePath к внутреннему абсолютному пути.
// Контракт совпадает с app.URL: path — внутренний путь без префикса.
func joinBasePath(basePath, path string) string {
	if basePath == "" || path == "" || !strings.HasPrefix(path, "/") {
		return path
	}
	return basePath + path
}
