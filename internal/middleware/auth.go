package middleware

import (
	"gemini-proxy/internal/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("x-goog-api-key")
		if apiKey == "" {
			apiKey = c.Query("key")
		}
		if apiKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "API key is required. Provide it in the x-goog-api-key header or as a 'key' query parameter."})
			c.Abort()
			return
		}

		authorized := false
		if cfg.ProxyAPIKey != "" && apiKey == cfg.ProxyAPIKey {
			authorized = true
		}

		if !authorized && cfg.AllowGeminiKeysForAuth {
			for _, key := range cfg.GeminiAPIKeys {
				if apiKey == key {
					authorized = true
					break
				}
			}
		}

		if !authorized {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Key"})
			c.Abort()
			return
		}

		c.Next()
	}
}