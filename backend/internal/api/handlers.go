package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/ai"
	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
	"github.com/TheInfamousToTo/gold-tracker/backend/internal/repository"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Repo *repository.PostgresRepository
	AI   *ai.Service
}

func NewHandler(repo *repository.PostgresRepository, aiService *ai.Service) *Handler {
	return &Handler{Repo: repo, AI: aiService}
}

// Health check
func (h *Handler) Health(c *gin.Context) {
	if err := h.Repo.Pool.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "disconnected", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "connected"})
}

// Items
func (h *Handler) GetItems(c *gin.Context) {
	items, err := h.Repo.GetItems(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) CreateItem(c *gin.Context) {
	var item model.GoldItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Basic validation
	if item.ItemName == "" || item.PurchaseDate == "" || item.WeightGrams <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: item_name, purchase_date, and weight_grams are required"})
		return
	}

	newItem, err := h.Repo.CreateItem(c.Request.Context(), item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, newItem)
}

func (h *Handler) UpdateItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var item model.GoldItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedItem, err := h.Repo.UpdateItem(c.Request.Context(), id, item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedItem)
}

func (h *Handler) DeleteItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if err := h.Repo.DeleteItem(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}

// Portfolio
func (h *Handler) GetPortfolio(c *gin.Context) {
	summary, err := h.Repo.GetPortfolioSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// Prices
func (h *Handler) GetPrices(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "60")
	limit, _ := strconv.Atoi(limitStr)
	if limit > 365 {
		limit = 365
	}

	prices, err := h.Repo.GetPrices(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, prices)
}

func (h *Handler) CreatePrice(c *gin.Context) {
	var price model.GoldPrice
	if err := c.ShouldBindJSON(&price); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if price.PriceDate == "" || price.PricePerGram24k <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: price_date and price_per_gram_24k > 0 are required"})
		return
	}

	newPrice, err := h.Repo.CreatePrice(c.Request.Context(), price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, newPrice)

	// Fire and forget: a fresh price may make a new signal due, but
	// this write is n8n's, and AI trouble must never fail it.
	go h.AI.MaybeAutoGenerate(context.Background())
}

// Signals
func (h *Handler) GetSignals(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "30")
	limit, _ := strconv.Atoi(limitStr)
	if limit > 200 {
		limit = 200
	}

	signals, err := h.Repo.GetSignals(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, signals)
}

// GenerateSignal starts a run and returns immediately. Generation can
// take minutes, which is far too long to hold an HTTP connection open
// through nginx, so the client polls SignalStatus instead.
func (h *Handler) GenerateSignal(c *gin.Context) {
	if !h.AI.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI is not configured"})
		return
	}
	if err := h.AI.TryStart("manual"); err != nil {
		status := http.StatusConflict
		if errors.Is(err, ai.ErrCoolingDown) {
			status = http.StatusTooManyRequests
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	go h.AI.RunOnce(context.Background(), "manual")
	c.JSON(http.StatusAccepted, gin.H{"status": "started"})
}

func (h *Handler) SignalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.AI.GetStatus())
}
