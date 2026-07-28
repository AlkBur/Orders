# WORKFLOW

Процесс разработки. AGENTS.md определяет правила поведения агента.
Этот документ описывает процедуры.

Официальная среда выполнения Makefile: Linux, WSL, Git Bash.
cmd.exe не является целевой средой.

---

## 1. Planning

### Before Implementation

- изучить RFC (если есть)
- изучить docs/ADR/ (история архитектурных решений)
- изучить связанную документацию (ARCHITECTURE, DATABASE, UI, etc.)

### Planning

- составить план реализации
- согласовать с пользователем

---

## 2. Implementation

- следовать AGENTS.md
- одна логическая единица за Commit

### Pre-Release Migration Corrections

До первого публичного релиза допускается изменение или повторное
применение миграций на локальной базе разработки. Для этого
разрешается вручную понизить `schema_version` в `system_info`
и повторно выполнить миграции.

После первого публичного релиза история миграций становится
неизменяемой (append-only). Любые изменения схемы выполняются
только через новые миграции.

---

## 3. Verification

### Definition of Done (обязательно)

- [ ] `make verify`
- [ ] Acceptance Tests из `docs/ACCEPTANCE.md`
- [ ] Regression: предыдущие Acceptance Tests пройдены
- [ ] `make agent-clean` (temp, reports очищены)
- [ ] тестовый сервер остановлен

### При необходимости

- Unit Tests
- Integration Tests

### Test Suites

#### Unit Tests

Проверяют один пакет, без HTTP, без БД (можно с SQLite in-memory).

Примеры:
- `sessions.Store.Create()`
- `sessions.Store.FindByID()`
- `sessions.Store.Delete()`
- `sessions.Store.Flash`
- `sessions.Store.Values`
- `sessions.Store.Cleanup()`
- Пакет `users`: хеширование, валидация

#### Integration Tests

Проверяют взаимодействие нескольких компонентов.

Способ проверки определяется характером изменений:

- автоматические Go-тесты
- curl
- браузер
- комбинация способов

#### Acceptance Tests

Проверяют новую функциональность.

Список сценариев: `docs/ACCEPTANCE.md`

#### Regression Tests

Проверяют, что существующая функциональность не нарушена.

После каждого Commit агент обязан проверить, что Acceptance Tests
предыдущих Commit пройдены.

---

## 4. Reporting

После каждого Commit агент предоставляет отчёт по шаблону ниже.

### Agent Report Template

```markdown
## Commit N

### Changed files
- ...

### Verification
- [ ] make verify
- [ ] Acceptance Tests (N/N)
- [ ] Regression
- [ ] make agent-clean
- [ ] сервер остановлен

### Notes
- архитектурные решения
- known limitations

### Risks
- ...

### Future work
- ...
```

---

## When updating documentation

Если изменение влияет на:

- архитектуру проекта
- тесты / acceptance criteria
- workflow / process
- правила поведения агента (AGENTS.md)

агент обязан проверить и обновить связанные документы.

---

## ADR (Architecture Decision Records)

ADR создаётся **не на каждый Commit**, а только когда принимается
или изменяется архитектурное решение, влияющее на устройство системы.

Пример:

```
Commit 3
├── Исправление бага          → нет ADR
├── Новый endpoint            → нет ADR
├── Рефакторинг               → нет ADR
└── Переход на server sessions → ADR-001
```

ADR хранятся в `docs/ADR/`.

---

## 5. Go Build Cache

Проект использует `modernc.org/sqlite` — его полная перекомпиляция
занимает >2 минут и потребляет ~2 ГБ RAM.

### Правила работы с кэшем

- Не выполнять `go clean -cache` без явного запроса пользователя.
- Не выполнять `go clean -modcache` без явного запроса пользователя.
- Не очищать тестовый кэш (`go clean -testcache`) без необходимости.
- Использовать существующий Go Build Cache для повторных сборок.
- На машинах с ≤2 ГБ RAM рекомендуется наличие swap.

### Допустимые команды очистки (без запроса пользователя)

- `make clean` — удаляет только бинарники и временные файлы агента.
  Не влияет на Go Build Cache.
- `make agent-clean` — удаляет temp/ и reports/ агента.
  Не влияет на Go Build Cache.
- `make clean-logs` — удаляет логи агента.
  Не влияет на Go Build Cache.
