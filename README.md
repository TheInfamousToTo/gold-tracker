# Gold Tracker 🚀

An enterprise-grade, self-hosted application for logging gold purchases, tracking spot prices, and managing your physical gold portfolio. Built for reliability, speed, and modern observability.

## 🌟 Key Features

- **Portfolio Management:** Log purchases with detailed metadata (karat, weight, vendor, notes).
- **Live Price Tracking:** Monitor current spot prices for 24K, 22K, 21K, and 18K gold.
- **Profit/Loss Analysis:** Real-time calculation of gain/loss based on current market data.
- **AI Recommendations:** Buy, sell, or hold calls generated from your own price history and holdings.
- **n8n Integration:** Designed to work with n8n for automated price feeds.
- **Grafana Ready:** Schema optimized for Grafana dashboards and long-term history tracking.

## 🛠️ Technology Stack

- **Backend:** [Go](https://go.dev/) (Gin Gonic) - High-performance, type-safe API.
- **Frontend:** [React](https://reactjs.org/) (Vite + Tailwind CSS) - Fast, responsive, and modern UI.
- **Database:** [PostgreSQL](https://www.postgresql.org/) - Robust relational storage.
- **Infrastructure:** [Docker](https://www.docker.com/) & [GitHub Actions](https://github.com/features/actions) - Containerized deployment and automated CI/CD.

## 🏗️ How it Works

The system consists of three main components working in harmony:

1.  **The API (Go):** Handles all business logic, Karat conversions, and portfolio calculations. It communicates directly with the PostgreSQL database.
2.  **The UI (React):** A modern dashboard that fetches data from the API and provides a clean interface for adding and tracking items.
3.  **The Ecosystem:**
    -   **Price Feeds (External):** An n8n workflow typically fetches live spot prices from external APIs and pushes them into the `gold_prices` table.
    -   **AI Signals:** The API generates recommendations itself and stores them in `signals_log`. An n8n workflow can also write there directly.
    -   **Dashboards:** Use Grafana to visualize your portfolio growth over time by reading directly from the `v_portfolio_summary` view.

## 🚀 Getting Started

### 1. Database Setup
Ensure you have a PostgreSQL instance running. Use the provided setup script to initialize the schema, including all necessary tables and the portfolio summary view.

```bash
chmod +x setup_gold_db.sh
./setup_gold_db.sh
```

### 2. Configuration
Copy the `.env.example` to `.env` and configure your database credentials.

```bash
cp .env.example .env
```

### 3. Build and Deploy
The entire stack is containerized for easy deployment:

```bash
docker compose up -d --build
```

## 🧠 AI Recommendations

The API can analyse your price history and holdings and record a **buy**, **sell**, or **hold**
call with its reasoning. It runs on your existing Claude subscription rather than metered API
credits, by invoking the [Claude Code CLI](https://claude.com/claude-code) in headless mode.

### Setup

1.  Generate a long-lived token:

    ```bash
    claude setup-token
    ```

2.  Set it in your `.env` (see `.env.example` for all options):

    ```
    AI_ENABLED=true
    CLAUDE_CODE_OAUTH_TOKEN=<the token from step 1>
    ```

The app runs normally with these unset — the Analyse button simply reports that AI is not
configured.

> **Running outside Docker?** Leave `CLAUDE_CODE_OAUTH_TOKEN` empty if the `claude` CLI is
> already logged in on that machine. A placeholder value overrides working session auth and
> every run fails with `401 Invalid bearer token`. The token is only needed in a container,
> where no session exists.

### How it runs

-   **On demand:** the *Analyse now* button. Rate limited by `AI_MANUAL_COOLDOWN_SECONDS`
    (default 60), because the API has no authentication and the subscription's rate limit is
    shared with your own interactive Claude Code use.
-   **Automatically:** when a new price arrives, at most once per `AI_AUTO_MIN_HOURS`
    (default 24).

Generation takes roughly 20 seconds, so the endpoint returns immediately and the UI polls
`GET /api/signals/status` until it settles.

Only numeric data is sent: price history and per-karat totals. Item names, vendor, and notes
never leave the database, and the returned recommendation is validated against a fixed schema
before it is stored.

## 📂 Project Structure

- `/backend`: The Go API source code.
- `/frontend`: The React application and its Nginx configuration.
- `/migrations`: Structured SQL logic for the database.
- `.github/workflows`: Automated build and deployment pipelines.

## 🤝 Contributing

We welcome contributions! Please feel free to submit a Pull Request.
