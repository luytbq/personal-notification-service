# Notification App

A personal notification service that receives notifications via HTTPS and delivers them through various channels (Telegram, Email).

## Features

- **HTTP API** for receiving notifications
- **Queue-based processing** with Redis + Asynq
- **Multiple channels** (Telegram implemented, Email scaffolded)
- **Rate limiting** (token bucket, per API key + channel)
- **Retry with exponential backoff** (5 retries)
- **Dead letter queue** for failed notifications (logged)
- **Graceful shutdown** (waits for in-flight tasks)
- **Structured JSON logging**

## Architecture

```
Automation Tool → API Server → Redis Queue → Worker → Telegram
                      ↓
                Rate Limiter
                      ↓
                Dead Letter Queue
```

## Quick Start

### Prerequisites

- Docker and Docker Compose
- A Telegram Bot (see setup below)

### 1. Create a Telegram Bot

1. Open Telegram and search for `@BotFather`
2. Send `/newbot` and follow the prompts
3. Copy the **bot token** (looks like `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`)
4. Start a chat with your new bot and send any message
5. Get your **chat ID**:
   ```bash
   curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates" | jq '.result[0].message.chat.id'
   ```
   Or visit: `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates` in your browser

### 2. Configure Environment

```bash
cp .env.example .env
```

Edit `.env` with your values:

```bash6577114265:AAGelRxsh0tz1YXCx0Ms6pa5fMC0oEVqhLY
# Generate a secure API key
API_KEYS=$(openssl rand -hex 32)

# Your Telegram bot token
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz

# Your Telegram chat ID
TELEGRAM_CHAT_ID=123456789
```

### 3. Run with Docker Compose

```bash
docker-compose up -d
```

### 4. Test the API

```bash
# Health check
curl http://localhost:8272/notify/health

# Send a notification
curl -X POST http://localhost:8272/notify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "title": "Test Notification",
    "message": "Hello from the notification app!",
    "level": "info",
    "channel": ["telegram"]
  }'
```

## API Reference

### POST /notify

Send a notification.

**Headers:**
- `Content-Type: application/json`
- `X-API-Key: <your-api-key>` (required)

**Request Body:**

```json
{
  "title": "Backup failed",
  "message": "Disk space full on VPS-01",
  "level": "error",
  "channel": ["telegram"]
}
```

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Notification title |
| `message` | string | Yes | Notification message body |
| `level` | string | Yes | One of: `info`, `warning`, `error`, `critical` |
| `channel` | array | Yes | List of channels: `telegram`, `email` |

**Response (202 Accepted):**

```json
{
  "status": "queued",
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Error Responses:**

| Status | Description |
|--------|-------------|
| 400 | Invalid request body or validation error |
| 401 | Missing or invalid API key |
| 429 | Rate limit exceeded |
| 500 | Internal server error |

### GET /health

Health check endpoint.

**Response (200 OK):**

```json
{
  "status": "ok"
}
```

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8272` | HTTP server port |
| `LOG_LEVEL` | `info` | Log level |
| `API_KEYS` | (required) | Comma-separated list of valid API keys |
| `RATE_LIMIT_PER_MINUTE` | `60` | Rate limit per API key per channel |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDIS_DB` | `0` | Redis database number |
| `WORKER_CONCURRENCY` | `10` | Number of concurrent workers |
| `MAX_RETRIES` | `5` | Maximum retry attempts |
| `TELEGRAM_BOT_TOKEN` | (required) | Telegram bot token |
| `TELEGRAM_CHAT_ID` | (required) | Telegram chat ID |
| `SHUTDOWN_TIMEOUT_SECONDS` | `30` | Graceful shutdown timeout |

## Notification Levels

Messages are formatted with a level prefix:

| Level | Prefix |
|-------|--------|
| `info` | `[INFO]` |
| `warning` | `[WARNING]` |
| `error` | `[ERROR]` |
| `critical` | `[CRITICAL]` |

Example Telegram message:
```
[ERROR] Backup failed

Disk space full on VPS-01
```

## Rate Limiting

- Token bucket algorithm
- 60 requests per minute per API key per channel
- Returns `429 Too Many Requests` when exceeded

## Retry Policy

- Maximum 5 retries
- Exponential backoff: 10s, 20s, 40s, 80s, 160s
- Failed notifications after max retries are logged (dead letter queue)

## Logging

Structured JSON logs to stdout:

```json
{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"notification sent","notification_id":"uuid","channel":"telegram","status":"sent","latency":"150ms"}
```

## Development

### Local Setup

```bash
# Install dependencies
go mod download

# Run Redis locally
docker run -d -p 6379:6379 redis:7-alpine

# Set environment variables
export API_KEYS=dev-key
export TELEGRAM_BOT_TOKEN=your-token
export TELEGRAM_CHAT_ID=your-chat-id

# Run the application
go run ./cmd/server
```

### Project Structure

```
notification-app/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── api/
│   │   ├── handler.go           # HTTP handlers
│   │   ├── middleware.go        # Auth & rate limiting
│   │   └── router.go            # Route setup
│   ├── config/
│   │   └── config.go            # Configuration
│   ├── notification/
│   │   ├── types.go             # Types & levels
│   │   └── validator.go         # Validation
│   ├── queue/
│   │   ├── client.go            # Queue client
│   │   ├── tasks.go             # Task definitions
│   │   └── worker.go            # Worker
│   ├── ratelimit/
│   │   └── limiter.go           # Rate limiter
│   └── channels/
│       ├── channel.go           # Channel interface
│       ├── telegram.go          # Telegram
│       └── email.go             # Email (scaffolded)
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## Adding New Channels

1. Create a new file in `internal/channels/` (e.g., `slack.go`)
2. Implement the `Channel` interface:

```go
type Channel interface {
    Name() notification.Channel
    Send(ctx context.Context, n *notification.Notification) error
}
```

3. Register the channel in `cmd/server/main.go`:

```go
registry.Register(channels.NewSlackChannel(...))
```

4. Add the channel to `ValidChannels` in `internal/notification/types.go`

## License

MIT
