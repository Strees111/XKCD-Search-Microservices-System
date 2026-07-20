# XKCD Search Microservices System

Данный проект представляет собой полноценную распределенную систему микросервисов для индексации, поиска и ранжирования комиксов XKCD. Проект построен на базе **гексагональной архитектуры (Ports & Adapters)** и демонстрирует взаимодействие сервисов через **REST** и **gRPC**, использование событийно-ориентированной архитектуры (Event-Driven) и паттернов отказоустойчивости.

## Планы на будущее (Roadmap)
Проект активно развивается. В ближайших обновлениях запланировано:
- [ ] **Дополнительное покрытие тестами:** расширение unit- и интеграционных тестов для повышения надежности.
- [X] **Web-Frontend:** создание небольшого пользовательского интерфейса (UI) для удобного поиска комиксов через браузер.
- [X] **Kubernetes:** написание манифестов (Deployment, Service, Ingress и др.) и развертывание всей системы в кластере K8s.

---

## Архитектура и микросервисы

<img width="1247" height="688" alt="изображение" src="https://github.com/user-attachments/assets/a4863949-3b8b-479d-b9bd-c9b4487888d9" />

Система состоит из нескольких независимых сервисов:


1. **API Gateway (REST)** — единая точка входа. Маршрутизирует REST-запросы к внутренним gRPC-сервисам, обрабатывает авторизацию (JWT), собирает метрики и ограничивает нагрузку (Rate/Concurrency limits).
2. **Search Service (gRPC)** — ядро поисковой системы. Ищет комиксы в БД (`/search`) и по in-memory индексу (`/isearch`).
3. **Update Service** — отвечает за обновление базы данных комиксов. При обновлении публикует события в брокер сообщений.
4. **Words Normalizer (gRPC)** — сервис нормализации слов (отсеивание стоп-слов, стемминг) на базе библиотеки Snowball. Возвращает ошибки `ResourceExhausted` при превышении лимита в 4 KiB.
5. **Hello & Fileserver (REST)** — базовые сервисы для отдачи приветствий и идеоматичного CRUD-хранилища файлов (с поддержкой multipart загрузки).
6. **Petname Generator (gRPC)** — сервис генерации случайных имён (демонстрирует потоковую передачу gRPC).

---

## Ключевые возможности и технологии

- **Golang 1.25+**: Использование возможностей стандартной библиотеки (вкл. новый роутер `http.ServeMux` из 1.22+) и структурированного логирования (`slog`).
- **Архитектура**: Ports & Adapters (гексагональная архитектура), принцип инверсии зависимостей.
- **Взаимодействие**: 
  - REST API (Gateway, Fileserver).
  - gRPC (внутреннее общение между Gateway, Search, Words, Petname).
  - **NATS** (шина событий): Асинхронное уведомление сервиса `search` об обновлениях в БД для моментальной перестройки in-memory индекса.
- **Управление трафиком (Middlewares)**: 
  - *Concurrency Limiter* (ограничение одновременных подключений, возврат `503 Service Unavailable`).
  - *Rate Limiter* (ограничение RPS, задержка соединений).
- **Безопасность (AAA)**: Защита критических эндпоинтов обновления/удаления с помощью **JWT-токенов** (валидация через middleware).
- **Мониторинг**: Интеграция с **VictoriaMetrics** и **Grafana**. Сбор гистограмм времени выполнения запросов (`http_request_duration_seconds`).
- **Конфигурация**: Пакет `cleanenv` (поддержка `config.yaml` и переменных окружения, таких как `HTTP_SERVER_ADDRESS`, `BROKER_ADDRESS`, `INDEX_TTL` и др.).

---

## Запуск проекта и тестирование

### Предварительные требования
- Docker и Docker Compose
- Утилиты командной строки: `curl`, `grpcurl`
- Утилиты сборки: `make`

### Запуск через Docker Compose
Сервисы полностью готовы к запуску через предоставленный Compose-файл.

```bash
# Установка необходимых инструментов для локального тестирования (в т.ч. bombardier)
make tools

# Запуск всех сервисов, базы данных PostgreSQL, NATS, VictoriaMetrics и Grafana
make up
```
---

## API Reference (Примеры запросов)

### 1. API Gateway & Search
**Авторизация (Получение JWT):**
```bash
TOKEN=$(curl -s -X POST -d '{"name": "admin", "password": "password"}' localhost:28080/api/login)
```

**Поиск с использованием In-Memory индекса:**
```bash
curl -H "Authorization: Token $TOKEN" 'localhost:28080/api/isearch?phrase=linux,forever&limit=2'
```

*Пример ответа:*
```json
{
  "comics": [
    {"id": 196, "url": "https://imgs.xkcd.com/comics/command_line_fu.png"}
  ],
  "total": 1
}
```

**Healthcheck / Ping:**
```bash
curl -v localhost:28080/api/ping
```

### 2. Fileserver (CRUD)
```bash
# Загрузка файла
curl -v -X POST -F file=@file1.txt localhost:28081/files

# Листинг файлов
curl -v localhost:28081/files

# Получение файла
curl -v localhost:28080/files/file1.txt

# Удаление файла
curl -v -X DELETE localhost:28081/files/file1.txt
```

### 3. gRPC сервисы (через grpcurl)
```bash
# Проверка доступных методов
grpcurl -plaintext localhost:28081 list petname.PetnameGenerator

# Генерация имени (унарный запрос)
grpcurl -plaintext -d '{"words": 3, "separator": " "}' localhost:28081 petname.PetnameGenerator.Generate
```

---

## Мониторинг и Нагрузочное тестирование

### Grafana & VictoriaMetrics
Система поставляется с настроенным стеком мониторинга.
1. Откройте Grafana: `http://localhost:3000`
2. **Логин / Пароль:** `admin` / `админ`
3. Установите плагин `VictoriaMetrics` (если не установлен) и добавьте Datasource: `http://victoriametrics:8428`.
4. Импортируйте Dashboard из папки `metrics` (Dashboards -> New -> Import -> Upload JSON).

### Нагрузочное тестирование (Bombardier)
Вы можете протестировать эффективность кэширования и Rate Limiting с помощью утилиты `bombardier`:

```bash
# Тест In-Memory индекса (потребуется предварительно полученный JWT)
bombardier -H "Authorization: Token $TOKEN" 'localhost:28080/api/isearch?phrase=linux'
```

---

## Переменные окружения (Окружение)
Основные переменные, используемые для настройки системы в `compose.yaml`:

1. `ADMIN_USER` / `ADMIN_PASSWORD` | Креды суперпользователя для получения JWT. |
2. `TOKEN_TTL` | Время жизни JWT токена (по умолчанию 2 минуты). |
3. `SEARCH_CONCURRENCY` | Лимит одновременных запросов в БД (возвращает 503). |
4. `SEARCH_RATE` | Лимит запросов в секунду (RPS) для in-memory поиска. |
5. `INDEX_TTL` | Время актуальности индекса (в новой версии индекс инвалидируется через NATS, рекомендуется ставить `24h`). |
6. `BROKER_ADDRESS` | Адрес NATS сервера. |
7. `DB_ADDRESS` | Адрес PostgreSQL. |
8. `*_ADDRESS` | Адреса маршрутизации для gRPC клиентов (например, `SEARCH_ADDRESS`, `WORDS_ADDRESS`). |
## Метрики
<img width="1917" height="1039" alt="изображение" src="https://github.com/user-attachments/assets/03e15184-8496-4fd2-b18d-908b2856cc64" />
