package app

import (
	"bytes"
	"text/template"
)

// notifyData — данные шаблонов уведомлений. User — инициатор события (actor),
// а не автор документа.
type notifyData struct {
	Number       string
	Date         string
	Organization string
	Customer     string
	Total        string
	User         string
	Status       string
	Action       string
}

// statusTemplates — текст уведомления об изменении статуса.
// Доступные плейсхолдеры: {{.Number}}, {{.Date}}, {{.Organization}},
// {{.Customer}}, {{.Total}}, {{.User}}, {{.Status}}.
var statusTemplates = template.Must(template.New("status").Parse(`
{{define "title"}}Чек №{{.Number}}: статус {{.Status}}{{end}}
{{define "body"}}Организация: {{.Organization}}
Клиент: {{.Customer}}
Сумма: {{.Total}}
Дата: {{.Date}}
Пользователь: {{.User}}
Статус: {{.Status}}{{end}}
`))

// actionTemplates — текст уведомления об изменении действия.
// Доступные плейсхолдеры: {{.Number}}, {{.Date}}, {{.Organization}},
// {{.Customer}}, {{.Total}}, {{.User}}, {{.Action}}. Отмена действия
// передаётся как Action = "отменено".
var actionTemplates = template.Must(template.New("action").Parse(`
{{define "title"}}Чек №{{.Number}}: действие {{.Action}}{{end}}
{{define "body"}}Организация: {{.Organization}}
Клиент: {{.Customer}}
Сумма: {{.Total}}
Дата: {{.Date}}
Пользователь: {{.User}}
Действие: {{.Action}}{{end}}
`))

// renderNotify выполняет именованную часть ("title" или "body") шаблона.
func renderNotify(tmpl *template.Template, name string, data notifyData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
