# Acceptance Tests

Сценарии запускаются против `ORDERS_CONFIG=config.agent.json`.

Порядок запуска:

```bash
make build-agent
make run-agent  # в отдельном терминале
# ... curl-сценарии ...
make agent-clean
```

---

## Authentication

- [ ] Login без пароля → `/set-password`
- [ ] Установка пароля → `/orders`
- [ ] Logout
- [ ] Повторный Login с паролем → `/orders`
- [ ] Неверный пароль → ошибка + NoCache
- [ ] F5 после ошибки → чистая форма (без ошибки)
- [ ] Без пароля → защищённые страницы блокированы
- [ ] После пароля `/set-password` недоступен

## Sessions

- [ ] Session создаётся только после успешного логина
- [ ] Cookie удаляется после Logout
- [ ] Просроченная сессия → `/login`

## Flash

- [ ] PRG: ошибка → 303 → GET с Flash
- [ ] F5 после Flash → Flash очищен
