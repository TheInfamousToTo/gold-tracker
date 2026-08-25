package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultSessionTTL = 7 * 24 * time.Hour

	// Brute-force limits. The password lives in an env file and is
	// likely human-chosen, so online guessing is the realistic attack.
	maxFailedAttempts = 5
	lockoutWindow     = 15 * time.Minute
)

// Credentials is what the login form is checked against, plus the
// secret used to sign the sessions it issues.
type Credentials struct {
	Username string
	Password string

	// APIToken lets machine clients (the n8n price feed) call the API
	// without logging in. Optional.
	APIToken string

	// SessionSecret signs session tokens. Defaults to a value derived
	// from the password, so changing the password logs everyone out.
	SessionSecret []byte

	SessionTTL time.Duration
}

// LoadCredentials reads the login configuration from the environment.
func LoadCredentials() Credentials {
	return LoadCredentialsFrom(nil)
}

// LoadCredentialsFrom builds the configuration from an explicit lookup
// table, falling back to the environment for anything absent. Tests use
// it to avoid mutating process state.
func LoadCredentialsFrom(env map[string]string) Credentials {
	get := func(key string) string {
		if env != nil {
			if v, ok := env[key]; ok {
				return v
			}
			return ""
		}
		return os.Getenv(key)
	}

	c := Credentials{
		Username: strings.TrimSpace(get("GOLD_AUTH_USERNAME")),
		Password: get("GOLD_AUTH_PASSWORD"),
		APIToken: strings.TrimSpace(get("GOLD_API_TOKEN")),
	}

	c.SessionTTL = defaultSessionTTL
	if v := get("GOLD_SESSION_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.SessionTTL = time.Duration(n) * time.Hour
		}
	}

	if secret := get("GOLD_SESSION_SECRET"); secret != "" {
		c.SessionSecret = []byte(secret)
	} else if c.Password != "" {
		// Deriving from the password means a password change invalidates
		// every outstanding session, which is the behaviour people
		// expect from changing a password.
		sum := sha256.Sum256([]byte("gold-tracker-session\x00" + c.Password))
		c.SessionSecret = sum[:]
	}

	return c
}

// LoginConfigured reports whether a username and password are both set.
// Without them nobody can obtain a session.
func (c Credentials) LoginConfigured() bool {
	return c.Username != "" && c.Password != ""
}

// Authenticator issues and checks sessions, and throttles failed logins.
type Authenticator struct {
	creds Credentials

	mu       sync.Mutex
	failures map[string]*failureRecord
}

type failureRecord struct {
	count int
	until  time.Time
}

func NewAuthenticator(creds Credentials) *Authenticator {
	return &Authenticator{creds: creds, failures: map[string]*failureRecord{}}
}

// issueSession mints a token of the form "v1.<expiry>.<signature>".
// It carries its own expiry and is signed, so the server stays
// stateless and sessions survive a restart.
func (a *Authenticator) issueSession(now time.Time) string {
	exp := now.Add(a.creds.SessionTTL).Unix()
	payload := fmt.Sprintf("v1.%d", exp)
	return payload + "." + a.sign(payload)
}

func (a *Authenticator) sign(payload string) string {
	mac := hmac.New(sha256.New, a.creds.SessionSecret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// validSession checks the signature first, then the expiry, so an
// expired-but-forged token is still rejected as a forgery.
func (a *Authenticator) validSession(token string, now time.Time) bool {
	if len(a.creds.SessionSecret) == 0 {
		return false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return false
	}
	payload := parts[0] + "." + parts[1]
	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(a.sign(payload))) != 1 {
		return false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}
	return now.Unix() < exp
}

// lockedOut reports whether this client has failed too many times
// recently, and is called before any password comparison.
func (a *Authenticator) lockedOut(key string, now time.Time) (bool, time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	rec := a.failures[key]
	if rec == nil || rec.until.IsZero() || now.After(rec.until) {
		return false, 0
	}
	return true, rec.until.Sub(now)
}

func (a *Authenticator) recordFailure(key string, now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	rec := a.failures[key]
	// Start a fresh record only when there is none, or when a previous
	// lockout has since expired. A zero `until` means "counting", not
	// "expired", so it must not reset the tally.
	if rec == nil || (!rec.until.IsZero() && now.After(rec.until)) {
		rec = &failureRecord{}
		a.failures[key] = rec
	}
	rec.count++
	if rec.count >= maxFailedAttempts {
		rec.until = now.Add(lockoutWindow)
		rec.count = 0
	}
}

func (a *Authenticator) clearFailures(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.failures, key)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login exchanges a username and password for a session token.
func (a *Authenticator) Login(c *gin.Context) {
	if !a.creds.LoginConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "No login is configured on the server. Set GOLD_AUTH_USERNAME and GOLD_AUTH_PASSWORD.",
		})
		return
	}

	now := time.Now()
	key := c.ClientIP()
	if locked, wait := a.lockedOut(key, now); locked {
		c.Header("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": fmt.Sprintf("Too many failed attempts. Try again in %d minutes.", int(wait.Minutes())+1),
		})
		return
	}

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter a username and password."})
		return
	}

	userOK := subtle.ConstantTimeCompare([]byte(req.Username), []byte(a.creds.Username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(req.Password), []byte(a.creds.Password)) == 1
	if !userOK || !passOK {
		a.recordFailure(key, now)
		// One message for both cases: saying which half was wrong tells
		// an attacker when they have found a valid username.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect username or password."})
		return
	}

	a.clearFailures(key)
	c.JSON(http.StatusOK, gin.H{
		"token":      a.issueSession(now),
		"expires_in": int(a.creds.SessionTTL.Seconds()),
	})
}

// RequireAuth rejects any request without a valid session token or the
// machine API token.
//
// The portfolio is personal data and the write endpoints can destroy it
// or spend Claude subscription quota, so this guards reads as well as
// writes. Only /api/health and /api/login are open.
//
// With nothing configured it refuses everything rather than falling
// open: an unauthenticated instance reachable from the internet is
// exactly the failure this exists to prevent.
func (a *Authenticator) RequireAuth() gin.HandlerFunc {
	open := map[string]bool{"/api/health": true, "/api/login": true}

	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions || open[c.FullPath()] {
			c.Next()
			return
		}

		if !a.creds.LoginConfigured() && a.creds.APIToken == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "No authentication is configured on the server, so the API is closed.",
			})
			return
		}

		supplied, ok := bearerToken(c.GetHeader("Authorization"))
		if ok {
			if a.validSession(supplied, time.Now()) {
				c.Next()
				return
			}
			if a.creds.APIToken != "" &&
				subtle.ConstantTimeCompare([]byte(supplied), []byte(a.creds.APIToken)) == 1 {
				c.Next()
				return
			}
		}

		c.Header("WWW-Authenticate", `Bearer realm="gold-tracker"`)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

// Session reports whether the caller currently holds a valid session,
// so the UI can decide between the login page and the app without
// provoking a 401 on every load.
func (a *Authenticator) Session(c *gin.Context) {
	supplied, ok := bearerToken(c.GetHeader("Authorization"))
	c.JSON(http.StatusOK, gin.H{
		"authenticated":     ok && a.validSession(supplied, time.Now()),
		"login_configured":  a.creds.LoginConfigured(),
	})
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}

// CORS restricts which origins may call the API from a browser.
//
// A wildcard would let any page a viewer visits issue writes against
// this API on their behalf. When GOLD_ALLOWED_ORIGIN is unset no
// cross-origin header is sent and only the bundled UI works.
func CORS(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowedOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
