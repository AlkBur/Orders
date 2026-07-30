# RFC-017: Object Editor

**Object Editor** — платформенный механизм построения карточек объектов.
Он не зависит от конкретной сущности и используется как для документов,
так и для справочников. Конкретная карточка определяется только своим
`ObjectDescriptor`.

## Status

Draft — after Commit A (Receipt infrastructure). First implementation target:
Receipt document.

## Motivation

До Commit A каждая карточка (Product, Customer, Organization) имела
уникальный шаблон и свою (или отсутствующую) логику валидации. Документы
(Receipt) добавили требования, которых нет у справочников:

- поля зависят друг от друга (организация → контрагент → товары);
- валидация агрегата (наличие организации, контрагента и строк);
- read-only после отправки;
- ошибки привязываются к полям, форма не сбрасывается.

Без общего механизма каждая новая карточка будет изобретать свой велосипед.

## Solution

Четыре слоя:

```
Object Descriptor
        ↓
Editor Engine
        ↓
Editor Context (Alpine)
        ↓
HTML + htmx
```

### 0. Разграничение: Entity Descriptor vs Object Descriptor

В проекте уже существует `entity.Descriptor`, который описывает доменную
модель: таблицу БД, ключи, отображаемые поля в списках, метаданные
сущности. Он используется Store, Schema, Integration API.

**Object Descriptor — другой уровень.** Он описывает не модель данных,
а её представление в конкретной форме редактирования: какие поля,
какого типа, как зависят друг от друга, какие действия доступны.

```
Customer (доменная сущность)
        │
        ▼
Entity Descriptor (модель, БД, ключи, инфраструктура)
        │
        ▼
Object Descriptor (карточка: поля, типы, зависимости, действия)
        │
        ▼
Object Editor (рендер, валидация, контекст)
```

Это разные обязанности. Object Editor не заменяет Entity Descriptor
и не дублирует его. Они сосуществуют на разных слоях абстракции.

### 1. Object Descriptor

Метаданные формы редактирования. Единственное место, где описывается
состав карточки.

```go
type ObjectDescriptor struct {
    Name    string
    Fields  []FieldDescriptor
    Actions []ActionDescriptor
}

type FieldDescriptor struct {
    Name      string
    Type      FieldType  // Lookup, Text, Number, Date, Boolean
    Label     string
    Required  bool
    ReadOnly  bool
    Visible   bool
    DependsOn []string   // имена полей, от которых зависит
    Lookup    *LookupConfig  // только для Lookup-полей
}

type LookupConfig struct {
    Entity      string // "customer", "product", "organization"
    FilterField string // "organization_id"
    Display     string // "name" — поле сущности для отображения
}

type ActionDescriptor struct {
    Name  string
    Label string
}

type FieldType int
const (
    FieldLookup FieldType = iota
    FieldText
    FieldNumber
    FieldDate
    FieldBoolean
)
```

Descriptor не знает URL пикеров и не содержит бизнес-логики.
Он только описывает структуру формы.

Поле `Display` в `LookupConfig` указывает, какое поле сущности
показывать пользователю. По умолчанию — `"name"`. Это избавляет
от специальных случаев (например, логин пользователя вместо имени,
или `"Наименование (ед. изм.)"` для товара).

### 2. Editor Engine

Связующий слой. Единственное место, которое знает, как рендерить форму
по Descriptor:

- Итерирует `Descriptor.Fields`
- Для каждого поля выбирает FieldComponent по `Type`
- Интерпретирует метаданные поля:
  - `DependsOn` — уведомляет поле о зависимостях; конкретное поведение
    (очистка значения, блокировка, смена фильтра Lookup) определяется
    типом поля и его метаданными, **не Engine**
  - `ReadOnly` — блокирует ввод
- Отображает `errors` рядом с полем

Editor Engine не знает конкретных сущностей и не содержит бизнес-правил.
Он интерпретирует метаданные Descriptor.

### 3. Editor Context (Alpine)

Минимальное состояние редактора в браузере. Общее для всех объектов.

```js
{
  values: {},   // { fieldName: value }
  errors: {},   // { fieldName: "error text" }
  dirty: false
}
```

- Alpine управляет только UI-состоянием (`:disabled` при отсутствии
  организации). Это progressive enhancement, не бизнес-логика.
- Бизнес-валидация — только на сервере.
- Кнопка «Сохранить» всегда активна. Сервер отвечает 422 с формой и
  ошибками.
- Без Alpine форма работает через полную перезагрузку.

### 4. HTTP 422 для ошибок валидации

```
POST /resource
    ↓
Валидация
    ↓
Ошибки? → 422 + рендер формы с errors
    ↓
Успех   → 302 Redirect
```

- htmx заменяет форму при 422 — данные не теряются.
- Без htmx (progressive enhancement) — обычный POST, сервер рендерит
  форму с ошибками и статусом 422.

### 5. Lookup Field (Reference Field)

Ссылочные поля (организация, контрагент, товар) используют единый
UI-элемент — кнопку `[Выбрать]` и отображение текущего значения:

```html
<div class="lookup-field">
  <input type="text" :value="values.customer_name" readonly disabled>
  <a :href="pickerUrl" role="button" class="outline"
     :class="{ disabled: !values.organization_id }">Выбрать</a>
</div>
```

Picker URL строится на основе `LookupConfig`:

```
/{entity}?mode=picker&org_id={orgId}&field={fieldName}&return_to={currentUrl}
```

Picker mode — режим list page, где:
- скрыта кнопка «Добавить»
- каждая строка содержит ссылку возврата

## Что не входит в RFC-017

| Механизм | Причина |
|----------|---------|
| Editor Session | Когда full-url picker перестанет справляться — отдельный RFC |
| Библиотека Field Components | Появится при втором пользователе Editor |
| `canSave` в Alpine | Только на сервере |
| `Table` (TableField) | Не включается до появления первой реализации, требующей табличного поля |
| `tables`/`lookups` в Editor Context | Добавлять по мере необходимости |

## Первый пользователь

Receipt (товарный чек). Commit B реализует:

- Alpine + htmx в layout
- Editor Context (Alpine)
- LookupField для организации, контрагента, товара
- `POST /receipts` с валидацией + 422
- `Save(*Document)` в одной транзакции
- Picker mode для customers и products
