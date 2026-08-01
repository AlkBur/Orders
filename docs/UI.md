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

# Модель представления

Главное правило UI-архитектуры проекта: данные движутся строго по
вертикали, и каждый слой отвечает только за свою часть.

```
Storage
   ↓
Domain
   ↓
Handler
   ↓
UI Model
   ↓
Template
   ↓
HTML
```

- **Storage** — доступ к данным (SQLite, prepared statements).
- **Domain** — бизнес-модель и бизнес-правила (`internal/customers`,
  `internal/receipts`, ...). Ничего не знает о `ui`.
- **Handler** — `app`-хендлер: валидация, вызов бизнес-логики, сборка
  модели представления.
- **UI Model** — данные для рендера (`ui.ListData`, `[]ui.Field`,
  структуры страниц вида `customersListData`). Только презентационные
  данные, никакой бизнес-логики.
- **Template** — представление (`html/template`); шаблон живёт рядом
  с кодом, который его рендерит.
- **HTML** — результат, отданный браузеру.

Правило: **доменные модели никогда не передаются в шаблоны.** Хендлер
всегда собирает модель представления из доменных данных. Шаблоны и
библиотека `internal/ui` не знают о предметной области: в
`components/*.html` и в `internal/ui` нет доменных имён (проверялось:
`internal/ui` не содержит констант полей домена, пресеты нейтральны
(`ListWide`/`ListCompact`), `form.html` знает только `Field`).

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

Список страницы строится из `ui.ListData` и рендерится доменным шаблоном
модуля (`templates/list/page.html`). Хендлер модуля собирает модель
представления (см. «Модель представления»); домен в шаблон не передаётся.

```go
data := customersListData{
    List: ui.ListData{
        Columns:    columns,              // из Descriptor.ListFields() + DisplayValue
        Rows:       rows,                 // []ui.ListRow
        RenderMode: ui.RenderComfortable,
    },
}
pageFS, _ := fs.Sub(customers.Templates(), "list")
ui.RenderPage(w, TemplateFS(), pageFS, data)
```

Строка `ui.ListRow{URL, Cells, Actions []RowAction}`:

- `URL` — задан → строка является ссылкой на объект;
- `Actions` — заданы → вместо перехода рисуются действия строки
  (например, «Выбрать» в пикере).

Действия строки не заменяют навигацию по списку: если строке нужна и
ссылка, и действие, это противоречит модели списка (вложенные ссылки
недопустимы) — решается либо ссылкой, либо действием, но не тем и другим.

Легаси `pages.ListPage` больше не используется для новых страниц.
Пока остаётся в:

- Products
- Receipts
- Users

(переносятся по мере миграции; Organizations и Customers уже переведены
на `ui.ListData`).

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
| List        | `components/list.html`   | `ListData{Columns, Rows, RenderMode, Preset}`; строка `ListRow{URL, Cells, Actions []RowAction}` | Semantic Classes |
| Form        | `components/form.html`   | `Field{Name, Label, Type, Value, Readonly, Required, Autofocus, Autocomplete, Icon, Placeholder, Options []SelectOption}` | Bulma field/control/input |
| Alert       | `components/alert.html`  | `AlertData{Type, Messages}`   | Semantic (`alert`)    |
| Dialog      | `components/dialog.html` | `DialogData{ID, Title}`         | Bulma modal           |
| FAB         | `components/fab.html`    | `FAB{Icon, Text, URL}`          | Semantic (`app-fab`) |

#### Row Actions

Действия строки списка (`RowAction{ID, Icon, Label, URL}`) — ссылки-иконки
на строке. Библиотека не знает бизнес-смысла действия: иконку, подпись и
URL задаёт хендлер. Типичный пример — пикер: вместо перехода по `URL`
строки рисуется действие «Выбрать» (`icon: check`), которое возвращает
в исходную форму с выбранным значением.

#### Select Options

Поле выбора (`FieldSelect`) описывается через `Field.Options []SelectOption`.
`SelectOption{Value, Label, Disabled}` описывает только реальные данные.
Placeholder для пустого значения — свойство `Field.Placeholder`, а не
искусственная опция.

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
ui.Render(w, baseFS, pageFS, name, data)
```

- `baseFS` — общие шаблоны: `layout/*.html` + `components/*.html`.
- `pageFS` — каталог конкретной страницы, содержит один файл `page.html`,
  который определяет `define "page_content"`.
- `name` — любой именованный шаблон: `base`, `auth`, `page_content`, `dialog`, ...
- Шаблон страницы живёт рядом со своим вызывающим кодом (домен или раздел
  приложения), а не в общем наборе шаблонов.
- Добавление нового компонента не требует изменения кода — достаточно положить
  `.html` файл в `components/`.

`ui.Render` умеет рендерить любой именованный шаблон. `RenderPage` и
`RenderAuthPage` — тонкие обёртки, выбирающие layout. HTMX лишь использует
фрагментную возможность рендера, но не определяет её: `page_content` — обычный
именованный шаблон, а не специальная возможность для HTMX.

### Layouts

| Layout       | Шаблон                | Назначение                                          |
|--------------|-----------------------|-----------------------------------------------------|
| Application  | `layout/base.html`    | После входа: Header, FAB, AppMenu, Toolbar, страница |
| Auth         | `layout/auth.html`    | Вход в систему: Login, Set Password                  |

Auth Layout — минимальный каркас: только центрированный контейнер страницы
(`auth-layout` → `auth-container` → `page_content`). Ни Header, ни FAB, ни
AppMenu, ни Toolbar, ни footer, ни логотипов.

Правило: **Layout отвечает только за каркас страницы. Он не содержит
бизнес-логики и не знает о конкретных экранах** (Login, Dashboard,
Organizations).

### ResponseMode

`ResponseMode` определяет формат ответа сервера: полная страница или
фрагмент. Способ определения режима (HTMX или иной транспорт) является
инфраструктурной деталью.

```go
type ResponseMode int // FullPage, Fragment
mode := ResponseModeFromRequest(r)
```

- `FullPage` — обычный ответ: полный Layout; редирект `303 See Other`.
- `Fragment` — только `page_content`; редирект через заголовок `HX-Redirect`.

Хендлер определяет режим один раз в начале (`mode := ResponseModeFromRequest(r)`)
и дальше работает через `a.RenderAuth(...)` / `a.Redirect(...)`, не проверяя
заголовки HTMX напрямую. Бизнес-логика не зависит от транспорта доставки.

### Реальные страницы

Шаблоны реальных страниц располагаются у своего владельца:

```
app:              internal/app/templates/pages/dashboard/page.html
                  internal/app/templates/pages/login/page.html
                  internal/app/templates/pages/set_password/page.html
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

`/` — «Рабочий стол» администратора: `internal/app/dashboard.go` +
`pages/dashboard/page.html`.

Dashboard — это **лаунчер приложения**: только навигационные карточки
модулей. Карточка содержит иконку раздела, название и количество объектов
(`dashboardModule{Name, Icon, URL, Count string, Hero, Note}`). Количество
подготавливается хендлером через `display.FormatNumber` — UI-модель уже
содержит готовое представление, шаблон ничего не форматирует.

Правила рабочего стола:

> Рабочий стол содержит только навигационные карточки модулей. Любая
> информация о состоянии модуля (например, количество объектов)
> отображается внутри карточки соответствующего модуля и не выносится
> в отдельные информационные панели.

> Рабочий стол не отображает содержимое модулей. Он показывает только
> точки входа в них.

> Каждая карточка является самостоятельной точкой входа в модуль. Вся
> площадь карточки является кликабельной.

> Рабочий стол не содержит бизнес-логики. Любая информация, отображаемая
> на карточке, должна помогать выбрать модуль, но не заменять переход в него.

> Иконка отражает раздел системы, а не конкретный тип объекта (раздел
> «Документы» — иконка документа, а не чека).

Toolbar и FAB отсутствуют намеренно: рабочий стол ничего не создаёт,
он только направляет. Возвращать кнопки на dashboard не нужно.

`Count(ctx) (int64, error)` — единая сигнатура счётчиков во всех хранилищах
(`customers`, `products`, `receipts`, `users`, `organizations`).

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

Страница входа — обычная страница платформы на Auth Layout: без Header,
FAB, AppMenu и Toolbar. Собирается только из общих компонентов:

```
Auth Layout
    └── Card («Вход»)
            ├── Alert (ошибки)
            └── Form
                    ├── Пользователь (text)
                    ├── Пароль (password)
                    └── Button «Войти»
```

Карточка центрируется темой (`auth-layout` + `auth-card`), ширина —
`min(420px, calc(100vw - 2rem))`.

Валидация выполняется только на сервере (HTML `required` не используется):

- пустые поля → «Пользователь обязателен.» / «Пароль обязателен.»;
- неверные данные → «Неверный логин или пароль.»;
- все найденные ошибки собираются сразу и показываются через Alert.

После успешной аутентификации пользователь попадает на стартовый экран,
который определяется `LandingURL(identity)` (см. «Landing»): установка
пароля, рабочий стол для админа, документы для пользователя.

Форма использует htmx для плавной доставки: при ошибке сервер возвращает
только карточку (`page_content`), которую htmx подменяет через
`hx-target="#login-card"`. Без JavaScript форма работает как обычный POST
(полная страница при ошибке, `303` при успехе).

# Set Password

Страница `set-password` — часть потока входа (Auth Layout, без Header).
Обязательный шаг после первого входа по bootstrap-паролю: пока пароль не
установлен, доступ к приложению закрыт (`RequirePassword`).

Серверная валидация собирает все ошибки сразу:

1. «Новый пароль обязателен.»
2. «Подтверждение пароля обязательно.»
3. «Пароли не совпадают.»

После успешной установки сессия завершается и пользователь перенаправляется
на `/` для входа с новым паролем.

# Landing

Стартовый экран после входа определяется ролью и состоянием аккаунта.
Единственная точка принятия этого решения — `LandingURL(u users.Identity)`
в `internal/app/landing.go`:

```go
func LandingURL(u users.Identity) string {
    if u.NeedsPasswordSetup() {
        return RouteSetPassword
    }
    if u.IsAdmin {
        return RouteDashboard
    }
    return RouteReceipts
}
```

Приоритет:

1. установка пароля — всегда, независимо от роли (`/set-password`);
2. админ — рабочий стол (`/`);
3. пользователь — документы (`/receipts`).

`LandingURL` намеренно «тупая»: она только выбирает первый экран и не
содержит ни проверки прав, ни навигационной логики. Маршруты задаются
константами в `internal/app/routes.go` (`RouteDashboard`, `RouteReceipts`,
`RouteSetPassword`).

Рабочий стол (`/`) доступен только администратору — `RequireAdmin`
в роутере (403 для остальных); обычный пользователь после входа
попадает на `/receipts`.

# Authentication Flow

```
GET /login
    ↓
RenderAuth (FullPage)

POST /login
    ↓
Серверная валидация
    ↓
Ошибки → RenderAuth (фрагмент для htmx / полная страница) + Alert
Успех  → Redirect (LandingURL: /set-password | / | /receipts)

POST /set-password
    ↓
Серверная валидация (все ошибки сразу)
    ↓
Ошибки → RenderAuth + Alert
Успех  → завершение сессии, Redirect (/)
```

# UI Polish Process

Визуальный полиш библиотеки выполняется **пакетами**. После завершения
пакета команда возвращается к разработке функциональности. Новый полиш
начинается только после появления новых реальных сценариев, требующих
изменений компонентов.

Полиш улучшает фундаментальные компоненты платформы, а не отдельные экраны
(Login не является причиной изменения компонентов — только первым местом,
где новое API используется). Новые косметические изменения не принимаются
до появления нескольких новых реальных сценариев, выявляющих необходимость
доработки компонентов.