# Orders Server — Структура базы данных

Версия: 1.0

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

Такое разделение позволяет:

- уменьшить размер рабочей базы данных;
- сохранить высокую производительность пользовательского интерфейса;
- выполнять резервное копирование файлов независимо от рабочих данных;
- при необходимости переносить архив документов отдельно.

---

# 2. Общие требования

Для обеих баз данных:

- используется SQLite;
- используется режим WAL;
- включены внешние ключи (`PRAGMA foreign_keys = ON`);
- таблицы создаются автоматически при запуске приложения.

---

# 3. Dual-ID Architecture

Все сущности (кроме сессий) имеют два идентификатора:

- **ID (INTEGER)** — внутренний первичный ключ. Автоинкремент. Используется
  в URL веб-интерфейса, внешних ключах и JOIN. Тип `INTEGER` с `PRIMARY KEY AUTOINCREMENT`.

- **UUID (TEXT)** — внешний идентификатор. Используется в API и синхронизации
  с внешними системами. Тип `TEXT NOT NULL`.

Ограничения уникальности:

| Сущность | Ограничение |
|----------|-------------|
| Organization | `UNIQUE(uuid)` |
| User | `UNIQUE(uuid)` |
| Customer | `UNIQUE(organization_id, uuid)` |
| Product | `UNIQUE(organization_id, uuid)` |

Сессии — исключение: используют строковый первичный ключ (токен сессии).

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

При запуске приложения:

```go
schema := database.NewSchema()
schema.Register(users.Table)
schema.Register(sessions.Table)

db, err := database.OpenPath(config.DatabasePath)
schema.RunMigrations(db)
```

## AddUniqueConstraint

`AddUniqueConstraint` добавляет составной UNIQUE-констрейнт к таблице:

```go
database.NewTable("customers", ...).AddUniqueConstraint("organization_id", "uuid")
```

Генерирует `UNIQUE (organization_id, uuid)` в CREATE TABLE.

## Сценарии RunMigrations

| Состояние | Действие |
|-----------|----------|
| Новая БД (v=0) | CREATE TABLE из описаний, запись v=1 |
| Старая система (v=4) | Одноразовый переход |
| v == code version | Ничего |
| v < code version | Выполнить недостающие миграции по порядку |
| v > code version | Ошибка (БД новее кода) |

## Служебная таблица system_info

```sql
key   TEXT PRIMARY KEY
value TEXT NOT NULL
```

Хранит одну запись:

```
key = 'schema_version'
value = '1'
```

---

# 5. Entity Descriptor и entity.Register

Модели регистрируются через `entity.Register[T]()` с указанием ключей.
Поля с тегом `db` автоматически обнаруживаются.

```go
type Customer struct {
    ID   int64  `db:"id" order:"2"`
    UUID string `db:"uuid" label:"ID" order:"5" list:"true"`
    Name string `db:"name" label:"Наименование" order:"20" list:"true" search:"true"`
    Active bool `db:"active" label:"Активен" order:"30" list:"true"`
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
| `order` | Порядок сортировки (int) |
| `list` | "true" = показывать в таблице |
| `search` | "true" = участвует в поиске |
| `readonly` | "true" = только для отображения |

Descriptor предоставляет:

- `PrimaryKey()` — первичный ключ (всегда одно поле).
- `ExternalKey()` — внешний ключ (один или несколько полей).
- `ListFields()` — поля с `list:"true"`.
- `SearchFields()` — поля с `search:"true"`.

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
| is_admin | BOOLEAN | Администратор |
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
| active | BOOLEAN | Активная |
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
| active | BOOLEAN | Активный |
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
| active | BOOLEAN | Активный |
| created_at | DATETIME | Создан |
| updated_at | DATETIME | Изменен |

Ограничения:
- `UNIQUE(organization_id, uuid)`.
- `FOREIGN KEY(organization_id) REFERENCES organizations(id) ON DELETE CASCADE`.

---

## Таблица Orders

Назначение: Товарные чеки.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER | Первичный ключ |
| customer_uuid | TEXT | UUID контрагента |
| customer_name | TEXT | Наименование контрагента на момент создания |
| document_date | DATE | Дата документа |
| status | INTEGER | Статус |
| total | DECIMAL | Итоговая сумма |
| created_at | DATETIME | Создан |
| updated_at | DATETIME | Изменен |

---

## Таблица OrderItems

Назначение: Строки товарного чека.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER | Первичный ключ |
| order_id | INTEGER | Документ |
| product_uuid | TEXT | UUID товара |
| product_name | TEXT | Наименование товара на момент создания |
| unit | TEXT | Единица измерения |
| quantity | DECIMAL | Количество |
| price | DECIMAL | Цена |
| amount | DECIMAL | Сумма |

---

# 7. База данных files.db

## Таблица OrderFiles

Назначение: Хранение PDF-файлов документов.

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER | Первичный ключ |
| order_id | INTEGER | Идентификатор документа |
| file_name | TEXT | Имя файла |
| mime_type | TEXT | MIME-тип |
| file_size | INTEGER | Размер файла |
| file_data | BLOB | Содержимое PDF |
| created_at | DATETIME | Дата загрузки |
| updated_at | DATETIME | Изменен |

---

# 8. Статусы документов

| Код | Статус |
|------|---------|
| 0 | Создан |
| 1 | Отправлен |
| 2 | Обработан |
| 3 | Отменен |

---

# 9. Связи

## base.db

Order

- имеет одного контрагента;
- содержит много строк.

OrderItems

- принадлежит одному документу;
- содержит снимок данных товара.

Customer / Product

- принадлежат одной организации;
- идентифицируются UUID в пределах организации.

## files.db

OrderFiles

- принадлежит одному документу;
- может существовать в количестве от нуля до нескольких файлов на один документ.

Связь осуществляется по полю `order_id`.

---

# 10. Правила хранения данных

Контрагенты и товары синхронизируются только через API.

Документы хранят снимок наименований товаров и контрагентов на момент создания.

Изменение справочников не влияет на ранее созданные документы.

Все справочники имеют dual-ID: внутренний `id` (int64, PK, используется
в URL и FK) и внешний `uuid` (TEXT, используется в API и синхронизации).

---

# 11. Правила хранения PDF

PDF-файлы сохраняются только в базе данных `files.db`.

Каждый файл хранится как BLOB.

Файлы никогда не изменяются после сохранения в интерфейсе.

Файлы могут обновляться через api.

Удаление PDF через пользовательский интерфейс не предусмотрено.

---

# 12. Резервное копирование

Для полного резервного копирования приложения необходимо сохранить:

- base.db
- files.db

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