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

Правило: **Шаблоны отображают готовые данные.** ViewModel должна содержать
все данные в готовом для отображения виде. Шаблоны не строят URL, не
форматируют значения, не вычисляют состояния и не содержат бизнес-логику.
Пример: строка списка чеков несёт готовые `ViewURL/EditURL/CopyURL/
SendURL/FilesURL` и флаги интерфейса `CanEdit/CanSend` — шаблон только
выводит их и не собирает пути из `ID`.

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

Правило: кнопки действий **без текстовой подписи** используют `.icon-button`.
Для таких кнопок не используется `button is-small` и другие компактные классы.

New icons are added only when a new action appears.

### Component

All icon-only buttons use the `.icon-button` class (2.75rem square, CSS variable `--icon-button-size`). Buttons with text use `.icon-button--labeled`. Icons use the CSS variable `--icon-size` (default 1rem). Disabled state is handled via `:disabled` or `[aria-disabled="true"]` with reduced opacity.

`.icon-button` отвечает только за область нажатия и состояние кнопки.
Цвет действия задаётся отдельными модификаторами (например, `.icon-button-danger`),
которые вводятся при появлении второго сценария использования.

`.icon-button__label` — опциональная подпись кнопки. Её показ/скрытие управляется
**местом использования**, а не базовым классом. Например, в табличной части документа
подпись скрыта на десктопе и показывается на мобильном (`.receipt-item-actions`).

Миграция на `.icon-button` выполняется постепенно по мере правки компонентов.

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

### Принцип карточки

Карточка объекта выполняет одно действие — редактирование объекта.

Навигация осуществляется глобальными элементами интерфейса (меню,
кнопка «Домой», хлебные крошки), а не кнопками внутри карточки.
Поэтому внутри карточки нет служебных кнопок — «К списку», «Удалить»
и т.п. Карточка максимально проста:

```
Основная информация
───────────────────

UUID
Наименование
...
Активна  [────●]

          💾 Сохранить
```

Правила карточки:

- единая структура: шапка «Основная информация» + поля формы + одна
  кнопка «Сохранить» (иконка дискеты);
- булевы поля — переключатель (switch) на чистом CSS: обычный
  `<input type="checkbox">`, стилизованный через CSS. Без JavaScript:
  клавиатура, screen reader и отправка формы работают как обычно;
- заголовок карточки — только имя сущности («Организация»,
  «Контрагент», «Товар», «Пользователь», «Документ»), без
  «Создание …» / «Редактирование …» — режим понятен по заполненности
  полей.

### Сохранение (PRG)

После успешной POST-операции пользователь всегда возвращается к
каноническому списку объекта, а результат операции отображается через
Flash-сообщение. Не «предыдущая страница», а канонический список:

- Организация → `/organizations`;
- Контрагент организации → `/organizations/{id}/customers`;
- Документ → `/receipts`.

Flash-сообщение устанавливается до редиректа (например,
«Организация сохранена.») и показывается на списке один раз, после
чего очищается. Транспортный тип Flash живёт в `internal/sessions`
(`sessions.FlashType`, `sessions.Flash`); преобразование в модель
представления — в `internal/app/flash.go` (`FlashToAlert`).

Это поведение — общий шаблон для всех карточек. При миграции Товаров,
Пользователей и Документов они должны вести себя так же: сохранить
объект → Flash → список.

# Entity Selection (Lookup)

Large directories (customers, products) are selected via a `[Выбрать]` button,
not a free-form input. In the receipt editor the button opens a modal picker;
the selected value is transferred into the editor and the picker closes.
The same list can also be opened in URL picker mode:

1. Form shows current value (or "Не выбран") with a `[Выбрать]` button.
2. The picker is always scoped to the selected organization.
3. Search remains available in picker mode.
4. The picker has no `[Добавить]` action.
5. A normal list row opens the entity card; a picker row has only the
   `Выбрать` action and returns the selected ID to the origin form.

Picker mode rules:

- `[Выбрать]` is disabled if no organization is selected.
- The picker list only shows entities of the selected organization.
- Changing the organization clears dependent customer/product values.
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
   - Alpine подключается из локального `/static/js/alpine.min.js`; без него
     серверные поля сохраняются, но modal-операции редактора недоступны.

3. **htmx** отвечает только за обмен с сервером:
   - сохранение документа (`hx-post`)
    - обмен с сервером и сохранение документа (`hx-get`/`hx-post`)
   - частичная подмена HTML
   - Всегда progressive enhancement — htmx-эндпоинты возвращают полную
     HTML-разметку, пригодную для обычного POST

4. **Вся бизнес-логика и окончательная валидация — только на сервере.**
   Браузер не принимает бизнес-решений (canSave, валидация форматов и т.п.).

5. **Ошибки валидации описываются `ValidationError` и доставляются в
   зависимости от режима ответа.**
   - htmx-режим: Fragment → транспортное представление ValidationError
     (сегодня — JSON `{title, errors, fields}`), клиент отображает его
     (модальное окно) через `htmx:afterRequest`.
   - без htmx: FullPage → `422` + форма с заполненными полями и ошибками.
   - Успех в обоих режимах — обычный редирект.

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
- Кнопка «Сохранить» всегда активна; ошибки приходят на сервер и
  доставляются как ValidationError (JSON в htmx-режиме, 422 + форма без htmx)
- Без Alpine все поля disabled до первой полной перезагрузки

## Lookup Field

Ссылочные поля (организация, контрагент, товар) используют кнопку
`[Выбрать]` и отображение текущего значения, а не `<select>`.

Picker mode — list page в режиме выбора:
- скрыта кнопка «Добавить»;
- обычная строка не открывает карточку;
- действие строки возвращает выбранное значение в исходную форму.

## Receipt Editor

Редактор документа показывает текущую дату и поле номера только для чтения.
Номер **не резервируется при открытии формы**: он назначается Store только
в транзакции успешного сохранения. Если пользователь отменил создание или
валидация завершилась ошибкой, номер не расходуется. Ниже находятся
организация и контрагент. Табличная часть имеет собственный toolbar с
добавлением и поиском по уже добавленным строкам. Добавление строки открывает
modal с товаром, количеством, ценой и вычисляемой суммой. В верхней части
редактора строки находятся только кнопки `ОК` и `Отмена`; плавающий modal-header
не используется. Сумму можно изменить вручную: при количестве больше нуля
цена пересчитывается как `сумма / количество`. Выбор товара фильтруется по
организации документа. Сервер
повторно проверяет принадлежность товара и контрагента организации перед
сохранением.

Удаление строки выполняется через confirm-модалку с номером строки и названием
товара. Удаление идёт по ссылке на строку, а не по индексу. Правило модалок:
`Escape` закрывает окно без подтверждения (без удаления).

Документ имеет собственный шаблон строк (в `card/page.html`); шаблон списка
справочников (`ListView`) для строк документа не используется. Изменения UX строк
документа выполняются только в документном шаблоне.

Табличная часть вынесена в **именованные шаблоны** (template blocks): `lines_editor`
и `lines_view`, подключаемые через `{{template "lines_editor" .}}` /
`{{template "lines_view" .}}`. Это позволяет менять редактор независимо от просмотра.
Внутри каждый режим разграничен контекстной обёрткой `receipt-items--editor` /
`receipt-items--view`.

Строка логически делится на три группы:

1. **описание товара** — `receipt-item-main`: №, название товара, ед. изм;
2. **числовые показатели** — количество, цена, сумма;
3. **действия** (`receipt-item-actions`, только в редакторе) — редактировать, удалить.

Шапка редактора имеет колонку `Действия`; её ширина считается через переменные
(`--icon-button-size` и `--bulma-column-gap`), чтобы шапка и строки имели одинаковое
распределение колонок и не смещались относительно друг друга. Описание товара на
десктопе растянуто по первым трём колонкам (повтор `grid-template-columns` для
первых трёх треков; без `display: contents` и без `subgrid`).

На мобильном редактор и просмотр перестраиваются в карточку из трёх строк:

```
1. Монитор LG UltraGear 24GS60F-B (шт.)   ← строка 1: описание товара, переносится
Кол-во | Цена | Сумма                    ← строка 2: три колонки чисел
[Редактировать] [Удалить]                ← строка 3: действия на всю ширину (editor)
```

Строка описания строится через `flex-wrap: wrap` (единица измерения — атомарный
элемент с `white-space: nowrap`, не разрывается). Точка после номера и скобки
единицы измерения добавляются **только на мобильном** через псевдоэлементы, чтобы
десктопная таблица оставалась чистой. Подписи на строке 1 скрыты; на строке 2
подпись «Количество» на мобильном заменяется на «Кол-во» средствами CSS (текст в
шаблоне остаётся полным). Кнопки действий имеють одинаковую ширину
(`flex: 1 1 0`). Мобильная раскладка переключается на существующем брейкпоинте
`max-width: 768px`.

> **Правило представлений.** Desktop и Mobile рассматриваются как два независимых
> представления одной модели данных. Совпадение структуры DOM не является целью;
> приоритет — удобство пользователя на каждом типе устройства. Мобильная версия не
> обязана повторять структуру десктопной и проектируется как самостоятельный
> интерфейс для малого экрана.

Любое информационное поле строки, не являющееся кнопкой действия (номер, товар,
ед. изм.), может открывать редактор строки.

## Необратимые действия

Правило: **необратимые действия выполняются POST-запросом после подтверждения;
GET никогда не меняет состояние.**

Пример — двухэтапная отправка документа в 1С (чек). Кнопка «Отправить в 1С»
в редакторе сохраняет документ и открывает экран подтверждения (`mode=send`),
не отправляя документ. На экране подтверждения:

- форма с кнопкой «Отправить» выполняет `POST /receipts/{id}/send`;
- подтверждение реализуется атрибутом `data-confirm` на этой форме,
  обработчиком в `static/js/receipts.js` через `window.confirm`;
- после успешной отправки `SentAt` устанавливается сервером, документ
  становится read-only, и HTTP-редирект возвращает к списку с Flash-сообщением;
- кнопка «Отмена» — обычная ссылка (`get`), ведущая на `ReturnURL`
  (источник: редактор или список).

Сервер дублирует защиту: повторная отправка уже отправленного документа
отклоняется (`400`), а `SentAt` устанавливается только в обработчике отправки
(`POST /send`), а не при сохранении.

## List Pages

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

- Все текущие списки используют `ui.ListData` и доменный шаблон списка.

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
| Toolbar     | `components/toolbar.html` | `ToolbarData{Buttons}` | Bulma buttons |
| Search      | `components/search.html` | `SearchData{URL, Query, Placeholder, Mode, MinLength}` | Bulma field/control/input |
| ListView    | `components/list.html` (`list_view`) | `ListView{Toolbar *ToolbarData, Search *SearchData, List ListData}` | Композиция |
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

#### Search

`components/search.html` — блок поиска списка. Модель:

```go
type SearchMode uint8

const (
    SearchManual SearchMode = iota
    SearchLive
)

type SearchData struct {
    URL         string // базовый путь списка (?q=...)
    Query       string // текущее значение запроса
    Placeholder string
    Mode        SearchMode // SearchLive — живой поиск по мере ввода
    MinLength   int        // минимальная длина автопоиска; 0 — дефолт (3)
}
```

Форма выполняет GET на `URL` с параметром `q`. При `Mode: SearchLive` поле ввода
получает `hx-trigger="search, keyup changed delay:500ms"` и атрибут
`data-auto-search`; `static/js/search.js` реализует правило автопоиска:

> Автоматический поиск выполняется при длине запроса ≥ `MinLength`
> (по умолчанию 3) или когда поле поиска очищено полностью (список
> автоматически возвращается к исходному состоянию). Промежуточные значения
> длиной 1..`MinLength`-1 — только после Enter или кнопки «Найти».
> Отменяются только запросы самого поля; отправка формы работает всегда.

`id="list"` на корне списка — целевой элемент `hx-target` поиска, поэтому
фрагмент поиска подменяет список целиком.

Правила платформы для поиска:

> **Поиск всегда выполняется только по отображаемым колонкам.** Handler
> определяет отображаемые поля списка (`entity.FieldName`), Store сопоставляет
> каждому отображаемому полю SQL-выражение (`MappedColumn.Expression`), а
> `VisibleColumns` оставляет только видимые. Модуль `internal/database/search`
> после `VisibleColumns` оперирует `SearchColumn{Expression}` и не знает о модели.

> **Поиск никогда не использует скрытые поля модели. Если колонка не
> отображается пользователю, она не участвует в поиске.**

> **Порядок поиска соответствует порядку отображения колонок списка.**

> Поисковый запрос разбирается `NormalizeQuery` на нормализованные слова
> (`normalizeWord`: приведение к нижнему регистру; `strings.Fields()`:
> схлопывание пробелов). `Words` никогда не содержат пустых строк. Для каждого
> слова должно существовать хотя бы одно совпадение в любой отображаемой
> колонке (AND между словами, OR между колонками). Порядок слов не важен.

> **Поиск регистронезависим и нормализован.** Обе стороны сравниваются через
> единый алгоритм `normalizeWord`: SQL-колонка — функцией `search_normalize`,
> слово запроса — той же функцией в Go. Любое изменение `normalizeWord()`
> автоматически изменяет SQL-поиск через `search_normalize()`.

> **Поиск является фильтром текущего списка, а не глобальным поиском системы.**

> **Поиск не использует индексы при поиске по подстроке (`LIKE '%...%'`)
> и предназначен для рабочих списков небольшого и среднего размера.** При
> необходимости полнотекстового поиска меняется только `BuildPredicate()`
> (одна точка) — Store, Handler и UI не затрагиваются.

#### ListView

`components/list.html` (`list_view`) — композиция «тулбар + поиск + список»:

```go
type ListView struct {
    Toolbar *ToolbarData // nil — блок не рендерится
    Search  *SearchData  // nil — блок не рендерится
    List    ListData
}
```

Раскладка через Bulma `.level`: на десктопе (≥ 769px) **тулбар слева, поиск
справа** в одной строке. На мобильном (≤ 768px) тулбар скрывается, когда есть
`FAB` (правило `.has-fab .toolbar-actions { display: none }`), — главное действие
доступно плавающей кнопкой, остаётся поле поиска.

Страница-список собирается из `list_view`, а не раскладывает toolbar/search/list
сама. Отдельные блоки остаются доступны для составных страниц.

#### Entity cards

Страницы создания и редактирования организаций, контрагентов, товаров,
пользователей и чеков используют общий внешний контейнер `card_open` /
`card_close` и Bulma-стиль формы. Простые формы выводят поля через
`form_field`; специализированный редактор чеков сохраняет собственную
Alpine-разметку внутри того же card-контейнера.

Legacy-шаблоны `page-card` и отдельные плоские card-шаблоны для товаров,
пользователей и чеков не используются.

### Header и AppMenu

Правило платформы: **Header показывает раздел приложения, страница показывает объект.**

```
🏠  Организации                        ⋮
```

- `app-header-home` — иконка «домой», ведёт на `/` — точку входа приложения, которая редиректит на стартовый экран по роли (см. «Application Entry»).
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

`/dashboard` — «Рабочий стол» администратора: `internal/app/dashboard.go` +
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
2. админ — рабочий стол (`/dashboard`);
3. пользователь — документы (`/receipts`).

`LandingURL` намеренно «тупая»: она только выбирает первый экран и не
содержит ни проверки прав, ни навигационной логики. Маршруты задаются
константами в `internal/app/routes.go` (`RouteHome`, `RouteDashboard`,
`RouteReceipts`, `RouteSetPassword`).

# Application Entry

`/` — точка входа в приложение и больше ничего:

```
GET /
    ↓
нет сессии           → /login
нет пользователя     → очистить сессию → /login
нужно сменить пароль → /set-password
администратор        → /dashboard
обычный пользователь → /receipts
```

`Home()` (`internal/app/home.go`) проверяет только сессию и всегда
делегирует выбор маршрута `LandingURL()`. Он не содержит собственной
логики выбора первого экрана — вся она живёт в одном месте, в `LandingURL()`.

`/dashboard` доступен только администратору — `RequireAdmin` в роутере
(403 для остальных); обычный пользователь после входа попадает
на `/receipts`. `/` не знает о правах — он только перенаправляет.

# Authentication Flow

```
GET /login
    ↓
RenderAuth (FullPage)

GET /
    ↓
Home(): валидация сессии → Redirect (LandingURL: /set-password | /dashboard | /receipts)

POST /login
    ↓
Серверная валидация
    ↓
Ошибки → RenderAuth (фрагмент для htmx / полная страница) + Alert
Успех  → Redirect (LandingURL: /set-password | /dashboard | /receipts)

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
