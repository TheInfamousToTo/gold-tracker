package api

import (
	"context"
	"errors"
	"fmt"
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

// validateItem applies the same rule as the schema's chk_item_purity
// constraint, so a bad payload comes back as a 400 explaining itself
// rather than a constraint violation surfacing as a 500.
//
// It normalises a missing metal to gold, which is what the column
// defaulted to before silver existed.
func validateItem(item *model.GoldItem) error {
	if item.MetalType == "" {
		item.MetalType = model.MetalGold
	}
	if item.ItemName == "" || item.PurchaseDate == "" || item.WeightGrams <= 0 {
		return errors.New("invalid payload: item_name, purchase_date, and weight_grams are required")
	}
	if !model.IsKnownMetal(item.MetalType) {
		return fmt.Errorf("invalid payload: metal_type must be %q or %q", model.MetalGold, model.MetalSilver)
	}

	// Each metal carries only the purity notation it is traded in, and
	// the other column stays null — so clear whichever does not apply
	// rather than trusting the client to omit it.
	if item.MetalType == model.MetalSilver {
		if item.PurityFineness == nil || *item.PurityFineness <= 0 {
			return errors.New("invalid payload: silver requires purity_fineness (999, 925, or 900)")
		}
		item.PurityKarat = nil
		return nil
	}
	if item.PurityKarat == nil || *item.PurityKarat <= 0 {
		return errors.New("invalid payload: gold requires purity_karat")
	}
	item.PurityFineness = nil
	return nil
}

func (h *Handler) CreateItem(c *gin.Context) {
	var item model.GoldItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateItem(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	if err := validateItem(&item); err != nil {
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
//
// Gold and silver are quoted independently and stored in separate
// tables, so the metal selects which one is read or written. It is
// optional and defaults to gold, which keeps the existing price feed's
// payload working unchanged.
func requestedMetal(value string) (string, error) {
	if value == "" {
		return model.MetalGold, nil
	}
	if !model.IsKnownMetal(value) {
		return "", fmt.Errorf("invalid metal %q: expected %q or %q", value, model.MetalGold, model.MetalSilver)
	}
	return value, nil
}

func (h *Handler) GetPrices(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "60")
	limit, _ := strconv.Atoi(limitStr)
	if limit > 365 {
		limit = 365
	}

	metal, err := requestedMetal(c.Query("metal"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if metal == model.MetalSilver {
		prices, err := h.Repo.GetSilverPrices(c.Request.Context(), limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, prices)
		return
	}

	prices, err := h.Repo.GetPrices(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, prices)
}

// pricePayload is the write shape for both metals. Each metal reads
// only its own rate field, so a body naming one metal and carrying the
// other's price is rejected rather than silently stored as zero.
type pricePayload struct {
	Metal           string  `json:"metal"`
	PriceDate       string  `json:"price_date"`
	PricePerGram24k float64 `json:"price_per_gram_24k"`
	PricePerGram999 float64 `json:"price_per_gram_999"`
	Source          string  `json:"source"`
}

func (h *Handler) CreatePrice(c *gin.Context) {
	var payload pricePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metal, err := requestedMetal(payload.Metal)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.PriceDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: price_date is required"})
		return
	}

	if metal == model.MetalSilver {
		if payload.PricePerGram999 <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: price_per_gram_999 > 0 is required for silver"})
			return
		}
		newPrice, err := h.Repo.CreateSilverPrice(c.Request.Context(), model.SilverPrice{
			PriceDate:       payload.PriceDate,
			PricePerGram999: payload.PricePerGram999,
			Source:          payload.Source,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, newPrice)
		go h.AI.MaybeAutoGenerate(context.Background())
		return
	}

	if payload.PricePerGram24k <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: price_date and price_per_gram_24k > 0 are required"})
		return
	}

	newPrice, err := h.Repo.CreatePrice(c.Request.Context(), model.GoldPrice{
		PriceDate:       payload.PriceDate,
		PricePerGram24k: payload.PricePerGram24k,
		Source:          payload.Source,
	})
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
