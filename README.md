# Task Manager — Сервис управления задачами

REST API сервис для управления задачами в командах с поддержкой ролевой модели, истории изменений и сложными SQL-запросами.

## Стек технологий

- **Go 1.25** — основной язык
- **go-chi** — HTTP-роутер
- **sqlx** — работа с MySQL
- **Redis** — кеширование
- **JWT** (`golang-jwt/jwt/v5`) — аутентификация
- **Prometheus** — метрики
- **Docker / Docker Compose** — контейнеризация
- **Testcontainers** — интеграционные тесты
- **sony/gobreaker** — circuit breaker
- **spf13/viper** — конфигурация

## Архитектура

Проект построен по принципам **Clean Architecture** и **SOLID**:

```
cmd/server/                          — точка входа, graceful shutdown
internal/
├── config/                          — загрузка конфигурации (viper)
├── domain/                          — доменные модели, интерфейсы репозиториев, ошибки
├── usecase/                         — бизнес-логика (auth, team, task)
├── repository/
│   ├── mysql/                       — реализация репозиториев (sqlx)
│   └── redis/                       — кеш-репозиторий
├── delivery/http/
│   ├── handler/                     — HTTP-обработчики
│   ├── middleware/                   — JWT-авторизация, rate limiter
│   └── router.go                    — маршрутизация
├── circuitbreaker/                  — circuit breaker для email-сервиса
└── metrics/                         — Prometheus middleware
migrations/                          — SQL-миграции
pkg/validator/                       — валидация входных данных (validator/v10)
```

## База данных

6 таблиц, 10 внешних ключей, 12 индексов:

| Таблица | Описание |
|---------|----------|
| `users` | Пользователи |
| `teams` | Команды (`created_by → users.id`) |
| `team_members` | Связь пользователь ↔ команда + роль (owner/admin/member) |
| `tasks` | Задачи (`assignee_id`, `team_id`, `created_by → users.id`) |
| `task_history` | Аудит-лог изменений задач (`task_id`, `changed_by`) |
| `task_comments` | Комментарии к задачам (`task_id`, `user_id`) |

**Внешние ключи (10):**
`teams.created_by → users.id`, `team_members.user_id → users.id`, `team_members.team_id → teams.id`, `tasks.assignee_id → users.id`, `tasks.team_id → teams.id`, `tasks.created_by → users.id`, `task_history.task_id → tasks.id`, `task_history.changed_by → users.id`, `task_comments.task_id → tasks.id`, `task_comments.user_id → users.id`

**Индексы (12):** `idx_users_email`, `idx_users_username`, `idx_teams_created_by`, `idx_tm_user_team` (UNIQUE), `idx_tm_team`, `idx_tasks_team_id`, `idx_tasks_assignee_id`, `idx_tasks_status`, `idx_tasks_team_status` (составной), `idx_tasks_created_at`, `idx_th_task_id`, `idx_tc_task_id`

## API-эндпоинты

### Публичные

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/register` | Регистрация |
| POST | `/api/v1/login` | Аутентификация (JWT) |

### Защищённые (требуется `Authorization: Bearer <token>`)

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/teams` | Создать команду (стать owner) |
| GET | `/api/v1/teams` | Список команд пользователя |
| POST | `/api/v1/teams/{id}/invite` | Пригласить в команду (только owner/admin) |
| GET | `/api/v1/teams/stats` | Статистика команд (JOIN 3+ таблиц + агрегация) |
| GET | `/api/v1/teams/top-creators?year=2026&month=3` | Топ-3 создателей задач (оконные функции) |
| POST | `/api/v1/tasks` | Создать задачу (только член команды) |
| GET | `/api/v1/tasks?team_id=1&status=todo&assignee_id=5&cursor=10&limit=20` | Фильтрация + курсорная пагинация |
| PUT | `/api/v1/tasks/{id}` | Обновить задачу (проверка прав + аудит-лог) |
| GET | `/api/v1/tasks/{id}/history` | История изменений задачи |
| GET | `/api/v1/tasks/orphaned` | Задачи с невалидным assignee (NOT EXISTS) |
| POST | `/api/v1/tasks/{id}/comments` | Добавить комментарий |
| GET | `/api/v1/tasks/{id}/comments` | Список комментариев |
| GET | `/metrics` | Prometheus-метрики |
| GET | `/health` | Health check |

## Сложные SQL-запросы

### 1. JOIN 3+ таблиц + агрегация
Для каждой команды: название, количество участников, количество завершённых задач за последние 7 дней.
```sql
SELECT t.id, t.name, COUNT(DISTINCT tm.user_id), COUNT(DISTINCT CASE WHEN tk.status='done' AND tk.updated_at >= DATE_SUB(NOW(), INTERVAL 7 DAY) THEN tk.id END)
FROM teams t LEFT JOIN team_members tm ... LEFT JOIN tasks tk ... GROUP BY t.id, t.name
```

### 2. Оконные функции (ROW_NUMBER)
Топ-3 пользователей по количеству созданных задач в каждой команде за указанный месяц.
```sql
SELECT ... ROW_NUMBER() OVER (PARTITION BY t.id ORDER BY COUNT(tk.id) DESC) AS rnk
FROM tasks tk INNER JOIN teams t ... INNER JOIN users u ...
WHERE YEAR(tk.created_at) = ? AND MONTH(tk.created_at) = ?
... WHERE rnk <= 3
```

### 3. NOT EXISTS — проверка целостности
Задачи, где assignee не является членом команды этой задачи.
```sql
SELECT tk.id, tk.title, u.username, t.name
FROM tasks tk ... WHERE tk.assignee_id IS NOT NULL
AND NOT EXISTS (SELECT 1 FROM team_members tm WHERE tm.user_id = tk.assignee_id AND tm.team_id = tk.team_id)
```

## Запуск

### Docker Compose (рекомендуется)

```bash
docker-compose up --build
```

Сервис будет доступен на `http://localhost:8080`.

### Локально

1. Убедитесь, что MySQL и Redis запущены.
2. Скопируйте и настройте конфигурацию:
   ```bash
   cp config.yaml config.local.yaml
   ```
   Отредактируйте DSN и адрес Redis в `config.local.yaml`.
3. Запустите:
   ```bash
   CONFIG_PATH=config.local.yaml go run ./cmd/server
   ```

## Конфигурация

Файл `config.yaml` (загружается через `spf13/viper` с поддержкой переменных окружения):

```yaml
server:
  port: ":8080"
  shutdown_timeout: 15s

database:
  dsn: "root:password@tcp(mysql:3306)/taskmanager?parseTime=true&multiStatements=true"
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime: 5m

redis:
  addr: "redis:6379"

jwt:
  secret: "your-secret-key"
  expiration: 24h
```

## Тестирование

### Unit-тесты (55+ тестов)
```bash
go test ./internal/usecase/... -v
```

### Интеграционные тесты с testcontainers (8 тестов, требуется Docker)
```bash
go test ./internal/repository/mysql/... -v
```

### Покрытие
```bash
go test ./internal/usecase/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Текущее покрытие use cases: **100%**.

## Оптимизация

| Оптимизация | Реализация |
|-------------|------------|
| **Redis-кеширование** | Список задач команды с TTL 5 мин, инвалидация при create/update |
| **Индексы MySQL** | 12 индексов, включая составной `(team_id, status)` |
| **Connection Pooling** | `MaxOpenConns=25`, `MaxIdleConns=10`, `ConnMaxLifetime=5m` |
| **Курсорная пагинация** | По ID (`id < cursor`) для стабильной пагинации без дубликатов |

## Дополнительные возможности

| Возможность | Реализация |
|-------------|------------|
| **Circuit Breaker** | `sony/gobreaker` v2 для email-сервиса (5 ошибок → размыкание, 10с таймаут) |
| **Rate Limiting** | 100 запросов/мин на пользователя (token bucket алгоритм) |
| **Graceful Shutdown** | Корректное завершение по SIGINT/SIGTERM с настраиваемым таймаутом (15с) |
| **Prometheus-метрики** | `http_requests_total`, `http_request_duration_seconds`, `http_errors_total` |
| **Конфигурация** | YAML-файл + переменные окружения через `spf13/viper` |

## Docker

- **Dockerfile** — мультистейдж сборка (`golang:1.25-alpine` → `alpine:3.19`)
- **docker-compose.yaml** — 4 сервиса: Go-приложение, MySQL 8.0, Redis 7, Prometheus
