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

| Action   | Icon     | Text |
| -------- | -------- | ---- |
| Lookup   | Search   | no   |
| Add      | Plus     | yes  |
| Delete   | Trash2   | no   |
| Save     | Save     | yes  |
| Send     | Send     | yes  |
| Logout   | LogOut   | no   |

New icons are added only when a new action appears.

### Component

All icon-only buttons use the `.icon-button` class (2.5rem square, CSS variable `--icon-button-size`). Buttons with text use `.icon-button--labeled`. Icons use the CSS variable `--icon-size` (default 1rem). Disabled state is handled via `:disabled` or `[aria-disabled="true"]` with reduced opacity.

**Current implementation:** standalone SVG files via `<img>`.
**Future option:** inline SVG to support `currentColor`.

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