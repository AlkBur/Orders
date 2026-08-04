# Acceptance Tests

Сценарии запускаются против `ORDERS_CONFIG=config.agent.json`.

Порядок запуска:

```bash
make build-agent
make run-agent
# ... curl-сценарии ...
make agent-clean
```

---

## Authentication

- [ ] admin / test-bootstrap → `/set-password`
- [ ] admin / wrong → "Invalid login or password"
- [ ] Set Password → `/login`
- [ ] admin / test-bootstrap после установки пароля → "Invalid login or password"
- [ ] admin / newpassword → `/orders`
- [ ] Logout
- [ ] Неверный пароль → ошибка + NoCache
- [ ] F5 после ошибки → чистая форма (без ошибки)
- [ ] Защищённая страница без Login → `/login`

## Sessions

- [ ] Session создаётся только после успешного логина
- [ ] Cookie удаляется после Logout
- [ ] Просроченная сессия → `/login`

## Rate Limiting (login)

Дефолтные лимиты из config: по IP 10/60, по аккаунту 5/600.

- [ ] FullPage: 11 быстрых POST /login с одного IP → последний = 429 + форма с alert
- [ ] Fragment: 11 быстрых POST /login с `HX-Request: true` → последний = 429 + JSON `{"kind":"infrastructure","errors":[...]}`
- [ ] Превышение лимита по IP не блокирует другие аккаунты с того же IP
- [ ] Превышение лимита одного аккаунта не блокирует другие аккаунты
- [ ] Пустой login не создаёт общий bucket: IP-лимит работает независимо
- [ ] Запрос без X-Forwarded-For → лимит считается по RemoteAddr (fallback)
- [ ] Запрос со спуфингом X-Forwarded-For напрямую (не через доверенный прокси) → лимит считается по RemoteAddr

## Request Guard (login)

- [ ] Тело > 4 KB → 413 (Fragment: 413 + JSON; FullPage: 413 + alert)
- [ ] Лишнее поле → 400
- [ ] login длиннее 64 → 400
- [ ] password длиннее 128 → 400
- [ ] Guard-ответы не расходуют лимит попыток (400/413 не считаются в счётчик)

## Flash

- [ ] PRG: ошибка → 303 → GET с Flash
- [ ] F5 после Flash → Flash очищен
