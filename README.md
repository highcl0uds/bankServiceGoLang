# bankServiceGoLang

REST API банковского сервиса на Go. Поддерживает управление счетами, виртуальными картами, переводами, кредитами и финансовой аналитикой.

## Стек технологий

- **Маршрутизация:** `gorilla/mux`
- **База данных:** PostgreSQL + `lib/pq`
- **Аутентификация:** JWT — `golang-jwt/jwt/v5`
- **Логирование:** `logrus`
- **Шифрование карт:** `ProtonMail/go-crypto` (PGP)
- **Хеширование:** `bcrypt`, HMAC-SHA256
- **Email-уведомления:** `gomail.v2`
- **Парсинг XML (ЦБ РФ):** `beevik/etree`

## Требования

- Go 1.23+
- PostgreSQL 17

## Быстрый старт

### 1. Установка зависимостей

```bash
go mod download
```

### 2. База данных

```bash
psql -U postgres -c "CREATE DATABASE bank_db;"
psql -U postgres -d bank_db -f db/migrations/001_init.sql
```

### 3. Генерация PGP-ключей

Для шифрования данных карт нужна пара PGP-ключей. Встроенная утилита (не требует gpg):

```bash
go run ./cmd/keygen/main.go
```

Утилита выведет готовые строки `PGP_PUBLIC_KEY`, `PGP_PRIVATE_KEY`, `JWT_SECRET` и `HMAC_SECRET` — скопируйте их в `.env`.

Альтернатива через gpg (Linux/macOS):

```bash
gpg --batch --gen-key <<EOF
%no-protection
Key-Type: RSA
Key-Length: 2048
Name-Real: BankService
Name-Email: bank@example.com
Expire-Date: 0
EOF

gpg --armor --export bank@example.com
gpg --armor --export-secret-keys bank@example.com
```

### 4. Переменные окружения

Скопируйте `.env.example` в `.env` и заполните значения. Файл загружается автоматически при запуске.

Обязательные: `DB_PASSWORD`, `JWT_SECRET` (мин. 32 символа), `PGP_PUBLIC_KEY`, `PGP_PRIVATE_KEY`, `HMAC_SECRET`. Остальные имеют дефолтные значения. SMTP необязателен — если `SMTP_HOST` не задан, уведомления пропускаются без ошибок.

### 5. Запуск

```bash
go run ./cmd/server/main.go
```

Сервер запустится на `http://localhost:8080`.

---

## API

### Публичные эндпоинты

`POST /register` — регистрация. Требования: username 3–64 символа (буквы, цифры, `_`, `-`), валидный email, пароль ≥ 8 символов.

`POST /login` — аутентификация. Возвращает JWT-токен, действительный 24 часа.

Токен передаётся в заголовке `Authorization: Bearer <token>` для всех защищённых запросов.

### Защищённые эндпоинты

**Счета**

- `POST /accounts` — создать счёт (валюта RUB)
- `GET /accounts` — список своих счетов
- `GET /accounts/{accountId}` — информация о счёте
- `POST /accounts/{accountId}/deposit` — пополнить счёт, тело: `{"amount":"5000.00"}`
- `POST /accounts/{accountId}/withdraw` — снять средства, тело: `{"amount":"1000.00"}`

**Карты**

- `POST /cards` — выпустить виртуальную карту, тело: `{"account_id":"<id>"}`. Номер генерируется по алгоритму Луна, хранится в зашифрованном виде (PGP + HMAC), CVV хешируется bcrypt.
- `GET /cards` — список своих карт с расшифрованными данными

**Переводы**

- `POST /transfer` — перевод между счетами, тело: `{"from_account_id":"<id>","to_account_id":"<id>","amount":"1000.00"}`

**Кредиты**

- `POST /credits` — оформить кредит, тело: `{"account_id":"<id>","principal":"100000.00","term_months":12}`. Ставка: ключевая ставка ЦБ РФ + 5%, расчёт по аннуитетной формуле. При недоступности ЦБ РФ используется запасная ставка 21%.
- `GET /credits/{creditId}/schedule` — график платежей

**Аналитика**

- `GET /analytics` — доходы и расходы за текущий месяц, кредитная нагрузка
- `GET /accounts/{accountId}/predict?days=30` — прогноз баланса на N дней (макс. 365)

---

## Безопасность

Пароли хешируются bcrypt. Номер и срок карты шифруются PGP, CVV хешируется bcrypt, целостность номера проверяется HMAC-SHA256 при каждом чтении. Доступ к API защищён JWT HS256 с TTL 24 часа. Генерация номеров карт и CVV использует `crypto/rand`.

---

## Фоновые процессы

Шедулер запускается вместе с сервером и каждые 12 часов обрабатывает просроченные платежи по кредитам. Если средств достаточно — списывает платёж и отправляет уведомление. Если нет — помечает платёж как просроченный, начисляет штраф +10% и повторно обрабатывает при следующем запуске.

---

## Структура проекта

```
.
├── cmd/
│   ├── server/main.go          # Точка входа
│   └── keygen/main.go          # Генерация PGP-ключей
├── config/                     # Конфигурация из env
├── db/
│   ├── db.go                   # Подключение к PostgreSQL
│   └── migrations/001_init.sql # DDL: таблицы, индексы, enum-типы
├── internal/
│   ├── handler/                # HTTP-обработчики
│   ├── middleware/             # JWT-аутентификация, логирование
│   ├── models/                 # Структуры данных + валидация
│   ├── repository/             # SQL-запросы
│   └── service/                # Бизнес-логика
└── pkg/
    ├── apperrors/              # Sentinel-ошибки
    └── crypto/                 # Luhn, HMAC, PGP
```

---

## Проверка сборки

```bash
go build ./...
```

## Проверка работоспособности

```powershell
$BASE = "http://localhost:8080"

# 1. Регистрация
Invoke-RestMethod -Uri "$BASE/register" -Method Post -ContentType "application/json" -Body '{"username":"testuser","email":"test@example.com","password":"password123"}'

# 2. Логин — сохраняем токен
$TOKEN = (Invoke-RestMethod -Uri "$BASE/login" -Method Post -ContentType "application/json" -Body '{"email":"test@example.com","password":"password123"}').token
$h = @{ Authorization = "Bearer $TOKEN" }

# 3. Создать счёт
$ACC = (Invoke-RestMethod -Uri "$BASE/accounts" -Method Post -Headers $h).id

# 4. Пополнить счёт
Invoke-RestMethod -Uri "$BASE/accounts/$ACC/deposit" -Method Post -Headers $h -ContentType "application/json" -Body '{"amount":"50000.00"}'

# 5. Выпустить карту
Invoke-RestMethod -Uri "$BASE/cards" -Method Post -Headers $h -ContentType "application/json" -Body "{`"account_id`":`"$ACC`"}"

# 6. Оформить кредит
$CREDIT = (Invoke-RestMethod -Uri "$BASE/credits" -Method Post -Headers $h -ContentType "application/json" -Body "{`"account_id`":`"$ACC`",`"principal`":`"20000.00`",`"term_months`":6}").credit.id

# 7. График платежей
Invoke-RestMethod -Uri "$BASE/credits/$CREDIT/schedule" -Method Get -Headers $h

# 8. Аналитика
Invoke-RestMethod -Uri "$BASE/analytics" -Method Get -Headers $h
```