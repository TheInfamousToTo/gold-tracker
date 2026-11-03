package api

import (
    "errors"
    "net/http"
    "strconv"
    "strings"

    "github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
    "github.com/TheInfamousToTo/gold-tracker/backend/internal/repository"
    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
)

func (h *Handler) ListStockBrokers(c *gin.Context) {
    brokers, err := h.Repo.ListStockBrokers(c.Request.Context())
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, brokers)
}

func (h *Handler) SaveStockBroker(c *gin.Context) {
    var broker model.StockBroker
    if err := c.ShouldBindJSON(&broker); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
    }
    if c.Param("id") != "" {
        id, err := strconv.Atoi(c.Param("id"))
        if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid broker id"}); return }
        broker.ID = id
    } else {
        broker.ID = 0
    }
    if err := broker.Validate(); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
    }
    saved, err := h.Repo.SaveStockBroker(c.Request.Context(), broker)
    var pgErr *pgconn.PgError
    switch {
    case errors.Is(err, pgx.ErrNoRows):
        c.JSON(http.StatusNotFound, gin.H{"error": "broker not found"})
    case errors.As(err, &pgErr) && pgErr.Code == "23505":
        c.JSON(http.StatusConflict, gin.H{"error": "broker name already exists"})
    case err != nil:
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    default:
        c.JSON(http.StatusOK, saved)
    }
}

func (h *Handler) ListStockPositions(c *gin.Context) {
    positions, err := h.Repo.ListStockPositions(c.Request.Context())
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, positions)
}

func (h *Handler) ListStockTrades(c *gin.Context) {
    trades, err := h.Repo.ListStockTrades(c.Request.Context())
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, trades)
}

func (h *Handler) CreateStockTrade(c *gin.Context) {
    var trade model.StockTradeInput
    if err := c.ShouldBindJSON(&trade); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
    }
    saved, err := h.Repo.RecordStockTrade(c.Request.Context(), trade)
    if errors.Is(err, repository.ErrInvalidStockTrade) {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
    }
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusCreated, saved)
}

func (h *Handler) SaveStockQuote(c *gin.Context) {
    symbol := strings.ToUpper(strings.TrimSpace(c.Param("symbol")))
    if symbol == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"}); return }
    var quote model.StockQuoteInput
    if err := c.ShouldBindJSON(&quote); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
    }
    if err := quote.Validate(); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
    }
    if err := h.Repo.SaveStockQuote(c.Request.Context(), symbol, quote); err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            c.JSON(http.StatusNotFound, gin.H{"error": "symbol not found"}); return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
    }
    c.JSON(http.StatusOK, gin.H{"symbol": symbol, "updated": true})
}
