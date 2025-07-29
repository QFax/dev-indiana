package services

import (
	"context"
	"fmt"
	"gemini-proxy/internal/config"
	"time"

	"github.com/valkey-io/valkey-go"
)

type ValkeyService struct {
	Client valkey.Client
}

func NewValkeyService(cfg *config.Config) (*ValkeyService, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{fmt.Sprintf("%s:%s", cfg.ValkeyHost, cfg.ValkeyPort)},
		Password:    cfg.ValkeyPassword,
	})
	if err != nil {
		return nil, err
	}

	return &ValkeyService{Client: client}, nil
}

func (s *ValkeyService) GetNextAPIKey(ctx context.Context, keys []string) (string, error) {
	if len(keys) == 0 {
		return "", fmt.Errorf("no API keys available")
	}
	index, err := s.Client.Do(ctx, s.Client.B().Incr().Key("proxy:round_robin_index").Build()).AsInt64()
	if err != nil {
		return "", err
	}
	return keys[index%int64(len(keys))], nil
}

func (s *ValkeyService) CheckRateLimit(ctx context.Context, apiKey string, limit int, window string) (bool, error) {
	now := time.Now().Unix()
	key := fmt.Sprintf("proxy:%s:requests:minute", apiKey)

	var err error
	if window == "fixed" {
		// For fixed window, we just need to check the count in the current minute.
		// A real implementation would need a more robust way to define the start of the minute.
		// This is simplified for the example.
		_, err = s.Client.Do(ctx, s.Client.B().Zremrangebyscore().Key(key).Min(fmt.Sprintf("%d", 0)).Max(fmt.Sprintf("%d", now-60)).Build()).AsInt64()
	} else { // sliding window
		_, err = s.Client.Do(ctx, s.Client.B().Zremrangebyscore().Key(key).Min(fmt.Sprintf("%d", 0)).Max(fmt.Sprintf("%d", now-60)).Build()).AsInt64()
	}

	if err != nil {
		return false, err
	}

	count, err := s.Client.Do(ctx, s.Client.B().Zcard().Key(key).Build()).AsInt64()
	if err != nil {
		return false, err
	}

	if count >= int64(limit) {
		return false, nil // Rate limit exceeded
	}

	_, err = s.Client.Do(ctx, s.Client.B().Zadd().Key(key).ScoreMember().ScoreMember(float64(now), fmt.Sprintf("%d", now)).Build()).AsInt64()
	if err != nil {
		return false, err
	}

	// Set expiration for the key to clean up old data
	s.Client.Do(ctx, s.Client.B().Expire().Key(key).Seconds(86400).Build()) // 24 hours

	return true, nil
}

func (s *ValkeyService) UpdateStats(ctx context.Context, apiKey string, promptTokens, completionTokens int) error {
	now := time.Now()
	minuteKey := fmt.Sprintf("proxy:%s:stats:minute:%s", apiKey, now.Format("2006-01-02T15:04"))
	dailyKey := fmt.Sprintf("proxy:%s:stats:daily:%s", apiKey, now.Format("2006-01-02"))

	// Update minute stats
	// Update minute stats
	s.Client.Do(ctx, s.Client.B().Hincrby().Key(minuteKey).Field("prompt_tokens").Increment(int64(promptTokens)).Build())
	s.Client.Do(ctx, s.Client.B().Hincrby().Key(minuteKey).Field("completion_tokens").Increment(int64(completionTokens)).Build())
	s.Client.Do(ctx, s.Client.B().Expire().Key(minuteKey).Seconds(86400).Build()) // 24 hours

	// Update daily stats
	s.Client.Do(ctx, s.Client.B().Hincrby().Key(dailyKey).Field("total_requests").Increment(1).Build())
	s.Client.Do(ctx, s.Client.B().Hincrby().Key(dailyKey).Field("total_prompt_tokens").Increment(int64(promptTokens)).Build())
	s.Client.Do(ctx, s.Client.B().Hincrby().Key(dailyKey).Field("total_completion_tokens").Increment(int64(completionTokens)).Build())
	s.Client.Do(ctx, s.Client.B().Hincrby().Key(dailyKey).Field("total_tokens").Increment(int64(promptTokens+completionTokens)).Build())
	s.Client.Do(ctx, s.Client.B().Expire().Key(dailyKey).Seconds(604800).Build()) // 7 days

	return nil
}