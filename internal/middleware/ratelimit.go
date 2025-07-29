package middleware

import (
	"gemini-proxy/internal/config"
	"gemini-proxy/internal/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RateLimitMiddleware(valkeyService *services.ValkeyService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		geminiAPIKey, err := valkeyService.GetNextAPIKey(c.Request.Context(), cfg.GeminiAPIKeys)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key"})
			c.Abort()
			return
		}

		c.Set("geminiAPIKey", geminiAPIKey)

		for {
			allowed, err := valkeyService.CheckRateLimit(c.Request.Context(), geminiAPIKey, cfg.RateLimitPerMinute, cfg.RateLimitWindow)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rate limit"})
				c.Abort()
				return
			}

			if allowed {
				break
			}

			// If rate limited, wait and retry. A more sophisticated implementation
			// might have a timeout or a more complex backoff strategy.
			time.Sleep(1 * time.Second)
		}

		c.Next()
	}
}