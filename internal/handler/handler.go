package handler

import (
	"bytes"
	"encoding/json"
	"gemini-proxy/internal/config"
	"gemini-proxy/internal/services"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type ProxyHandler struct {
	ValkeyService *services.ValkeyService
	Config        *config.Config
}

func NewProxyHandler(valkeyService *services.ValkeyService, cfg *config.Config) *ProxyHandler {
	return &ProxyHandler{
		ValkeyService: valkeyService,
		Config:        cfg,
	}
}

type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
}

type GeminiResponse struct {
	UsageMetadata UsageMetadata `json:"usageMetadata"`
}

func (h *ProxyHandler) HandleProxy(c *gin.Context) {
	geminiAPIKey, _ := c.Get("geminiAPIKey")

	target, err := url.Parse(h.Config.GeminiAPIURL)
	if err != nil {
		log.Printf("Error parsing target URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = c.Request.URL.Path
		req.Header.Set("x-goog-api-key", geminiAPIKey.(string))
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(body)) // Restore the body

			var geminiResp GeminiResponse
			if err := json.Unmarshal(body, &geminiResp); err == nil {
				h.ValkeyService.UpdateStats(
					c.Request.Context(),
					geminiAPIKey.(string),
					geminiResp.UsageMetadata.PromptTokenCount,
					geminiResp.UsageMetadata.CandidatesTokenCount,
				)
			}
		}
		return nil
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}