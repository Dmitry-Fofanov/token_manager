# Менеджер токенов

## Запуск проекта

Для запуска проекта выполните следующие шаги:

### 1. Клонирование репозитория

```bash
git clone https://github.com/Dmitry-Fofanov/token_manager
cd token_manager
```

### 2. Настройка окружения

Создайте файл `.env`. Можно использовать пример из `.env.example`:

```bash
cp .env.example .env
```

### 3. Запуск контейнеров

Запустите Docker Compose:

```bash
docker compose up
```

## Проверка работы API

### Получение пары токенов

При запуске сервера с `DEBUG=TRUE` будут созданы 3 тестовых пользователя, и их GUID будут выведены в лог.

Для получения пары токенов выполните запрос:

```bash
curl --request POST \
     --url 'http://localhost/tokens/get' \
     --header 'Content-Type: application/json' \
     --data '{"user_id": "GUID нужного пользователя вставить сюда"}'
```

### Обновление пары токенов

Для обновления пары токенов используйте маршрут refresh. Данные токенов заполните соответствующими, полученными в предыдущем запросе.

Пример запроса:

```bash
curl --request POST \
     --url 'http://localhost/tokens/refresh' \
     --header 'Content-Type: application/json' \
     --data '{
           "access_token": "...",
           "refresh_token": "..."
         }'
```
