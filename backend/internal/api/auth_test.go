package api

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func testCreds() Credentials {
	c := Credentials{Username: "owner", Password: "correct horse", APIToken: "machine-token"}
	c.SessionTTL = time.Hour
	c.SessionSecret = []byte("test-secret")
	return c
}

func authRouter(creds Credentials) (*gin.Engine, *Authenticator) {
	gin.SetMode(gin.TestMode)
	a := NewAuthenticator(creds)
	r := gin.New()
	r.POST("/api/login", a.Login)
	r.GET("/api/session", a.Session)
	r.Use(a.RequireAuth())
	r.GET("/api/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/api/portfolio", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"items": []int{}}) })
	r.DELETE("/api/items/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"deleted": true}) })
	return r, a
}

func req(r *gin.Engine, method, path, auth, body string) *httptest.ResponseRecorder {
	var rq *http.Request
	if body == "" {
		rq = httptest.NewRequest(method, path, nil)
	} else {
		rq = httptest.NewRequest(method, path, strings.NewReader(body))
		rq.Header.Set("Content-Type", "application/json")
	}
	if auth != "" {
		rq.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, rq)
	return w
}

func login(t *testing.T, r *gin.Engine, user, pass string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(loginRequest{Username: user, Password: pass})
	return req(r, http.MethodPost, "/api/login", "", string(body))
}

func tokenFrom(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("login response is not JSON: %v", err)
	}
	if out.Token == "" {
		t.Fatal("login returned no token")
	}
	return out.Token
}

// The vulnerability this exists to close: anyone reaching the site
// could read and destroy the portfolio.
func TestUnauthenticatedRequestsAreRejected(t *testing.T) {
	r, _ := authRouter(testCreds())
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/portfolio"},
		{http.MethodDelete, "/api/items/1"},
	} {
		w := req(r, tc.method, tc.path, "", "")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401", tc.method, tc.path, w.Code)
		}
	}
}

func TestLoginIssuesUsableSession(t *testing.T) {
	r, _ := authRouter(testCreds())

	w := login(t, r, "owner", "correct horse")
	if w.Code != http.StatusOK {
		t.Fatalf("login = %d, want 200", w.Code)
	}
	token := tokenFrom(t, w)

	if got := req(r, http.MethodGet, "/api/portfolio", "Bearer "+token, "").Code; got != http.StatusOK {
		t.Fatalf("portfolio with session = %d, want 200", got)
	}
	if got := req(r, http.MethodDelete, "/api/items/1", "Bearer "+token, "").Code; got != http.StatusOK {
		t.Fatalf("delete with session = %d, want 200", got)
	}
}

func TestLoginRejectsWrongCredentials(t *testing.T) {
	r, _ := authRouter(testCreds())
	for _, tc := range []struct{ user, pass string }{
		{"owner", "wrong"},
		{"someone", "correct horse"},
		{"", ""},
	} {
		w := login(t, r, tc.user, tc.pass)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("login(%q,%q) = %d, want 401", tc.user, tc.pass, w.Code)
		}
	}
}

// Saying which half was wrong tells an attacker when they have found a
// valid username.
func TestLoginFailureDoesNotRevealWhichFieldWasWrong(t *testing.T) {
	r, _ := authRouter(testCreds())
	badUser := login(t, r, "someone", "correct horse").Body.String()
	badPass := login(t, r, "owner", "wrong").Body.String()
	if badUser != badPass {
		t.Fatalf("responses differ:\n user: %s\n pass: %s", badUser, badPass)
	}
}

func TestRepeatedFailuresLockOut(t *testing.T) {
	r, _ := authRouter(testCreds())
	for i := 0; i < maxFailedAttempts; i++ {
		login(t, r, "owner", "wrong")
	}

	if got := login(t, r, "owner", "wrong").Code; got != http.StatusTooManyRequests {
		t.Fatalf("after %d failures = %d, want 429", maxFailedAttempts, got)
	}
	// The correct password is refused too, or the lockout would be
	// trivially bypassed by guessing it during the window.
	if got := login(t, r, "owner", "correct horse").Code; got != http.StatusTooManyRequests {
		t.Fatalf("correct password during lockout = %d, want 429", got)
	}
}

func TestForgedAndTamperedSessionsAreRejected(t *testing.T) {
	r, a := authRouter(testCreds())
	valid := tokenFrom(t, login(t, r, "owner", "correct horse"))

	other := NewAuthenticator(Credentials{
		Username: "owner", Password: "correct horse",
		SessionSecret: []byte("different-secret"), SessionTTL: time.Hour,
	})

	for name, token := range map[string]string{
		"signed with another secret": other.issueSession(time.Now()),
		"tampered expiry":            "v1.99999999999." + strings.Split(valid, ".")[2],
		"garbage":                    "v1.abc.def",
		"wrong version":              strings.Replace(valid, "v1.", "v2.", 1),
		"not a token":                "hello",
	} {
		if got := req(r, http.MethodGet, "/api/portfolio", "Bearer "+token, "").Code; got != http.StatusUnauthorized {
			t.Errorf("%s = %d, want 401", name, got)
		}
	}
	_ = a
}

func TestExpiredSessionIsRejected(t *testing.T) {
	creds := testCreds()
	creds.SessionTTL = time.Hour
	r, a := authRouter(creds)

	expired := a.issueSession(time.Now().Add(-2 * time.Hour))
	if got := req(r, http.MethodGet, "/api/portfolio", "Bearer "+expired, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("expired session = %d, want 401", got)
	}
}

// The n8n price feed cannot log in through a form.
func TestMachineTokenStillWorks(t *testing.T) {
	r, _ := authRouter(testCreds())
	if got := req(r, http.MethodGet, "/api/portfolio", "Bearer machine-token", "").Code; got != http.StatusOK {
		t.Fatalf("machine token = %d, want 200", got)
	}
}

func TestHealthAndLoginStayOpen(t *testing.T) {
	r, _ := authRouter(testCreds())
	if got := req(r, http.MethodGet, "/api/health", "", "").Code; got != http.StatusOK {
		t.Errorf("health = %d, want 200 for container checks", got)
	}
	if got := login(t, r, "owner", "correct horse").Code; got != http.StatusOK {
		t.Errorf("login = %d, want 200 without credentials attached", got)
	}
}

// A misconfigured deploy must not silently serve an open API.
func TestNothingConfiguredClosesTheAPI(t *testing.T) {
	r, _ := authRouter(Credentials{SessionTTL: time.Hour})
	if got := req(r, http.MethodGet, "/api/portfolio", "Bearer anything", "").Code; got != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured = %d, want 503", got)
	}
	if got := login(t, r, "owner", "pw").Code; got != http.StatusServiceUnavailable {
		t.Fatalf("login when unconfigured = %d, want 503", got)
	}
}

func TestSessionEndpointReportsState(t *testing.T) {
	r, _ := authRouter(testCreds())

	var out struct {
		Authenticated   bool `json:"authenticated"`
		LoginConfigured bool `json:"login_configured"`
	}
	json.Unmarshal(req(r, http.MethodGet, "/api/session", "", "").Body.Bytes(), &out)
	if out.Authenticated {
		t.Error("authenticated = true without a token")
	}
	if !out.LoginConfigured {
		t.Error("login_configured = false though a username and password are set")
	}

	token := tokenFrom(t, login(t, r, "owner", "correct horse"))
	json.Unmarshal(req(r, http.MethodGet, "/api/session", "Bearer "+token, "").Body.Bytes(), &out)
	if !out.Authenticated {
		t.Error("authenticated = false with a valid session")
	}
}

func TestChangingThePasswordInvalidatesSessions(t *testing.T) {
	// The secret derives from the password, so a password change logs
	// everyone out — what people expect from changing a password.
	before := LoadCredentialsFrom(map[string]string{
		"GOLD_AUTH_USERNAME": "owner", "GOLD_AUTH_PASSWORD": "old",
	})
	after := LoadCredentialsFrom(map[string]string{
		"GOLD_AUTH_USERNAME": "owner", "GOLD_AUTH_PASSWORD": "new",
	})

	oldSession := NewAuthenticator(before).issueSession(time.Now())
	if NewAuthenticator(after).validSession(oldSession, time.Now()) {
		t.Fatal("a session issued under the old password still validates")
	}
}

func TestBearerSchemeIsCaseInsensitive(t *testing.T) {
	r, _ := authRouter(testCreds())
	token := tokenFrom(t, login(t, r, "owner", "correct horse"))
	if got := req(r, http.MethodGet, "/api/portfolio", "bearer "+token, "").Code; got != http.StatusOK {
		t.Fatalf("lowercase scheme = %d, want 200 (RFC 7235 schemes are case-insensitive)", got)
	}
}

func TestMalformedAuthorizationHeaders(t *testing.T) {
	r, _ := authRouter(testCreds())
	for _, header := range []string{"machine-token", "Basic machine-token", "Bearer", "Bearer "} {
		if got := req(r, http.MethodGet, "/api/portfolio", header, "").Code; got != http.StatusUnauthorized {
			t.Errorf("header %q = %d, want 401", header, got)
		}
	}
}

func TestCORSOmitsHeaderWhenNoOriginConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(""))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	if got := req(r, http.MethodGet, "/x", "", "").Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty so only same-origin works", got)
	}
}

func TestCORSEchoesConfiguredOriginOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS("https://gold.example"))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	got := req(r, http.MethodGet, "/x", "", "").Header().Get("Access-Control-Allow-Origin")
	if got != "https://gold.example" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want the configured origin", got)
	}
}

// CodeQL flagged deriving the signing key with a bare SHA-256 over the
// password: cheap to reverse, so a leaked key would give up the
// password. The derivation must be a slow KDF.
func TestSessionSecretDerivationIsSlowAndStable(t *testing.T) {
	first := deriveSessionSecret("correct horse")
	second := deriveSessionSecret("correct horse")

	if len(first) != 32 {
		t.Fatalf("derived key is %d bytes, want 32", len(first))
	}
	if string(first) != string(second) {
		t.Fatal("derivation must be reproducible, or sessions break on restart")
	}
	if string(first) == string(deriveSessionSecret("correct horse ")) {
		t.Fatal("a different password produced the same key")
	}

	// A bare SHA-256 of the password would be indistinguishable from a
	// precomputed hash; the KDF must not produce one.
	sum := sha256.Sum256([]byte("correct horse"))
	if string(first) == string(sum[:]) {
		t.Fatal("derived key is a plain SHA-256 of the password")
	}

	start := time.Now()
	deriveSessionSecret("timing check")
	if elapsed := time.Since(start); elapsed < time.Millisecond {
		t.Errorf("derivation took %v, too fast to resist brute force", elapsed)
	}
}
