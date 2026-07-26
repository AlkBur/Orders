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

## Flash

- [ ] PRG: ошибка → 303 → GET с Flash
- [ ] F5 после Flash → Flash очищен
