# План: идемпотентная установка действия документа (no-op при том же значении)

## Цель

При установке action в списке документов (`POST /receipts/{id}/action`) повторный
выбор **того же** значения действия не должен менять запись в `receipt_actions`:

- не обновлять дату изменения (`action_set_at` — «дата установки действия»);
- не «старить» отправку в 1С (не сбрасывать `action_received_at` в `NULL`,
  из-за чего действие снова попадает в очередь синхронизации и передаётся в 1С повторно).

## Текущее поведение

`SetAction` в `internal/receipts/store.go:830` всегда перезаписывает запись:

- непустое значение (`store.go:842-849`) — `INSERT ... ON CONFLICT DO UPDATE SET action_set_at=now(), action_received_at=NULL` без проверки, изменилось ли значение;
- пустое значение / «Отмена» (`store.go:833-840`) — `UPDATE ... SET action='', action_set_at=now(), action_received_at=NULL` без проверки, что действие уже пустое.

Следствие: повторный выбор «Удалить» после того, как 1С уже подтвердила получение
(`action_received_at` не NULL), сбрасывает `action_received_at` в `NULL` и заново
ставит `action_set_at` — документ снова попадает в очередь `ListActionsForSync`
(`store.go:888-918`) и в фильтр по дате установки действия.

Проверено в `internal/app/receipts.go:1121` (`ReceiptActionSave`): обработчик просто
вызывает `SetAction`, никакой проверки на «то же значение» нет.

## Требуемая семантика (матрица состояний)

| Текущее значение | Новое значение | Результат |
| --- | --- | --- |
| записи нет | `Удалить` / `Изменить` | создать запись |
| `Удалить` | `Изменить` | обновить `action_set_at`, сбросить `action_received_at` |
| `Удалить` | `Удалить` | **no-op** |
| `Удалить` | `""` | отменить: `action=''`, обновить `action_set_at`, сбросить `action_received_at` |
| `""` | `""` | **no-op** |
| `""` | `Удалить` / `Изменить` | установить новое действие, `action_received_at=NULL` (в очередь) |
| записи нет | `""` | **no-op** |

## Предлагаемое изменение

Изменить только `SetAction` в `internal/receipts/store.go` (и его doc-комментарий).

### Непустое значение — добавить `WHERE` в upsert

```go
_, err := s.db.ExecContext(ctx, `
    INSERT INTO receipt_actions (receipt_id, action, action_set_at, action_received_at)
    VALUES (?, ?, ?, NULL)
    ON CONFLICT(receipt_id) DO UPDATE SET
        action = excluded.action,
        action_set_at = excluded.action_set_at,
        action_received_at = NULL
    WHERE receipt_actions.action <> excluded.action
`, id, action, now)
```

Поведение:
- строки нет → вставка (как раньше);
- значение изменилось → обновление `action`, `action_set_at=now()`, `action_received_at=NULL` (как раньше);
- значение то же → `WHERE` ложен, `DO UPDATE` не выполняется, запись не трогается.

### Пустое значение / «Отмена» — добавить `AND action <> ''`

```go
_, err := s.db.ExecContext(ctx, `
    UPDATE receipt_actions
    SET action = '', action_set_at = ?, action_received_at = NULL
    WHERE receipt_id = ? AND action <> ''
`, now, id)
```

Поведение:
- записи нет → `UPDATE` затрагивает 0 строк (как раньше);
- действие непустое → очистка и сброс `action_received_at` (как раньше);
- действие уже пустое → no-op.

Оба изменения атомарны (один SQL-запрос), без гонки read-then-write.

## Тесты

В `internal/receipts/store_test.go` добавить тест `TestStore_SetAction_SameActionIsNoOp`:

1. `SetAction(ctx, id, ActionDelete)` → установить действие.
2. `ConfirmActions(ctx, orgID, []string{uuid})` → `action_received_at` становится не NULL.
3. Считать через `GetAction` действие и `action_received_at`; отдельным `SELECT`
   прочитать точное значение `action_set_at` из `receipt_actions`.
4. Повторный `SetAction(ctx, id, ActionDelete)` (то же значение).
5. Утвердить: `GetAction` возвращает то же действие, `action_received_at` равен
   сохранённому (не NULL), а `action_set_at` **точно совпадает** со значением,
   прочитанным на шаге 3.

Проверка неизменности должна быть на **точное равенство** обоих временных
значений (`action_set_at` и `action_received_at`) со считанными до повторного
вызова. Не полагаться на паузу или разницу в миллисекундах: сравнивать
сохранённые строки, а не «не NULL».

Дополнительно проверка «Отмена»-no-op: после `SetAction(ctx, id, "")` прочитать
`action_set_at` и повторным `SetAction(ctx, id, "")` убедиться, что `action_set_at`
точно не изменился (логика `AND action <> ''`).

Существующие тесты не ломаются: `TestStore_SetAction_Upsert` меняет значение
(`ActionDelete` → `ActionChange`), поэтому обновление по-прежнему происходит.

## Документация

Обновить `docs/DATABASE.md` (раздел «Таблица ReceiptActions», правила, строки 372-380):
добавить правило о no-op при повторном выборе того же значения:

> - Повторный выбор того же действия — no-op: `action_set_at` и `action_received_at`
>   не изменяются (запись в очередь не возвращается). Аналогично повторная «Отмена»
>   уже отменённого действия ничего не делает.

`docs/integration-api.md` менять не требуется: контракт очереди (GET/PUT actions)
не меняется, меняется только то, когда запись попадает в очередь.

## Вне области / открытые вопросы

- Flash-сообщение в `ReceiptActionSave` (`receipts.go:1158-1161`) остаётся прежним
  («Действие установлено…» / «Действие отменено.»). Если нужно сообщение
  «Действие не изменилось» — отдельное уточнение у пользователя.

## Валидация

```bash
make test-container    # go test ./... -count=1
make check-container   # gofmt -l . (обнаружение) + go vet ./...
```

Проверки выполняются только внутри контейнера (см. AGENTS.md). Перед
`check-container` при необходимости `gofmt -l -w .` внутри контейнера.
