package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/ai"
	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
	"github.com/gin-gonic/gin"
)

type stubRepo struct{}

func (stubRepo) GetPrices(ctx context.Context, limit int) ([]model.GoldPrice, error) {
	return nil, nil
}

func (stubRepo) GetPortfolioSummary(ctx context.Context) (model.PortfolioSummary, error) {
	return model.PortfolioSummary{}, nil
}

func (stubRepo) CreateSignal(ctx context.Context, s model.SignalLog) (model.SignalLog, error) {
	return s, nil
}

func (stubRepo) GetLatestSignal(ctx context.Context, source string) (*model.SignalLog, error) {
	return nil, nil
}

// blockingRunner holds a run open so a test can observe the in-flight state.
type blockingRunner struct{ release chan struct{} }

func (b *blockingRunner) Run(ctx context.Context, prompt, model string) (ai.RunResult, error) {
	<-b.release
	return ai.RunResult{Result: `{"signal":"HOLD","confidence":0.5,"reasoning":"x","horizon_days":30,"key_factors":[]}`}, nil
}

func newTestHandler(enabled bool, runner ai.Runner) *Handler {
	cfg := ai.Config{Enabled: enabled, Model: "claude-opus-5", Timeout: 5 * time.Second, AutoMinHours: 24}
	return &Handler{AI: ai.NewService(stubRepo{}, runner, cfg)}
}

func waitUntilIdle(t *testing.T, h *Handler) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if !h.AI.GetStatus().Running {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("service never returned to idle")
}

func TestGenerateSignalDisabledReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(false, &blockingRunner{release: make(chan struct{})})
	r := gin.New()
	r.POST("/api/signals/generate", h.GenerateSignal)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/signals/generate", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestGenerateSignalStartsThenReports409WhileRunning(t *testing.T) {
	gin.SetMode(gin.TestMode)
	release := make(chan struct{})
	h := newTestHandler(true, &blockingRunner{release: release})
	r := gin.New()
	r.POST("/api/signals/generate", h.GenerateSignal)

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodPost, "/api/signals/generate", nil))
	if w1.Code != http.StatusAccepted {
		t.Fatalf("first call status = %d, want 202", w1.Code)
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/api/signals/generate", nil))
	if w2.Code != http.StatusConflict {
		t.Fatalf("second call status = %d, want 409", w2.Code)
	}

	close(release)
	waitUntilIdle(t, h)
}

func TestSignalStatusReportsIdleState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(true, &blockingRunner{release: make(chan struct{})})
	r := gin.New()
	r.GET("/api/signals/status", h.SignalStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/signals/status", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var got ai.Status
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid Status JSON: %v", err)
	}
	if got.Running {
		t.Error("running = true, want false")
	}
	if !got.Enabled {
		t.Error("enabled = false, want true so the UI can show the button")
	}
}

func TestSignalStatusReportsDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestHandler(false, &blockingRunner{release: make(chan struct{})})
	r := gin.New()
	r.GET("/api/signals/status", h.SignalStatus)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/signals/status", nil))

	var got ai.Status
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid Status JSON: %v", err)
	}
	if got.Enabled {
		t.Error("enabled = true, want false so the UI renders the not-configured state")
	}
}
