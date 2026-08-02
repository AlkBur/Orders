# Orders Server — Структура базы данных

Версия: 2.0

---

# 1. Общие положения

Система использует две независимые базы данных SQLite.

## base.db

Содержит оперативные данные приложения:

- пользователи;
- организации;
- контрагенты;
- товары;
- товарные чеки;
- строки товарных чеков.

## files.db

Содержит только двоичные данные документов (PDF).

*Реализация запланирована. Текущая версия приложения использует только `base.db`.*

Такое разделение позволяет:

- уменьшить размер рабочей базы данных;
- сохранить высокую производительность пользовательского интерфейса;
- выполнять резервное копирование файлов независимо от рабочих данных;
- при необходимости переносить архив документов отдельно.

---

# 2. Общие требования

- используется SQLite;
- используется режим WAL;
- включены внешние ключи (`PRAGMA foreign_keys = ON`);
- таблицы создаются автоматически при запуске приложения.

---

# 3. Dual-ID Architecture

Все сущности (кроме сессий и товарных чеков) имеют два идентификатора:

- **ID (INTEGER)** — внутренний первичный ключ. Автоинкремент. Используется
  в URL веб-интерфейса, внешних ключах и JOIN. Тип `INTEGER` с `PRIMARY KEY AUTOINCREMENT`.

- **UUID (TEXT)** — внешний идентификатор. Используется в API и синхронизации
  с внешними системами. Тип `TEXT NOT NULL UNIQUE` (Organization, User)
  или `TEXT NOT NULL` с составным `UNIQUE(organization_id, uuid)` (Customer, Product).

Сессии — исключение: используют строковый первичный ключ (токен сессии).

Товарные чеки (Receipts) — исключение: используют три идентификатора:

- **ID (INTEGER)** — внутренний первичный ключ.
- **ExchangeID (TEXT UNIQUE NOT NULL)** — стабильный UUID документа,
  генерируется при создании. Используется как внешний ключ в интеграции
  (`entity.ExternalKey("OrganizationID", "ExchangeID")`).
- **UUID (TEXT UNIQUE)** — внешний UUID, присваивается 1С при синхронизации.
  Может быть NULL до первой синхронизации.

Ограничения уникальности:

| Сущность | Ограничение |
|----------|-------------|
| Organization | `UNIQUE(uuid)` |
| User | `UNIQUE(uuid)`, `UNIQUE(login)` |
| Customer | `UNIQUE(organization_id, uuid)` |
| Product | `UNIQUE(organization_id, uuid)` |
| Receipt | `UNIQUE(uuid)`, `UNIQUE(exchange_id)`, `UNIQUE(organization_id, number)` |
| ReceiptItem | — |

---

# 4. Schema Builder и миграции

Схема базы данных описывается декларативно в Go с помощью Schema Builder.

Каждый доменный пакет определяет свою таблицу:

```go
var Table = database.Must(database.NewTable("customers",
    database.Int("id").PrimaryKey().AutoIncrement(),
    database.String("uuid").NotNull(),
    database.Int("organization_id").NotNull().References("organizations", "id").OnDelete("CASCADE"),
    database.String("name").NotNull(),
    database.Bool("active").NotNull().Default(true),
    database.DateTime("created_at").NotNull().Default("CURRENT_TIMESTAMP"),
    database.DateTime("updated_at").NotNull().Default("CURRENT_TIMESTAMP"),
)).AddUniqueConstraint("organization_id", "uuid")
```

## Типы колонок

| Функция | SQLite type |
|---------|-------------|
| `String()` | TEXT |
| `Int()` | INTEGER |
| `Real()` | REAL |
| `Bool()` | INTEGER (0/1) |
| `DateTime()` | DATETIME |

## AddUniqueConstraint

`AddUniqueConstraint` добавляет составной UNIQUE-констрейнт к таблице:

```go
database.NewTable("customers", ...).AddUniqueConstraint("organization_id", "uuid")
```

Генерирует `UNIQUE (organization_id, uuid)` в CREATE TABLE.

## CreateSQL

```go
func (t Table) CreateSQL() string              // CREATE TABLE tablename (...)
func (t Table) CreateSQLIfNotExists() string   // CREATE TABLE IF NOT EXISTS tablename (...)
```

`CreateSQL()` используется для первоначального создания схемы (новая БД).
`CreateSQLIfNotExists()` — для миграций, где таблица может уже существовать.

## Сценарии RunMigrations

| Состояние | Действие |
|-----------|----------|
| Новая БД (v=0) | CREATE TABLE из описаний всех зарегистрированных Table, запись v = код |
| v == code version | Ничего |
| v < code version | Выполнить недостающие миграции по порядку |
| v > code version | Ошибка (БД новее кода) |

## Служебная таблица system_info

```sql
CREATE TABLE system_info (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
)
```

Хранит одну запись:

```
key = 'schema_version'
value = '<текущая версия>'
```

---

# 5. Entity Descriptor и entity.Register

Модели регистрируются через `entity.Register[T]()` с указанием ключей.
Поля с тегом `db` автоматически обнаруживаются.

```go
type Customer struct {
    ID     int64  `db:"id" order:"2"`
    UUID   string `db:"uuid" label:"ID" order:"5" list:"true"`
    Name   string `db:"name" label:"Наименование" order:"20" list:"true" search:"true"`
    Active bool   `db:"active" label:"Активен" order:"30" list:"true"`
    ...
}

var Descriptor = entity.Register[Customer](
    entity.PrimaryKey("ID"),
    entity.ExternalKey("OrganizationID", "UUID"),
)
```

Теги полей:

| Тег | Описание |
|-----|----------|
| `db` | Имя колонки в БД |
| `label` | Отображаемое имя |
| `order` | Порядок сортировки (int, обязателен для всех полей) |
| `list` | "true" = показывать в таблице |
| `search` | Информационный: поле кандидат для поиска. Поиск списка строит Store |
| `readonly` | "true" = только для отображения |

Descriptor предоставляет:

- `PrimaryKey()` — первичный ключ (всегда одно поле).
- `ExternalKey()` — внешний ключ (один или несколько полей).
- `ListFields()` — поля с `list:"true"`.

Поиск списка не использует тег `search`. Store определяет собственные
поисковые колонки (`searchableColumns()`) и сопоставляет им SQL-выражения,
возвращающие строку, которую видит пользователь. Handler передаёт только
видимые поля списка (`entity.FieldName`). Условие `WHERE` строит
`internal/database/search` (см. `docs/UI.md`).

---

# 6. База данных base.db

## Таблица Users

Назначение: Пользователи системы.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | Внутренний ID |
| uuid | TEXT UNIQUE | Внешний UUID |
| login | TEXT UNIQUE | Логин |
| email | TEXT | Email |
| password_hash | TEXT | Хэш пароля |
| is_admin | INTEGER | Администратор |
| created_at | DATETIME | Создан |
| updated_at | DATETIME | Изменен |

---

## Таблица Sessions

Назначение: Сессии пользователей.

| Поле | Тип | Описание |
|------|-----|----------|
| id | TEXT PK | Токен сессии |
| user_id | INTEGER | FK → users.id |
| flash_type | TEXT | Тип Flash-сообщения |
| flash_message | TEXT | Flash-сообщение |
| values_json | TEXT | Сериализованные значения сессии |
| user_agent | TEXT | User-Agent при создании |
| created_at | DATETIME | Создан |
| last_seen_at | DATETIME | Последняя активность |
| expires_at | DATETIME | Срок истечения |

---

## Таблица Organizations

Назначение: Организации.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | Внутренний ID |
| uuid | TEXT UNIQUE | Внешний UUID |
| name | TEXT | Наименование |
| api_key | TEXT UNIQUE | Ключ для API-интеграции |
| active | INTEGER | Активная |
| created_at | DATETIME | Создана |
| updated_at | DATETIME | Изменена |

---

## Таблица Customers

Назначение: Контрагенты.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | Внутренний ID |
| uuid | TEXT | Внешний UUID (из 1С или сгенерированный) |
| organization_id | INTEGER NOT NULL | FK → organizations.id |
| name | TEXT | Наименование |
| active | INTEGER | Активный |
| created_at | DATETIME | Создан |
| updated_at | DATETIME | Изменен |

Ограничения:
- `UNIQUE(organization_id, uuid)` — гарантирует уникальность UUID
  в пределах организации.
- `FOREIGN KEY(organization_id) REFERENCES organizations(id) ON DELETE CASCADE`.

---

## Таблица Products

Назначение: Товары.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | Внутренний ID |
| uuid | TEXT | Внешний UUID |
| organization_id | INTEGER NOT NULL | FK → organizations.id |
| name | TEXT | Наименование |
| unit | TEXT | Единица измерения |
| active | INTEGER | Активный |
| created_at | DATETIME | Создан |
| updated_at | DATETIME | Изменен |

Ограничения:
- `UNIQUE(organization_id, uuid)`.
- `FOREIGN KEY(organization_id) REFERENCES organizations(id) ON DELETE CASCADE`.

---

## Таблица Receipts

Назначение: Товарные чеки.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | Внутренний ID |
| uuid | TEXT UNIQUE | Внешний UUID (присваивается 1С, nullable) |
| exchange_id | TEXT UNIQUE NOT NULL | Локальный UUID, генерируется при создании |
| number | TEXT NOT NULL | Номер документа |
| date | TEXT NOT NULL | Дата документа (YYYY-MM-DD) |
| organization_id | INTEGER NOT NULL | FK → organizations.id |
| user_id | INTEGER NOT NULL | FK → users.id |
| customer_id | INTEGER NOT NULL | FK → customers.id |
| total | REAL | Итоговая сумма |
| sent_at | DATETIME | Дата отправки (когда документ готов к выдаче 1С) |
| status | TEXT | Статус (управляется 1С) |
| status_color | TEXT | Цвет статуса (управляется 1С) |
| created_at | DATETIME | Создан |
| updated_at | DATETIME | Изменен |

Ограничения:
- `UNIQUE(uuid)` — внешний UUID уникален (может быть NULL).
- `UNIQUE(exchange_id)` — локальный UUID уникален.
- `UNIQUE(organization_id, number)` — номер уникален в пределах организации.
- `FOREIGN KEY(organization_id) REFERENCES organizations(id)`.
- `FOREIGN KEY(user_id) REFERENCES users(id)`.
- `FOREIGN KEY(customer_id) REFERENCES customers(id)`.

---

## Таблица ReceiptItems

Назначение: Строки товарного чека.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | Внутренний ID |
| receipt_id | INTEGER NOT NULL | FK → receipts.id |
| line_num | INTEGER | Номер строки |
| product_id | INTEGER NOT NULL | FK → products.id |
| unit | TEXT | Единица измерения |
| quantity | REAL | Количество |
| price | REAL | Цена |
| amount | REAL | Сумма |

Ограничения:
- `FOREIGN KEY(receipt_id) REFERENCES receipts(id) ON DELETE CASCADE`.
- `FOREIGN KEY(product_id) REFERENCES products(id)`.

---

# 7. Статусы документов

`Status` — открытая строка (TEXT). Значение полностью управляется 1С
через `Synchronize()`. Orders не интерпретирует и не ограничивает статус.

Единственный внутренний статус — состояние отправки:

- `sent_at IS NULL` — документ редактируется.
- `sent_at IS NOT NULL` — документ опубликован для 1С, read-only.

Документ считается доступным для синхронизации, если:

```
sent_at IS NOT NULL
  AND uuid IS NULL
```

---

# 8. База данных files.db (запланировано)

*Раздел описывает целевую архитектуру. Реализация начнётся после
добавления функциональности PDF.*

## Таблица ReceiptFiles

Назначение: Хранение PDF-файлов документов.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | Первичный ключ |
| receipt_id | INTEGER | FK → receipts.id |
| file_name | TEXT | Имя файла |
| mime_type | TEXT | MIME-тип |
| file_size | INTEGER | Размер файла |
| file_data | BLOB | Содержимое PDF |
| created_at | DATETIME | Дата загрузки |
| updated_at | DATETIME | Изменен |

Связи:

- ReceiptFile принадлежит одному документу.
- На один документ может быть от нуля до нескольких файлов.

---

# 9. Связи

Receipt

- принадлежит одной организации;
- создан одним пользователем;
- относится к одному контрагенту;
- содержит много строк.

ReceiptItem

- принадлежит одному документу;
- содержит снимок данных товара.

Customer / Product

- принадлежат одной организации;
- идентифицируются UUID в пределах организации.

---

# 10. Правила хранения данных

Контрагенты и товары синхронизируются только через API.

Документы хранят снимок наименований товаров и контрагентов на момент создания.
Изменение справочников не влияет на ранее созданные документы.

Все справочники (кроме чеков) имеют dual-ID: внутренний `id` (int64, PK, используется
в URL и FK) и внешний `uuid` (TEXT, используется в API и синхронизации).

Чеки имеют три идентификатора: `id` (внутренний), `exchange_id` (локальный UUID
для интеграции), `uuid` (внешний UUID от 1С, может быть NULL).

---

# 11. Правила хранения PDF (запланировано)

PDF-файлы будут сохраняться только в базе данных `files.db`.

Каждый файл хранится как BLOB.

Файлы никогда не изменяются после сохранения в интерфейсе.
Файлы могут обновляться через API.
Удаление PDF через пользовательский интерфейс не предусмотрено.

---

# 12. Резервное копирование

Для полного резервного копирования приложения необходимо сохранить:

- base.db
- files.db (после реализации)

Других данных, необходимых для восстановления системы, приложение не хранит.

---

# 13. Миграции

## Версия 2

- **Название:** Redesign customers: composite PK (organization_id, id)
- **Операция:** DROP customers, CREATE customers с новым составным ключом.

## Версия 3

- **Название:** Add products table
- **Операция:** DROP products, CREATE products.

## Версия 4

- **Название:** Unified schema: internal ID (INTEGER PK) + external UUID for all dictionaries
- **Операция:** DROP всех таблиц (sessions, customers, products, organizations, users),
  CREATE всех таблиц заново с dual-ID архитектурой:
  - Все сущности получают `id INTEGER PRIMARY KEY AUTOINCREMENT` и `uuid TEXT NOT NULL`.
  - Customers/Products получают `organization_id INTEGER` как FK на organizations.id.
  - Customers/Products получают `UNIQUE(organization_id, uuid)`.
  - Organizations/Users получают `UNIQUE(uuid)`.
  - Sessions — исключение (строковый PK).

## Версия 5

- **Название:** Add receipts tables
- **Операция:** CREATE TABLE IF NOT EXISTS receipts, CREATE TABLE IF NOT EXISTS receipt_items.
- Без DROP — идемпотентно, безопасно для существующих данных.
