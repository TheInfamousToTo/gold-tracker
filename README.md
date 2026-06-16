# Gold Tracker 🚀

Enterprise-grade self-hosted app for logging gold purchases, tracking spot prices, and portfolio management. Now powered by a high-performance Go backend and a modern React frontend.

## Stack

- **Backend**: Go (Gin) + PostgreSQL (`pgx`)
- **Frontend**: React + Vite + Tailwind (TypeScript ready)
- **Infrastructure**: Docker Compose, npm workspaces monorepo
- **Deployment**: Integrated with Traefik for automated TLS

## Architecture

This project is structured as a monorepo for better maintainability and open-source readiness:

- `/backend`: High-performance API written in Go.
- `/frontend`: Modern React application.
- `/migrations`: (Planned) Structured SQL migrations.

## Getting Started

### 1. Database setup

Run the setup script against your PostgreSQL instance. It creates the database, app user, tables, and views.

```bash
chmod +x setup_gold_db.sh
./setup_gold_db.sh
```

### 2. Configure environment

```bash
cp .env.example .env
# Set GOLD_DB_PASS to your database password
```

### 3. Build and run (Docker)

The easiest way to get started is using Docker Compose:

```bash
docker compose up -d --build
```

The app will be available at `https://gold.satrawi.com` (via Traefik) and locally on port `3960`.

## API reference

The API is now served by the Go backend on port `3000`.

| Method | Path | Description |
|---|---|---|
| GET | `/api/health` | DB connectivity check |
| GET | `/api/items` | List all purchases |
| POST | `/api/items` | Add a purchase |
| PUT | `/api/items/:id` | Update a purchase |
| DELETE | `/api/items/:id` | Remove a purchase |
| GET | `/api/portfolio` | Holdings + current value, gain/loss, totals |
| GET | `/api/prices` | Recent spot prices |
| POST | `/api/prices` | Add/update a spot price |
| GET | `/api/signals` | AI buy/sell/hold signal history |

## Development

### Backend (Go)
Requires Go 1.23+ or Docker.
```bash
cd backend
go run cmd/main.go
```

### Frontend (React)
```bash
cd frontend
npm install
npm run dev
```

## Contributing

We welcome contributions! Please feel free to submit a Pull Request. This project is now structured for scale and open-source collaboration.
