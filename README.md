# AI API Proxy Platform

A SaaS platform that proxies AI model APIs (OpenAI, Claude, Gemini, Qwen) with per-token billing, credits system, and Alipay/WeChat Pay support.

## Architecture

- **Backend**: Go + Gin (API gateway + management API)
- **Frontend**: Next.js 14 + TypeScript + ShadCN/UI
- **Database**: PostgreSQL + Redis
- **Deployment**: Docker Compose

## Quick Start

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 22+](https://nodejs.org/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/)

### 1. Configure environment

```bash
cp .env.example .env
# Edit .env with your API keys and payment credentials
```

### 2. Start databases

```bash
cd deploy
docker-compose up -d postgres redis
```

### 3. Run database migrations

```bash
cd backend
# Install migrate tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
make migrate-up
```

### 4. Start backend

```bash
cd backend
go mod tidy
make run
# Server starts on :8080
```

### 5. Start frontend

```bash
cd frontend
npm install
npm run dev
# Frontend starts on :3000
```

### 6. Access the platform

- User portal: http://localhost:3000
- Admin panel: http://localhost:3000/admin/dashboard
- API endpoint: http://localhost:8080/v1/chat/completions

## Using the API

Users can call the proxy exactly like the OpenAI API:

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-your-platform-key",
    base_url="http://localhost:8080/v1"
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello!"}],
    stream=True
)
```

## Supported Models

| Provider | Models |
|----------|--------|
| OpenAI | gpt-4o, gpt-4o-mini, gpt-4-turbo, gpt-3.5-turbo |
| Anthropic | claude-3-5-sonnet-20241022, claude-3-5-haiku-20241022, claude-3-opus-20240229 |
| Google | gemini-1.5-pro, gemini-1.5-flash, gemini-2.0-flash |
| Alibaba | qwen-max, qwen-plus, qwen-turbo |

## Credit Pricing

1 Credit = 0.001 CNY (1 CNY = 1000 Credits)

| Model | Input (Credits/1K tokens) | Output (Credits/1K tokens) |
|-------|--------------------------|---------------------------|
| gpt-4o | 37 | 111 |
| claude-3-5-sonnet | 22 | 110 |
| gemini-1.5-pro | 9 | 27 |
| qwen-max | 36 | 108 |

Pricing is configurable via the admin panel at `/admin/models`.

## Payment

Supports Alipay (支付宝) and WeChat Pay (微信支付). Configure credentials in `.env`.

For sandbox testing, set `ENV=development` (Alipay sandbox mode will be enabled automatically).

## Production Deployment

```bash
cd deploy
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```
