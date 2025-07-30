package middleware

import (
	"context"
	"gemini-proxy/internal/config"
	"gemini-proxy/internal/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RateLimitMiddleware(valkeyService *services.ValkeyService, cfg *config.Config, queue *services.RequestQueue) gin.HandlerFunc {
	go processQueue(valkeyService, cfg, queue)

	return func(c *gin.Context) {
		req := &services.Request{
			APIKeyChan: make(chan services.APIKeyResult, 1),
		}
		queue.Add(req)

		result := <-req.APIKeyChan
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			c.Abort()
			return
		}

		c.Set("geminiAPIKey", result.APIKey)
		c.Next()
	}
}

func processQueue(valkeyService *services.ValkeyService, cfg *config.Config, queue *services.RequestQueue) {
	for {
		req := queue.Get()
		var earliestReset time.Time

		for {
			for _, apiKey := range cfg.GeminiAPIKeys {
				allowed, resetTime, err := valkeyService.CheckRateLimit(context.Background(), apiKey, cfg.RateLimitPerMinute, 100, 250000, cfg.RateLimitWindow)
				if err != nil {
					req.APIKeyChan <- services.APIKeyResult{Error: err}
					continue
				}

				if allowed {
					req.APIKeyChan <- services.APIKeyResult{APIKey: apiKey}
					goto nextRequest
				}

				if earliestReset.IsZero() || resetTime.Before(earliestReset) {
					earliestReset = resetTime
				}
			}
			time.Sleep(time.Until(earliestReset))
		}
	nextRequest:
	}
}