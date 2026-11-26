package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/ai"
	"github.com/TheInfamousToTo/gold-tracker/backend/internal/api"
	"github.com/TheInfamousToTo/gold-tracker/backend/internal/report"
	"github.com/TheInfamousToTo/gold-tracker/backend/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if it exists
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Initialize repository
	repo, err := repository.NewPostgresRepository()
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	// Initialize the AI service. It runs on the owner's Claude
	// subscription through the headless CLI rather than the metered
	// API, and stays fully disabled unless both AI_ENABLED and a token
	// are set.
	aiCfg := ai.LoadConfigFromEnv()
	aiService := ai.NewService(repo, &ai.CLIRunner{Timeout: aiCfg.Timeout}, aiCfg)
	if aiCfg.Enabled {
		log.Printf("AI signals enabled (model %s)", aiCfg.Model)
	} else {
		log.Print("AI signals disabled (set AI_ENABLED=true and CLAUDE_CODE_OAUTH_TOKEN to enable)")
	}

	// Monthly report: generated on the 1st for the previous month and
	// posted to Discord when DISCORD_WEBHOOK_URL is set.
	reports := report.NewService(repo, &ai.CLIRunner{Timeout: aiCfg.Timeout}, aiCfg, os.Getenv("DISCORD_WEBHOOK_URL"))
	reports.Sure = report.SureFromEnv()
	go reports.Schedule(context.Background())

	// Initialize handler
	h := api.NewHandler(repo, aiService, reports)

	r := gin.Default()
	// The login lockout keys on ClientIP. Trust X-Forwarded-For hops only
	// from private addresses (the UI's nginx, the ingress), so a client
	// can't dodge the lockout by sending its own header.
	if err := r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.1"}); err != nil {
		log.Fatal(err)
	}

	// Cross-origin access is off unless an origin is named. A wildcard
	// would let any page a viewer visits issue writes against this API.
	r.Use(api.CORS(os.Getenv("GOLD_ALLOWED_ORIGIN")))

	// Everything except /api/health and /api/login requires a session
	// from the login form, or the machine token the price feed uses.
	creds := api.LoadCredentials()
	auth := api.NewAuthenticator(creds)
	if !creds.LoginConfigured() {
		log.Print("WARNING: GOLD_AUTH_USERNAME and GOLD_AUTH_PASSWORD are not set — nobody can sign in")
	}
	r.POST("/api/login", auth.Login)
	r.GET("/api/session", auth.Session)
	r.Use(auth.RequireAuth())

	// Routes
	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/health", h.Health)
		
		apiGroup.GET("/items", h.GetItems)
		apiGroup.POST("/items", h.CreateItem)
		apiGroup.PUT("/items/:id", h.UpdateItem)
		apiGroup.DELETE("/items/:id", h.DeleteItem)
		
		// Brokerage schedules and stock/ETF transactions (manual quotes).
		apiGroup.GET("/stocks/brokers", h.ListStockBrokers)
		apiGroup.POST("/stocks/brokers", h.SaveStockBroker)
		apiGroup.PUT("/stocks/brokers/:id", h.SaveStockBroker)
		apiGroup.GET("/stocks/positions", h.ListStockPositions)
		apiGroup.GET("/stocks/trades", h.ListStockTrades)
		apiGroup.POST("/stocks/trades", h.CreateStockTrade)
		apiGroup.DELETE("/stocks/trades/:id", h.DeleteStockTrade)
		apiGroup.PUT("/stocks/quotes/:symbol", h.SaveStockQuote)
		
		apiGroup.GET("/portfolio", h.GetPortfolio)
		
		apiGroup.GET("/prices", h.GetPrices)
		apiGroup.POST("/prices", h.CreatePrice)
		
		apiGroup.GET("/signals", h.GetSignals)
		apiGroup.POST("/signals/generate", h.GenerateSignal)
		apiGroup.GET("/signals/status", h.SignalStatus)
		apiGroup.POST("/signals/review", h.ReviewPlan)

		apiGroup.GET("/reports", h.ListReports)
		apiGroup.GET("/reports/status", h.ReportStatus)
		apiGroup.POST("/reports/generate", h.GenerateReport)

		apiGroup.GET("/ai/settings", h.GetAdvisorSettings)
		apiGroup.PUT("/ai/settings", h.UpdateAdvisorSettings)
	}

	fmt.Printf("Gold Tracker Go API listening on port %s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
