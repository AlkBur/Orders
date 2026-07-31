# UI Architecture

Version: 1.0

---

# Goal

Orders uses a lightweight Server Side Rendering (SSR) user interface.

The UI must:

- be fast;
- work without JavaScript;
- support desktop, tablet and mobile devices;
- have a consistent appearance;
- be PWA Ready;
- remain easy to maintain.

---

# Architecture

Rendering model:

- Server Side Rendering (SSR)
- html/template
- embed.FS

Templates are embedded into the application binary.

Static resources are served by the HTTP server.

---

# UI Stack

## Rendering

- html/template
- embed.FS

---

## CSS

Framework:

- Pico CSS

Reasons:

- Mobile First
- lightweight
- semantic HTML
- responsive
- good browser compatibility

Project styles:

```
web/static/css/orders.css
```

Pico CSS is never modified directly.

---

## Icons

Library:

- Lucide

Format:

- SVG

Icons are stored locally in `static/icons/`.

### Standard set

| Действие  | Lucide   | Template      | Использование |
| --------- | -------- | ------------- | ------------- |
| Главная   | house    | icon_house    | Header        |
| Создать   | plus     | icon_plus     | Toolbar       |
| Найти     | search   | icon_search   | Lookup        |
| Сохранить | save     | icon_save     | Toolbar, формы |
| Отправить | send     | icon_send     | Формы         |
| Изменить  | file-pen | icon_file_pen | Таблицы       |
| Удалить   | trash-2  | icon_trash_2  | Таблицы       |
| Выход     | log-out  | icon_log_out  | Header        |

### Правила использования кнопок с иконками

| Паттерн                | Использовать                      | Применение                                                                  |
| ---------------------- | --------------------------------- | --------------------------------------------------------------------------- |
| `icon-button`          | Компактная кнопка (только иконка) | Таблицы, lookup, панели с ограниченным пространством                        |
| `icon-button--labeled` | Кнопка с иконкой и текстом        | Header, Toolbar, панели действий и другие места, где есть место для подписи |

Правило: `aria-label` добавляется только тогда, когда у кнопки нет видимого текста.
Если есть видимый текст (как в `icon-button--labeled`), `aria-label` избыточен.

New icons are added only when a new action appears.

### Component

All icon-only buttons use the `.icon-button` class (2.75rem square, CSS variable `--icon-button-size`). Buttons with text use `.icon-button--labeled`. Icons use the CSS variable `--icon-size` (default 1rem). Disabled state is handled via `:disabled` or `[aria-disabled="true"]` with reduced opacity.

### Inline SVG

Icons are rendered as inline SVG via Go template named templates (`templates/icons.html`). This gives native `currentColor` support — icons automatically inherit the text color, which makes theme switching (light/dark), `:hover`, `:focus`, and `:disabled` states work without additional CSS rules or duplicate icon files. The same mechanism also eliminates extra HTTP requests per icon.

`static/icons/` contains the original Lucide SVG files downloaded from the official repository. These files are not used at runtime — they serve as the source of truth for updating or regenerating `templates/icons.html`.

---

## JavaScript

Native JavaScript only.

Third-party JavaScript frameworks are not used.

JavaScript is optional and provides only progressive enhancement.

The application remains fully functional without JavaScript.

---

# PWA Ready

The application architecture is prepared for Progressive Web App support.

Current version does not require:

- Service Worker
- Offline mode
- Push notifications

Future versions may add these features without changing the UI architecture.

Reserved files:

```
web/static/manifest.json
web/static/sw.js
```

---

# Project Structure

```
internal/

    app/

        templates/

            layout.html

            login.html

            orders.html

            customers.html

            products.html

            users.html

            components/

                header.html
                sidebar.html
                toolbar.html
                table.html
                dialog.html
                pagination.html

web/

    static/

        css/

            pico.min.css
            orders.css

        js/

        icons/

        images/

        manifest.json

        sw.js
```

---

# Rendering

Every page is rendered through a single function.

```go
func (a *App) Render(
    w http.ResponseWriter,
    page string,
    data any,
)
```

Render is responsible for:

- loading templates;
- executing the layout;
- rendering the requested page.

Handlers never execute templates directly.

---

# Layout

All pages use a common layout.

Layout contains:

- HTML document
- Head
- Navigation
- Main content
- Footer

Only the page content changes.

---

# Components

The interface is built from reusable components.

Core components:

- Button
- Toolbar
- Input
- Select
- Checkbox
- Table
- Dialog
- Pagination
- Menu
- Alert

Components contain no business logic.

Components receive rendering data only.

---

# Forms

General rules:

- labels above controls;
- full width on mobile devices;
- consistent spacing;
- identical buttons.

# Entity Selection (Lookup)

Large directories (customers, products) are selected via a `[Выбрать]` button,
not a `<select>` element. The pattern:

1. Form shows current value (or "Не выбран") with a `[Выбрать]` link.
2. `[Выбрать]` navigates to the entity list page in picker mode
   (`?mode=picker&organization_id={oid}`).
3. In picker mode, rows link back to the origin form with
   `?customer_id={id}&return_to=...`.
4. The origin handler reads the selected ID from query params and
   re-renders the form with the entity pre-filled.

Picker mode rules:

- `[Выбрать]` is disabled if no organization is selected.
- The picker list only shows entities of the selected organization.
- This mechanism is used for customers, products, and any future
  large directories.

---

# Tables

Tables are one of the primary UI elements.

Supported features:

- sorting
- filtering
- paging
- responsive layout

All tables use the same appearance.

---

# Navigation

Desktop:

Left:

- Orders
- Customers
- Products
- Users

Right:

- Current user
- Logout

Mobile:

Navigation collapses into a menu button.

---

# Themes

Supported themes:

- Default

Future:

- Dark

Themes affect only CSS.

Templates remain unchanged.

---

# Browser Support

Supported browsers:

- Chrome
- Edge
- Firefox
- Safari

Responsive targets:

- Desktop
- Tablet
- Mobile

The UI follows the Mobile First approach.

---

# Static Assets

All static assets are bundled with the application.

External CDN resources are not used.

Assets include:

- CSS
- JavaScript
- Fonts
- Icons
- Images

The application is fully self-contained.

---

# Design Principles

The UI follows these principles:

- Server Side Rendering
- Mobile First
- Progressive Enhancement
- PWA Ready
- Semantic HTML
- Reusable Components
- Minimal JavaScript
- Local Static Assets
- Consistent Layout

# UI Philosophy

The server is the single source of truth.
HTML is generated on the server.
The browser is responsible only for presentation and user interaction.
Business logic never executes in the browser.

# Client-Side Architecture

## Principles

1. **HTML генерируется сервером** (`html/template`). Браузер получает готовую
   разметку без клиентского рендеринга.

2. **Alpine.js** отвечает только за локальное состояние интерфейса:
   - активация/деактивация элементов (`:disabled`)
   - модальные окна (отображение HTML, полученного от сервера)
   - реактивные вычисления (сумма документа)
   - Всегда progressive enhancement — форма работает и без Alpine
     (с полной перезагрузкой страницы)

3. **htmx** отвечает только за обмен с сервером:
   - сохранение документа (`hx-post`)
   - загрузка пикера (`hx-get`)
   - частичная подмена HTML
   - Всегда progressive enhancement — htmx-эндпоинты возвращают полную
     HTML-разметку, пригодную для обычного POST

4. **Вся бизнес-логика и окончательная валидация — только на сервере.**
   Браузер не принимает бизнес-решений (canSave, валидация форматов и т.п.).

5. **HTTP 422 Unprocessable Entity** используется для ошибок валидации.
   При 422 сервер возвращает форму с заполненными полями и ошибками.
   htmx заменяет форму; без htmx — полная перезагрузка с той же формой.

## Распределение ответственности

| Компонент | Что делает |
|-----------|------------|
| Сервер (Go) | Валидация, сохранение, рендер HTML |
| htmx | POST/GET к серверу, частичная подмена HTML |
| Alpine.js | `:disabled`, модалки, реактивные вычисления |

## Editor Context (Alpine)

Единое состояние редактора для всех типов объектов:

```js
{
  values: {},   // { fieldName: value }
  errors: {},   // { fieldName: "error text" }
  dirty: false
}
```

- Alpine управляет `:disabled` через `values.organization_id > 0`
  (UI-logic, не бизнес-логика)
- Кнопка «Сохранить» всегда активна; сервер отвечает 422 с ошибками
- Без Alpine все поля disabled до первой полной перезагрузки

## Lookup Field

Ссылочные поля (организация, контрагент, товар) используют кнопку
`[Выбрать]` и отображение текущего значения, а не `<select>`.

Picker mode — list page в режиме выбора:
- скрыта кнопка «Добавить»
- каждая строка ссылается на возврат с выбранным значением


# List Pages

All list pages use the generic `pages.ListPage` struct and the `list.html` template.

```go
page := pages.ListPage{
    Title:     "...",
    Columns:   pageColumns,
    Rows:      rows,
    NewURL:    ".../new",
    RowAction: pages.RowAction{...},
    EmptyText: "...",
}

a.Render(w, "entity", page)
```

Creating a dedicated list page struct (e.g. `ReceiptListPage`) is discouraged.
The standard `ListPage` already provides all fields that `list.html` expects:
`Title`, `Search`, `Columns`, `Rows`, `NewURL`, `RowAction`, `SearchURL`, `EmptyText`.

A custom list struct may only be introduced if `list.html` cannot be reused
and a completely different template is required.

Current list pages using `pages.ListPage`:

- Customers
- Organizations
- Products
- Receipts
- Users

---

---

# Component Library (Component Catalog)

## Architecture

The catalog page at `/ui` demonstrates the platform component library.
One page with sections; every new component appears here before real use:

```
Layout          →  layout/base.html
Components      →  components/*.html
Pages           →  pages/<page>/page.html
Theme           →  themes/<name>/theme.css
```

### Layout

`layout/base.html` — каркас страницы (HTML document, `<head>`, `<body>`).

Допустимо использовать классы темы для инфраструктуры (`container`, `section`).

### Components

| Компонент   | Файл                  | API                             | Внутренняя реализация |
|-------------|-----------------------|---------------------------------|-----------------------|
| Header      | `components/header.html` | `HeaderData{Section, Username, Menu []MenuItem}` | Semantic (`app-header`) |
| AppMenu     | `components/app_menu.html` | `MenuItem{ID, Label, Icon, URL, Danger, Separator}` | Semantic (`app-menu`) |
| Toolbar     | `components/toolbar.html` | `ToolbarData{Buttons}` / `SearchData{Placeholder, Value}` | Bulma level + button |
| Card        | `components/card.html`   | `card_open` / `card_close` (title) | Bulma card        |
| List        | `components/list.html`   | `ListData{Columns, Rows, RenderMode, Preset}` | Semantic Classes |
| Form        | `components/form.html`   | `Field{Name, Label, Type, Value}` | Bulma field/control/input |
| Dialog      | `components/dialog.html` | `DialogData{ID, Title}`         | Bulma modal           |
| FAB         | `components/fab.html`    | `FAB{Icon, Text, URL}`          | Semantic (`app-fab`) |

### Header и AppMenu

Правило платформы: **Header показывает раздел приложения, страница показывает объект.**

```
🏠  Организации                        ⋮
```

- `app-header-home` — иконка «домой», ведёт на `/` (dashboard).
- `app-header-title` — название раздела, усекается через `ellipsis`.
- `app_menu` — собственный компонент (не Bulma dropdown, не `<details>`).
  Без JS панель видна (пользователь + пункты). С JS (`html.js`) — toggle
  через `data-app-menu-toggle`, закрытие по клику вне / Esc / выбору пункта.
- Заголовок объекта (например, «ООО Ромашка») — ответственность страницы (`h1`),
  а не Header.

### Pages

Страницы собираются только из компонентов. Не используют CSS-классы темы напрямую.
Страницы могут содержать локальные элементы навигации (Breadcrumb), не выделяемые в компоненты.

Каждая страница — каталог с одним файлом `page.html` (`pages/<page>/page.html`),
определяющим `define "page_content"`. Шаблон живёт рядом с кодом, который его рендерит.

Страница каталога (`/ui`) — исключение: это стенд компонентов, он может
использовать собственные `catalog-*` классы для демонстрационной раскладки.

### Theme

`themes/bulma/theme.css` — реализация внешнего вида компонентов и страниц.

Использует CSS-переменные Bulma (`var(--bulma-*)`) для консистентности.

## Principles

### 1. Компонент платформы

Компонент платформы — это законченный функциональный блок интерфейса со своей моделью данных. Внутри компонента допускается использовать возможности выбранной темы (Bulma), но страницы работают только с компонентами платформы, а не с CSS-фреймворком.

### 2. Единая модель представления

Компонент должен иметь одну доминирующую модель представления. Дополнительные семантические классы допускаются, если они не подменяют и не дублируют структуру темы. Не смешивать Bulma и собственную семантику внутри одного компонента (либо Bulma, либо semantic — но не оба одновременно в одном элементе).

### 3. Компонент не навязывает содержимое

Компонент отвечает только за собственную область ответственности. Содержимое компонента остаётся ответственностью страницы или другого компонента.

Пример: Card отвечает за обёртку (`.card > .card-header + .card-content`), но не знает, что внутри: число, график, список или форма.

### 4. Один файл — один компонент

Каждый компонент размещается в одном файле шаблона. Если компонент состоит из нескольких частей, они оформляются как несколько `define` внутри этого файла.

### 5. Компоненты не закрытый список

Новые компоненты добавляются при появлении самостоятельной функциональной ответственности.

### 6. Компонент сначала нужен приложению

Компонент проектируется только когда он понадобился реальному приложению:

```
Реальная задача
  ↓
не хватает компонента
  ↓
Component Catalog
  ↓
реальное использование
```

Не проектировать компоненты «на будущее» («а вдруг потом понадобится...»).

### 7. Два независимых использования

Компонент считается частью общей библиотеки только после минимум двух
независимых использований в приложении (например, `Card` — организации
и dashboard). Если компонент нужен одному экрану — возможно, ему ещё
не место в общей библиотеке.

## Template Loading

Layout и компоненты загружаются через маску (не требуют изменения кода при добавлении):

```go
ui.RenderPage(w, baseFS, pageFS, data)
```

- `baseFS` — общие шаблоны: `layout/*.html` + `components/*.html`.
- `pageFS` — каталог конкретной страницы, содержит один файл `page.html`,
  который определяет `define "page_content"`.
- Шаблон страницы живёт рядом со своим вызывающим кодом (домен или раздел
  приложения), а не в общем наборе шаблонов.
- Добавление нового компонента не требует изменения кода — достаточно положить
  `.html` файл в `components/`.

### Реальные страницы

Шаблоны реальных страниц располагаются у своего владельца:

```
app:              internal/app/templates/pages/dashboard/page.html
домен модуля:     internal/organizations/templates/list/page.html
                  internal/organizations/templates/card/page.html
```

Домен отдаёт шаблоны через `organizations.Templates()` (embedded в release,
диск в debug — те же правила, что и у `app.TemplateFS()`). Хендлер в `app`
собирает данные из `pages`-моделей и вызывает:

```go
pageFS, _ := fs.Sub(organizations.Templates(), "list")
ui.RenderPage(w, TemplateFS(), pageFS, data)
```

Данные для рендера собирает слой `app` (хендлеры); домен не знает о `pages`.
FAB реализуется данными списка через `FABProvider` (например,
`organizationsListData.FAB()`).

### Dashboard

`/` — dashboard: `internal/app/dashboard.go` + `pages/dashboard/page.html`.
Счётчики (`Count`, `CountActive`), последние организации (`List` с
`OrderBy: "created_at"` и `Limit`), placeholder-блоки будущих модулей.

### Header в реальных страницах

Пункт меню с `ID == "logout"` рендерится как `<form method="post">` —
logout принимает только POST. Пункт меню приложения задаётся в `pageHeader()`
в `internal/app/organizations.go`.

### FAB

| Компонент | Файл | API | Внутренняя реализация |
|-----------|------|-----|----------------------|
| FAB | `components/fab.html` | `FAB{Icon, Text, URL}` | Semantic (`app-fab`) |

Правила:
- FAB — платформенный компонент (не Button).
- Только для главного действия страницы списка (Create).
- Отображается только на mobile (<769px).
- На desktop действие остаётся в Toolbar.
- Не использует классы Bulma (`button`, `is-*`).
- Использует собственные классы (`app-fab`) и токены темы (`--app-fab-*`).

---

## Platform Components

Header
AppMenu
Toolbar
FAB
Card
List
Form
Dialog

## Bulma Components Used

- button
- input
- field
- table
- icon

---

### Rules

Platform components must not depend on Bulma layout components (`navbar`, `level`, `hero`, etc.).

Допускается использование Bulma как:
- дизайн-токенов (`--bulma-*`);
- базовых элементов (`button`, `input`, `table`, `field`).

Компоновка платформенных компонентов реализуется собственными классами.

---

# Login Page

The login page is the application entry point.

It is available without authentication.

Layout:

- Application title
- Optional subtitle
- Login field
- Password field
- Login button
- Error message area

The page uses the common layout template.

The login form is centered vertically on desktop.

On mobile devices it occupies the full available width.

The login button always has full width.

After successful authentication:

/login
    ↓
302 Redirect
    ↓
/orders

After failed authentication:

- login page is rendered again;
- entered login is preserved;
- password field is cleared;
- an error message is displayed.

The page is fully functional without JavaScript.

# Authentication Flow

GET  /login
    ↓
Render login page

POST /login
    ↓
Authenticate user

Success
    ↓
302 → /orders

Failure
    ↓
Render login page
Display authentication error