package api

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenFromEnv reads the shared secret guarding the API. An empty
// result means the API is unauthenticated.
func TokenFromEnv() string {
	return strings.TrimSpace(os.Getenv("GOLD_API_TOKEN"))
}

// RequireToken rejects any request without the shared secret.
//
// The portfolio is personal data and the write endpoints can destroy it
// or spend Claude subscription quota, so this guards reads as well as
// writes. Only /api/health is left open, for container health checks.
//
// When no token is configured the middleware refuses every request
// rather than falling open: an unauthenticated instance reachable from
// the internet is exactly the failure this exists to prevent.
func RequireToken(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		if c.FullPath() == "/api/health" {
			c.Next()
			return
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "GOLD_API_TOKEN is not set on the server, so the API is closed",
			})
			return
		}

		if !validCredential(c.GetHeader("Authorization"), token) {
			c.Header("WWW-Authenticate", `Bearer realm="gold-tracker"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Next()
	}
}

// validCredential accepts "Bearer <token>". Comparison is
// constant-time so a wrong token cannot be recovered by timing.
func validCredential(header, token string) bool {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return false
	}
	supplied := strings.TrimSpace(header[len(prefix):])
	return subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) == 1
}

// CORS restricts which origins may call the API from a browser.
//
// The previous "*" meant any page a viewer visited could issue writes
// against this API on their behalf. allowedOrigin comes from
// GOLD_ALLOWED_ORIGIN; when unset, no cross-origin header is sent at
// all and only same-origin requests (the bundled UI) work.
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
