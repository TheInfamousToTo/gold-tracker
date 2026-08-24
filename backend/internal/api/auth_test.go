package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func authRouter(token string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireToken(token))
	r.GET("/api/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/api/portfolio", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"items": []int{}}) })
	r.DELETE("/api/items/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"deleted": true}) })
	return r
}

func do(r *gin.Engine, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// The vulnerability this middleware exists to close: an unauthenticated
// DELETE from the public internet erasing the owner's holdings.
func TestUnauthenticatedDeleteIsRejected(t *testing.T) {
	w := do(authRouter("secret"), http.MethodDelete, "/api/items/1", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestUnauthenticatedReadIsRejected(t *testing.T) {
	// The portfolio is personal data, so reads are guarded too.
	w := do(authRouter("secret"), http.MethodGet, "/api/portfolio", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestValidTokenIsAccepted(t *testing.T) {
	w := do(authRouter("secret"), http.MethodDelete, "/api/items/1", "Bearer secret")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestWrongTokenIsRejected(t *testing.T) {
	w := do(authRouter("secret"), http.MethodGet, "/api/portfolio", "Bearer wrong")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestHealthStaysOpenForContainerChecks(t *testing.T) {
	w := do(authRouter("secret"), http.MethodGet, "/api/health", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// Failing closed matters more than staying available: a misconfigured
// deploy must not silently serve an open API to the internet.
func TestMissingServerTokenClosesTheAPI(t *testing.T) {
	w := do(authRouter(""), http.MethodGet, "/api/portfolio", "Bearer anything")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestMissingServerTokenStillAllowsHealth(t *testing.T) {
	w := do(authRouter(""), http.MethodGet, "/api/health", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestMalformedAuthorizationHeaders(t *testing.T) {
	r := authRouter("secret")
	for _, header := range []string{
		"secret",         // no scheme
		"Basic secret",   // wrong scheme
		"Bearer",         // no value
		"Bearer ",        // empty value
		"Bearer secretX", // near miss
		"Bearer  secret", // padded, but TrimSpace makes this valid
	} {
		w := do(r, http.MethodGet, "/api/portfolio", header)
		if header == "Bearer  secret" {
			if w.Code != http.StatusOK {
				t.Errorf("header %q: status = %d, want 200 (surrounding space is trimmed)", header, w.Code)
			}
			continue
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("header %q: status = %d, want 401", header, w.Code)
		}
	}
}

func TestBearerSchemeIsCaseInsensitive(t *testing.T) {
	w := do(authRouter("secret"), http.MethodGet, "/api/portfolio", "bearer secret")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (RFC 7235 schemes are case-insensitive)", w.Code)
	}
}

func TestUnauthorizedResponseAdvertisesScheme(t *testing.T) {
	w := do(authRouter("secret"), http.MethodGet, "/api/portfolio", "")
	if got := w.Header().Get("WWW-Authenticate"); got == "" {
		t.Error("401 should carry a WWW-Authenticate header")
	}
}

func TestCORSOmitsHeaderWhenNoOriginConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(""))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := do(r, http.MethodGet, "/x", "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty so only same-origin works", got)
	}
}

func TestCORSEchoesConfiguredOriginOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS("https://gold.example"))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := do(r, http.MethodGet, "/x", "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://gold.example" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want the configured origin", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got == "*" {
		t.Fatal("wildcard origin lets any page issue writes on a viewer's behalf")
	}
}
