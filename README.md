# Gold Tracker 🚀

An enterprise-grade, self-hosted application for logging gold purchases, tracking spot prices, and managing your physical gold portfolio. Built for reliability, speed, and modern observability.

## 🌟 Key Features

- **Portfolio Management:** Log purchases with detailed metadata (karat, weight, vendor, notes).
- **Live Price Tracking:** Monitor current spot prices for 24K, 22K, 21K, and 18K gold.
- **Profit/Loss Analysis:** Real-time calculation of gain/loss based on current market data.
- **n8n Integration:** Designed to work with n8n for automated price feeds and AI signals.
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
    -   **AI Signals:** Another workflow can process price history to generate buy/sell/hold signals stored in the `signals_log` table.
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

## 📂 Project Structure

- `/backend`: The Go API source code.
- `/frontend`: The React application and its Nginx configuration.
- `/migrations`: Structured SQL logic for the database.
- `.github/workflows`: Automated build and deployment pipelines.

## 🤝 Contributing

We welcome contributions! Please feel free to submit a Pull Request.
