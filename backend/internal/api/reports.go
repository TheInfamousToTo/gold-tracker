package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/report"
	"github.com/gin-gonic/gin"
)

func (h *Handler) ListReports(c *gin.Context) {
	reports, err := h.Repo.ListReports(c.Request.Context(), 24)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, reports)
}

func (h *Handler) ReportStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.Reports.Status())
}

// GenerateReport starts a report for the given YYYY-MM month (default:
// last month) and returns at once; the client polls ReportStatus.
func (h *Handler) GenerateReport(c *gin.Context) {
	var body struct {
		Month string `json:"month"`
	}
	_ = c.ShouldBindJSON(&body)
	now := time.Now()
	if body.Month == "" {
		body.Month = report.PreviousMonth(now)
	}
	if body.Month > now.Format("2006-01") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "month is in the future"})
		return
	}
	if err := h.Reports.Start(body.Month); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, report.ErrRunning) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "started", "month": body.Month})
}
