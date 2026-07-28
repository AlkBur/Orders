# Orders Server — Архитектура

Версия: 0.1

---

# 1. Назначение

Orders Server — веб-приложение для создания, отправки и просмотра товарных чеков.

Приложение предназначено для небольших организаций и оптимизировано для работы со смартфонов.

---

# 2. Цели проекта

Основные цели:

- простота эксплуатации;
- минимальное количество зависимостей;
- простое развертывание;
- высокая надежность;
- возможность работы на недорогом VPS;
- адаптация под мобильные устройства.

---

# 3. Архитектурные принципы

При проектировании используются следующие принципы.

## Простота

Предпочтение всегда отдается более простому решению.

Если задачу можно решить средствами стандартной библиотеки Go, использование сторонних библиотек не допускается.

---

## Server Side Rendering

Интерфейс полностью формируется сервером.

Клиент получает готовые HTML-страницы.

Использование SPA-фреймворков не предусматривается.

---

## Mobile First

Основной пользователь работает со смартфона.

Все экраны проектируются в первую очередь для мобильного устройства.

Интерфейс для планшетов и ПК является адаптацией мобильной версии.

---

## Authentication Principles

1. Cookie never contains authentication state.
2. Authentication does not imply session creation.
   Session creation is a consequence of successful authentication.
3. Session is created only after successful authentication.
4. Authentication is performed through explicit authentication flows.

---

## Password State

The absence of a password is an account state.

It is not an authentication method.

`NeedsPasswordSetup()` exists in the model solely to represent
this state, not to serve as an authentication gate.

Authentication is always performed through an explicit
authentication flow.

---

## Authentication Flow

```
Login
  │
  ▼
Find User
  │
  ▼
Determine Authentication Method
  │     │
 YES   NO
(has   (no
password)  password)
  │     │
  ▼     ▼
Verify User   Verify Initial
Password      Password
  │     │
  └──┬──┘
     ▼
Authenticated
     │
     ▼
Needs Password Setup?
  │     │
 YES   NO
  │     │
  ▼     ▼
/set-password   /orders
```

---

## SQLite

SQLite является единственной базой данных приложения.

Используется режим WAL.

Использование PostgreSQL или других СУБД не предусматривается.

---

## API как источник справочников

Справочники товаров и контрагентов синхронизируются только через внешнее API.

Приложение не является владельцем этих данных.

---

# 4. Используемые технологии

| Компонент | Технология |
|-----------|------------|
| Язык | Go |
| Web | net/http |
| HTML | html/template |
| База данных | SQLite |
| Reverse Proxy | Caddy |
| ОС | Ubuntu Linux |

---

# 5. Структура приложения

```
Интернет
      │
      ▼
    Caddy
      │
      ▼
Go Application
      │
      ▼
    SQLite
```

---

# 6. Структура каталогов

```
cmd/
    server/       — точка входа
internal/
    app/          — composition root (сборка приложения)
    database/     — Schema Builder, миграции, OpenPath
    users/        — домен пользователей
    sessions/     — домен сессий
    customers/    — домен контрагентов
    organizations/ — домен организаций
    ui/           — UI-компоненты
    testutil/     — утилиты для тестов
data/             — SQLite базы данных
docs/             — документация
static/           — статические файлы (CSS, JS)
```

Единственным источником истины для структуры БД является
`internal/database/` — Schema Builder и миграции.

---

# 7. Бизнес-модель

Основной объект системы — товарный чек.

Все остальные сущности используются только для его создания и просмотра.

Пользователь работает исключительно с товарными чеками.

---

# 8. Источник данных

Источник данных               | Владельцем является
-----------------------------|---------------------
Товарные чеки                | Orders Server
Товары                       | Внешняя система
Контрагенты                  | Внешняя система
PDF-файлы                    | Внешняя система

---

# 9. Основные ограничения

Через пользовательский интерфейс запрещено:

- создавать товары;
- изменять товары;
- удалять товары;
- создавать контрагентов;
- изменять контрагентов;
- удалять контрагентов;
- загружать PDF;
- удалять PDF.

Все перечисленные операции выполняются исключительно через API.

---

# 10. Правило неизменяемости документа

Товарный чек может редактироваться только в статусе **Создан**.

После отправки документ становится неизменяемым.

Дальнейшее изменение статуса возможно только через API.

---

# 11. Справочники: единый шаблон

Все справочники платформы Orders следуют единому шаблону.

## Архитектурные правила

**Организация является корнем бизнес-модели Orders.** Все бизнес-объекты
(справочники, документы и регистры) существуют в контексте организации
и адресуются через маршрут `/organizations/{oid}/...`.

**Идентификация:** Объект идентифицируется парой `(OrganizationID, ID)`.
Составной первичный ключ. Никаких суррогатных ключей и ExternalID.

**NilUUID** (`00000000-0000-0000-0000-000000000000`) — единственное
специальное значение транспортного уровня (URL/UI). NilUUID никогда
не сохраняется в базе данных:

- URL, формы, DTO, модели до вызова `Save()` — разрешён.
- Хранение в таблицах, `Synchronize()`, возврат из `Store.Get()` — запрещён.

**URL — источник истины** для OrganizationID. Из формы — только при
oid == NilUUID (создание из глобального списка).

**Два режима доступа:**

1. **Глобальный (администратор):** `GET /{resource}` — read-only список
   всех объектов. Создание через `/organizations/NilUUID/{resource}/NilUUID`.
2. **Контекст организации (authenticated):** все мутации через
   `/organizations/{oid}/{resource}`.

**Права доступа:**

- Администратор: просмотр, создание, изменение, удаление.
- Пользователь: только просмотр (view only).

Проверка прав — через middleware `RequireAdmin`, не в хендлерах.

## Шаблон справочника

Каждый справочник реализует:

```go
// Model — чистая структура, без знаний о БД
type Entity struct {
    OrganizationID string
    ID             string
    Name           string
    Active         bool
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// Schema — Table через Schema Builder с composite PK
// .SetPrimaryKey("organization_id", "id")

// Store — единый интерфейс методов
New() *T
Get(ctx, organizationID, id) (*T, error)
List(ctx, organizationID) ([]*T, error)    // NilUUID = все организации
Save(ctx, *T) error                         // NilUUID → INSERT, иначе UPDATE
Delete(ctx, organizationID, id) error
Synchronize(ctx, organizationID, items) (Result, error)  // upsert для интеграции

// HTTP — nested маршруты
GET    /{resource}                                    — глобальный список
GET    /organizations/{oid}/{resource}/{id}            — карточка (NilUUID = создание)
POST   /organizations/{oid}/{resource}                 — сохранение (admin)
DELETE /organizations/{oid}/{resource}/{id}            — удаление (admin)
PUT    /api/integration/organizations/{oid}/{resource} — синхронизация (API key)

// Templates — список + карточка
// Rights — RequireAdmin для мутаций
```

Customers — эталонная реализация данного шаблона.

## Правила Save

- `ID == NilUUID` → `GenerateUUID()` → INSERT.
- `ID != NilUUID` → UPDATE по `(OrganizationID, ID)`. 0 rows → `ErrNotFound`.
- `OrganizationID` неизменяем после сохранения (обеспечивается URL source of truth).

## Synchronize contract

```
Input:
    oid (из URL)
    []Entity

Rules:
    - одна транзакция (all-or-nothing)
    - дубликаты ID в запросе → ошибка, полный откат
    - NilUUID запрещён → ошибка
    - OrganizationID из URL, не из тела запроса
    - UPDATE по (OrganizationID, ID)
    - 0 rows → INSERT (upsert)
    - Нет DELETE
    - Нет deactivate
    - Нет поиска по Name или другим полям
```

---

# 12. Будущее развитие

Архитектура должна позволять:

- добавление новых видов документов;
- интеграцию с несколькими внешними системами;
- добавление пользователей и ролей;
- замену мобильного интерфейса без изменения серверной логики.