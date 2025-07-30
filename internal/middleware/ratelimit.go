package middleware

import (
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
			C:    c,
			Done: make(chan struct{}),
		}
		queue.Add(req)
		<-req.Done
	}
}

func processQueue(valkeyService *services.ValkeyService, cfg *config.Config, queue *services.RequestQueue) {
	for {
		req := queue.Get()
		var earliestReset time.Time
		for _, apiKey := range cfg.GeminiAPIKeys {
			allowed, resetTime, err := valkeyService.CheckRateLimit(req.C.Request.Context(), apiKey, cfg.RateLimitPerMinute, 100, 250000, cfg.RateLimitWindow)
			if err != nil {
				req.C.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rate limit"})
				close(req.Done)
				continue
			}

			if allowed {
				req.C.Set("geminiAPIKey", apiKey)
				req.C.Next()
				close(req.Done)
				goto nextRequest
			}
			if earliestReset.IsZero() || resetTime.Before(earliestReset) {
				earliestReset = resetTime
			}
		}
		time.Sleep(time.Until(earliestReset))
		queue.Add(req)
	nextRequest:
	}
}