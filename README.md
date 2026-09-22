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

### 2. Write the config file

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` with your values. Generate the API key rather than inventing one:

```bash
openssl rand -hex 32
```

At minimum set `api_keys`, `telegram.bot_token` and `telegram.chat_id`.

### 3. Run with Docker Compose

Compose reads a separate config so the container can use service names instead of
`localhost`:

```bash
cp config.yaml config.docker.yaml
```

In `config.docker.yaml` set `server.host` to `0.0.0.0` and `redis.addr` to
`redis:6379`, then put the Redis password in `.env` under `REDIS_PASSWORD` so the
Redis container and the app agree on it:

```bash
echo "REDIS_PASSWORD=$(openssl rand -base64 32)" >> .env
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
    "channel": ["telegram"],
    "source": "my-automation-script"
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
  "channel": ["telegram"],
  "source": "backup-script"
}
```

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Notification title |
| `message` | string | Yes | Notification message body |
| `level` | string | Yes | One of: `info`, `warning`, `error`, `critical` |
| `channel` | array | Yes | List of channels: `telegram`, `email` |
| `source` | string | No | Source identifier (e.g., script name, service name) |

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

All configuration lives in a single YAML file. The path comes from the `PNS_CONFIG`
environment variable and defaults to `config.yaml` in the working directory. No other
environment variable is read. See `config.yaml.example` for the full file.

| Key | Default | Description |
|-----|---------|-------------|
| `server.host` | `127.0.0.1` | Bind address. Use `0.0.0.0` only when a container runtime publishes the port |
| `server.port` | `8272` | HTTP server port |
| `server.log_level` | `info` | Log level |
| `server.shutdown_timeout_seconds` | `30` | Graceful shutdown timeout |
| `api_keys` | (required) | List of valid API keys |
| `rate_limit_per_minute` | `60` | Rate limit per API key per channel |
| `redis.addr` | `localhost:6379` | Redis address |
| `redis.password` | (empty) | Redis password |
| `redis.db` | `0` | Redis database number |
| `redis.key_prefix` | `pns` | Prefix for queue and task names |
| `worker.concurrency` | `10` | Number of concurrent workers |
| `worker.max_retries` | `5` | Maximum retry attempts |
| `telegram.bot_token` | (required) | Telegram bot token |
| `telegram.chat_id` | (required) | Telegram chat ID |
| `webhooks` | (empty) | Webhook targets, addressed as channel `webhook:<name>` |

`config.yaml` holds live credentials and is gitignored. Keep it that way.

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

Source: backup-script
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

# Run Redis locally, on loopback only
docker run -d -p 127.0.0.1:6379:6379 redis:8.4-alpine

# Write your config
cp config.yaml.example config.yaml

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
├── config.yaml.example
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
