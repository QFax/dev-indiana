package main

import (
	"gemini-proxy/internal/config"
	"gemini-proxy/internal/handler"
	"gemini-proxy/internal/middleware"
	"gemini-proxy/internal/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	valkeyService, err := services.NewValkeyService(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Valkey: %v", err)
	}

	proxyHandler := handler.NewProxyHandler(valkeyService, cfg)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// All other routes are handled by the proxy
	proxyGroup := r.Group("/")
	proxyGroup.Use(middleware.AuthMiddleware(cfg))
	proxyGroup.Use(middleware.RateLimitMiddleware(valkeyService, cfg))
	proxyGroup.Any("/*path", proxyHandler.HandleProxy)

	log.Printf("Server starting on port 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}