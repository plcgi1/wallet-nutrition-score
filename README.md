# Wallet Nutrition Score

## Описание

Wallet Nutrition Score — это сервис для анализа безопасности кошельков блокчейна, который предоставляет оценку "здоровья" кошелька на основе различных параметров и рисков.

## Структура проекта

```
.
├── cmd/               # Входные точки приложения
│   └── app/           # Основное приложение
├── config/            # Конфигурационные файлы
├── docs/              # Документация для Swagger - генерится
├── documentation/     # Документация
├── internal/          # Внутренние пакеты
│   ├── aggregator/    # Агрегатор проверок
│   ├── checker/       # Проверки и фабрика
│   │   └── internal/
│   │       └── checks/# Реализации проверок
│   ├── entity/        # Общие структуры данных
│   └── provider/      # Клиенты для внешних API
├── pkg/               # Общие утилиты
│   └── logger/        # Логирование
|   └── utils/         # Утилиты
└── plans/             # Планы разработки
```

## Установка

1. Установите Go версии 1.22 или выше
2. Склонируйте репозиторий
3. Перейдите в директорию проекта
4. Установите зависимости:
   ```bash
   go mod tidy
   ```

## Конфигурация

1. Скопируйте файл `.env.example` в `.env`
2. Заполните переменные окружения в файле `.env`

## Запуск

### Локально

```bash
go run cmd/app/main.go
```

Сервер будет доступен по адресу `http://localhost:8080`.

### Docker Compose

```bash
docker-compose up -d
```

Сервер будет доступен по адресу `http://localhost:8080`.

### Docker (по отдельности)

```bash
docker build -t wallet-nutrition-score .
docker run -p 8080:8080 --env-file .env --link redis:redis wallet-nutrition-score
```

## API Endpoints

### Проверка статуса сервиса

```http
GET /health
```

Возвращает статус сервиса.

**Ответ:**
```json
{
  "status": "healthy"
}
```

### Анализ кошелька

```http
POST /api/check
Content-Type: application/json
```

Параметры запроса:
```json
{
  "address": "0x742d35Cc6634C0532925a3b88650D7241EfF5cbc"
}
```

**Ответ:**
```json
{
  "address": "0x742d35Cc6634C0532925a3b88650D7241EfF5cbc",
  "score": 100,
  "checks": [
    {
      "check_name": "assets",
      "risk_found": false,
      "risk_level": "MEDIUM",
      "score_penalty": 0,
      "details": "Stable assets: 13.5%, volatile assets: 86.5%",
      "raw_data": [...]
    },
    {
      "check_name": "dead_nft",
      "risk_found": false,
      "risk_level": "LOW",
      "score_penalty": 0,
      "details": "No dead NFTs found",
      "raw_data": null
    },
    ...
  ],
  "errors": [
    "failed to get token approvals: GoPlus API error: "
  ]
}
```

## Проверки

В текущей версии сервиса реализованы следующие проверки:

### 1. Аппрувы (approvals)
- Проверяет активные approvals на токены
- Анализирует экспозицию риска и наличие злоумышленных спендеров
- Использует API GoPlus для получения данных о approvals

### 2. Ассеты (assets)
- Анализирует состав активов на кошельке
- Рассчитывает соотношение стабильных и волатильных токенов
- При наличии >90% волатильных активов возвращает высокий риск

### 3. Скам-токены (scam_tokens)
- Проверяет токены на кошельке на наличие признаков скам
- Анализирует токены на кошельке на риск rug pull
- Использует Alchemy API для получения списка токенов на кошельке
- Использует GoPlus API для анализа безопасности токенов (поиск черных списков, фейковых токенов, ханипотов)

### 4. Мертвые NFT (dead_nft)
- Проверяет наличие NFT, которые могут быть рискованными или мертвыми
- Использует API GoPlus для проверки безопасности NFT


## Логирование

Логирование настроено с помощью logrus и выводится в формате JSON. Уровень логирования можно настроить в файле `config/config.yaml`.

## Лицензия

MIT
