# simple_bot

Телеграмм бот выполняет простые вычисленя. Не использует готовых пакетов для работы с telegram api (по этой причине его легко можно перевести на vk api).

## Установка

1. Скачать репозиторий
2. Создать телеграмм бота у [botfather](https://t.me/BotFather). По итогу он выдаст токен.
3. Поместить токен телеграмм бота в переменную `BotToken`.

## Запуск

1. Запустить локально [ngrok](https://ngrok.com/) командой `ngrok http 8081`. В консоле будет указан `url`.
2. Поместить `url` в переменную `WebhookURL`.
3. Запустить бота командой `go run ./bot.go`
