# Orders Server — Архитектура

Версия: 1.0

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
/set-password   LandingURL(user)
                (/dashboard | /receipts)
```

---

## Application Entry

`/` — единая точка входа в приложение (`internal/app/home.go`).

`Home()` проверяет только сессию и всегда делегирует выбор маршрута
`LandingURL()`. Он не содержит собственной логики выбора первого экрана:

```
GET /
    ↓
нет сессии           → /login
нет пользователя     → очистить сессию → /login
нужно сменить пароль → /set-password
администратор        → /dashboard
обычный пользователь → /receipts
```

`LandingURL()` — единственный источник выбора стартовой страницы после
аутентификации (`internal/app/landing.go`). И `Home()`, и редирект после
входа используют одну и ту же функцию. Выбор маршрута в `Home()`
запрещён — вся логика живёт только в `LandingURL()`.

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

## Dual-ID Architecture (ID + UUID)

Все сущности (справочники, пользователи, организации) имеют два идентификатора:

- `ID int64` — внутренний первичный ключ, автоинкремент. Используется в:
  - URL веб-интерфейса (`/organizations/{oid}/customers/{id}`);
  - внешних ключах (`user_id`, `organization_id`);
  - JOIN и внутренних запросах.

- `UUID string` — внешний идентификатор. Используется в:
  - API-запросах (lookup по UUID);
  - синхронизации с внешними системами (1С, API);
  - идентификации на транспортном уровне.

Организации и пользователи имеют `UNIQUE`-ограничение на UUID. Справочники
в контексте организации — составной `UNIQUE(organization_id, uuid)`.

Web UI никогда не использует UUID в URL. Интеграционное API никогда
не использует int64 ID в запросах.

**Исключение:** сессии имеют строковый первичный ключ (токен сессии)
и не следуют dual-ID шаблону.

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
    entity/       — Descriptor, Register, Key (PrimaryKey, ExternalKey)
    users/        — домен пользователей
    sessions/     — домен сессий
    customers/    — домен контрагентов
    products/     — домен товаров
    organizations/ — домен организаций
    common/       — утилиты (GenerateUUID)
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

# 11. Entity Descriptor и Key System

## entity.Descriptor

Каждый домен регистрирует свою модель через `entity.Register[T]()`.
Descriptor содержит мета-информацию о полях, ключах и тегах.

### Поля

Поля модели помечаются тегом `db` с именем колонки. Поле без `db`
игнорируется (кроме `readonly:"true"`).

```go
type Customer struct {
    ID               int64  `db:"id" order:"2"`
    UUID             string `db:"uuid" label:"ID" order:"5" list:"true"`
    OrganizationID   int64  `db:"organization_id" order:"3"`
    OrganizationName string `readonly:"true" label:"Организация" order:"15" list:"true"`
    Name             string `db:"name" label:"Наименование" order:"20" list:"true" search:"true"`
    Active           bool   `db:"active" label:"Активен" order:"30" list:"true"`
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

Теги:
- `db` — имя колонки в БД.
- `label` — отображаемое имя в UI.
- `order` — порядок сортировки полей.
- `list` — показывать в таблице списка.
- `search` — участвует в поиске.
- `readonly` — только для отображения, не сохраняется.

### Key System

Descriptor содержит два типа ключей:

1. **PrimaryKey** — единственное поле, идентифицирующее внутренний int64 ID.
   Обязателен. Должен ссылаться на поле с `db:"id"`.

2. **ExternalKey** — одно или несколько полей, идентифицирующих запись
   для внешнего API. Составной ключ (OrganizationID + UUID) для справочников,
   простой (UUID) для организаций и пользователей.

```go
var Descriptor = entity.Register[Customer](
    entity.PrimaryKey("ID"),
    entity.ExternalKey("OrganizationID", "UUID"),
)
```

Регистрация паникует при:
- отсутствии PrimaryKey;
- нескольких PrimaryKey;
- ссылке на несуществующее поле;
- дублировании order.

---

# 12. Справочники: единый шаблон

Все справочники платформы Orders следуют единому шаблону.

## Архитектурные правила

**Организация является корнем бизнес-модели Orders.** Все бизнес-объекты
(справочники, документы и регистры) существуют в контексте организации
и адресуются через маршрут `/organizations/{oid}/...`.

**Идентификация:** Объект имеет два идентификатора:
- `ID int64` — внутренний автоинкрементный PK, используется в URL веб-интерфейса.
- `UUID string` — внешний идентификатор, используется в API и синхронизации.

**Проверка организации:** OrganizationID (int64) является FK на organizations.id.
При создании проверяется существование организации через `SELECT EXISTS`.

**Два режима доступа:**

1. **Глобальный (администратор):** `GET /{resource}` — read-only список
   всех объектов. Создание через `/organizations/0/{resource}/0`.
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
    ID               int64     `db:"id"`
    UUID             string    `db:"uuid"`
    OrganizationID   int64     `db:"organization_id"`
    OrganizationName string    `readonly:"true"`
    Name             string    `db:"name"`
    Active           bool      `db:"active"`
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

// Descriptor — через entity.Register с PrimaryKey и ExternalKey
var Descriptor = entity.Register[Entity](
    entity.PrimaryKey("ID"),
    entity.ExternalKey("OrganizationID", "UUID"),
)

// Schema — Table через Schema Builder с AUTOINCREMENT PK
// .AddUniqueConstraint("organization_id", "uuid")

// Store — единый интерфейс методов
New() *T
GetByID(ctx, id) (*T, error)
GetByExternal(ctx, organizationID, uuid) (*T, error)
List(ctx, organizationID) ([]*T, error)           // 0 = все организации
Save(ctx, *T) error                                // 0 = INSERT, иначе UPDATE
DeleteByID(ctx, id) error
DeleteByExternal(ctx, organizationID, uuid) error
Synchronize(ctx, organizationID, items) (Result, error)  // upsert для интеграции

// HTTP — nested маршруты
GET    /{resource}                                    — глобальный список
GET    /organizations/{oid}/{resource}/{id}            — карточка (0 = создание)
POST   /organizations/{oid}/{resource}                 — сохранение (admin)
DELETE /organizations/{oid}/{resource}/{id}            — удаление (admin)
PUT    /api/integration/organizations/{oid}/{resource} — синхронизация (API key)

// Templates — список + карточка
// Rights — RequireAdmin для мутаций
```

Customers и Products — эталонные реализации данного шаблона.

## Правила Save

- `ID == 0` → `GenerateUUID()` → INSERT.
- `ID != 0` → UPDATE по `id`. 0 rows → `ErrNotFound`.
- `OrganizationID` проверяется через `SELECT EXISTS` — ошибка при отсутствии.

## Synchronize contract

```
Input:
    oid (из URL)
    []Entity

Rules:
    - одна транзакция (all-or-nothing)
    - дубликаты UUID в запросе → ошибка, полный откат
    - пустой UUID запрещён → ошибка
    - OrganizationID из URL, не из тела запроса
    - UPDATE по (organization_id, uuid)
    - 0 rows → INSERT (upsert)
    - Нет DELETE
    - Нет deactivate
    - Нет поиска по Name или другим полям
```

---

# 13. Шаблон для организаций и пользователей

Организации и пользователи — глобальные сущности, не привязанные к контексту
организации. Они используют упрощённый шаблон:

- `ExternalKey("UUID")` — один UUID, без OrganizationID.
- `GetByUUID(ctx, uuid)` вместо `GetByExternal`.
- `DeleteByUUID(ctx, uuid)` вместо `DeleteByExternal`.
- `List(ctx)` без параметра организации.

```go
var Descriptor = entity.Register[Organization](
    entity.PrimaryKey("ID"),
    entity.ExternalKey("UUID"),
)
```

---

# 14. Store naming conventions

| Сущность | Lookup по PK | Lookup по ExternalKey | Delete по PK | Delete по ExternalKey |
|-----------|-------------|----------------------|-------------|----------------------|
| Organization | GetByID | GetByUUID | DeleteByID | DeleteByUUID |
| User | GetByID | GetByUUID | DeleteByID | DeleteByUUID |
| Customer | GetByID | GetByExternal(orgID, uuid) | DeleteByID | DeleteByExternal(orgID, uuid) |
| Product | GetByID | GetByExternal(orgID, uuid) | DeleteByID | DeleteByExternal(orgID, uuid) |

- Единичные сущности (без OrganizationID): `GetByUUID` / `DeleteByUUID`.
- Справочники в контексте организации: `GetByExternal(orgID, uuid)` / `DeleteByExternal(orgID, uuid)`.

---

# 15. IdentityService

IdentityService — резидентный сервис аутентификации и авторизации.
Содержит минимальный набор данных, необходимых для обработки каждого
HTTP-запроса.

## Инварианты

### 1. Единственный источник хранения пользователей — UserStore

IdentityService не записывает данные в БД.
IdentityService является runtime-кэшем.
Восстановление состояния выполняется через `Load()` или `Reload()`.

### 2. HTTP-запросы не обращаются к таблице users

После запуска приложения обычные HTTP-запросы не обращаются к таблице `users`:

- аутентификация использует IdentityService;
- авторизация (RequireAdmin, RequirePassword) использует IdentityService;
- построение Layout использует IdentityService.

### 3. Все изменения пользователей синхронно отражаются в IdentityService

- Создание → `Add()`
- Изменение → `Update()`
- Удаление → `Remove()`
- Массовые изменения → `Reload()`

### 4. Identity — неизменяемое представление пользователя

Внешний код получает копию (value type), а не указатель.
Изменение полученной Identity не влияет на состояние сервиса.

### 5. Нормализация логина — единообразно через NormalizeLogin()

Поиск всегда ведётся по нормализованному логину.
Отображение использует оригинальный логин.

### 6. Инвариант: последний администратор

После любой операции создания, изменения или удаления пользователей
в системе должен существовать хотя бы один пользователь с правами
администратора.

Проверка должна выполняться **перед любой операцией, которая может
уменьшить количество администраторов** (удаление пользователя, снятие
прав администратора и другие подобные операции).

Нарушение возвращает `ErrLastAdministrator`.

Поиск всегда ведётся по нормализованному логину.
Отображение использует оригинальный логин.

## Startup Flow

```
app.New()
    │
    ├── database.Open()
    ├── schema.RunMigrations()
    ├── users.NewStore(db)
    ├── users.Seed(store)              — создаёт admin при необходимости
    ├── users.NewIdentityService()
    ├── identity.Load(ctx, store)      — SELECT всех пользователей в память
    ├── sessions.NewStore(db)
    ├── organizations.NewStore(db)
    └── ...
```

После `identity.Load()` таблица `users` участвует только в CRUD-операциях
(UserSave, UserDelete, SetPasswordSubmit) и больше не читается на hot path.

## Request Flow

```
Request
    │
    ├── RequestID
    ├── RealIP
    ├── ClientIPFromXFF(trusted proxies)   ← доверенный IP клиента в context
    ├── Logger
    ├── Recoverer
    ├── SessionMiddleware (cookie → session)
    │
    ├── RequireAuth (session → identity.GetByID → context)
    │
    ├── RequireAdmin (context → Identity.IsAdmin)
    │
    └── Handler (context → Identity.Login, .IsAdmin, .NeedsPasswordSetup)
```

### Login Request Chain

`POST /login` дополнительно обёрнут в два слоя (порядок важен):

```
guard → rate limiter → Login
```

1. **Request Guard** — защита от переполнения формы: ограничивает размер тела
   (413), число полей и длину значений (400). Работает до RateLimiter, поэтому
   переполненные запросы отклоняются раньше счётчика попыток.
2. **Rate Limiter** — защита от перебора паролей. Два независимых лимита:
   по IP и по аккаунту (нормализованный логин). HTTP 429.

Guard и RateLimiter возвращают только инфраструктурные ошибки и не знают про
HTML, HTMX и JSON — они вызывают `RenderInfrastructureError`.

### Trusted Proxies

`ClientIPFromXFF("127.0.0.1/8", "::1/128")` допустимо только за локальным
reverse proxy (Caddy), который на каждом запросе перезаписывает
`X-Forwarded-For`. Rate Limiter читает IP из контекста (`GetClientIP`) с
fallback на `RemoteAddr`. При изменении схемы развёртывания (другой прокси,
несколько хопов, прямое подключение клиентов) доверенные прокси необходимо
пересмотреть.

## Data

```go
type Identity struct {
    ID              int64
    UUID            string
    Login           string    // оригинал (отображается в UI)
    NormalizedLogin string    // для поиска (NormalizeLogin)
    PasswordHash    string
    IsAdmin         bool
}

type IdentityService struct {
    mu      sync.RWMutex
    byID    map[int64]Identity
    byLogin map[string]Identity  // key: NormalizedLogin
}
```

## Принцип восстанавливаемости

IdentityService полностью восстанавливается из persistent-слоя:

```
SQLite → IdentityService.Load() → Runtime
```

Никаких уникальных данных только в памяти.
Перезапуск приложения безопасен: состояние полностью восстанавливается
из UserStore.

---

# 16. Object Editor

Object Editor — платформенный механизм построения карточек объектов.
Полное описание: `RFC-017-object-editor.md`.

Три фундаментальных понятия:

1. **Object Descriptor** — метаданные объекта (поля, типы, обязательность,
   зависимости, действия). Editor читает Descriptor, но не знает имён полей.

2. **Editor Engine** — связующий слой: итерирует Descriptor, выбирает
   FieldComponent по типу поля, применяет DependsOn, ReadOnly, ошибки.

3. **Editor Context (Alpine)** — минимальное состояние редактора в браузере:
   `values`, `errors`, `dirty`. Alpine управляет только UI-состоянием
   (`:disabled`), бизнес-логика — только на сервере.

Слои:

```
Object Descriptor
        ↓
Editor Engine
        ↓
Editor Context (Alpine)
        ↓
HTML + htmx
        ↓
Server Validation (ValidationError)
        ↓
Fragment → транспортное представление / FullPage → 422 + форма
```

Первый пользователь: Receipt (Commit B).

---

# 16.1. ValidationError

`ValidationError` — платформенная модель ошибок валидации. Располагается в
`internal/app/validation.go`; `ValidationResponse` — её сериализуемое
представление (`internal/app/validation_response.go`).

```
ValidationError
        ↓
NewValidationResponse()
        ↓
ValidationResponse
        ↓
WriteValidationResponse()   (транспорт, сегодня — JSON)
```

Правила:

1. **`ValidationError` — единственная модель ошибок валидации.**
   `ValidationResponse` — единственное сериализуемое представление
   `ValidationError`. Любая форма платформы обязана использовать их
   независимо от способа отображения (HTML, HTMX, API). Параллельные
   механизмы (`map[string]string`, `[]string`, голый `error`) запрещены.
2. **`ValidationError` описывает только ошибки пользовательского ввода.**
   Внутренние ошибки приложения (ошибки БД, файловой системы, сети, паники
   и т.п.) не передаются через `ValidationError`.
3. **`ValidationError` не содержит локализованных кодов, HTTP-идентификаторов,
   статусов ответа и информации о транспорте.** Модель ничего не знает о
   представлении; преобразование в представление — ответственность
   транспортного слоя (`NewValidationResponse`).
4. **Fragment → транспортное представление ValidationError** (сегодня это
   JSON `{title, errors, fields}`). **FullPage → 422 + форма с ошибками**.
   Успех в обоих режимах — обычный редирект (`HX-Redirect` для htmx,
   `303` без htmx).

Выбор режима ответа — `ResponseModeFromRequest()`; единая точка ветвления —
диспетчер на уровне App (например, `RenderReceiptValidationError`), запись
в HTTP — `WriteValidationResponse`. Способ доставки (JSON сегодня, что-то
другое завтра) — внутренняя деталь платформы.

---

# 16.2. InfrastructureError

Инфраструктурные ответы — вторая семья структурированных ответов платформы
(после `ValidationError`). Отличаются от ошибок валидации тем, что описывают
не пользовательский ввод, а отклонение запроса до его обработки: некорректная
форма (400), слишком большое тело (413), превышение лимита (429).

```
RenderInfrastructureError()        ← единая точка доставки
    ├── Fragment → JSON InfrastructureResponse (реальный HTTP-статус)
    └── FullPage → RenderPageStatus (форма + alert, статус)
```

Расположение:

- `ResponseKind`, `ResponseMessage`, `InfrastructureResponse` — в
  `internal/app/infrastructure_error.go`;
- `RenderInfrastructureError` — единый диспетчер в `internal/app/httperror.go`.

Правила:

1. **`ResponseMessage` — единственное место хранения общих пользовательских
   сообщений структурированных ответов.** Поля `Title` и `Errors` не
   копируются между моделями; конкретные ответы (`ValidationResponse`,
   `InfrastructureResponse`) получают их через композицию (`ResponseMessage`).
2. **`InfrastructureResponse` несёт свой `kind` (типизированный
   `ResponseKind`) и реальный HTTP-статус в payload** — для клиентов,
   обрабатывающих тело независимо от HTTP. Ошибки валидации статус в payload
   не выносят (всегда 200 + форма), поэтому использование общего контракта
   (`ResponseMessage` и т.п.) оправдано, а общий transport-конверт вводится
   только когда им пользуются ≥2 независимых модуля.
3. **Guard и RateLimiter не знают про HTML, HTMX и JSON.** Они возвращают
   ошибку через `RenderInfrastructureError`; выбор представления — внутренняя
   деталь платформы.
4. **Защита входа** (`internal/app/request_guard.go`, `internal/app/ratelimit.go`):
   лимит по IP и по аккаунту в секундах/количестве попыток приходит из
   `config.json` (`rate_limit`), дефолты применяются при отсутствии секции.

---

# 17. Будущее развитие

Архитектура должна позволять:

- добавление новых видов документов;
- интеграцию с несколькими внешними системами;
- добавление пользователей и ролей;
- замену мобильного интерфейса без изменения серверной логики.