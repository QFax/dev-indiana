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

func (s *ValkeyService) CheckRateLimit(ctx context.Context, apiKey string, limitPerMinute, limitPerDay, tokensPerMinute int, window string) (bool, time.Time, error) {
	now := time.Now()
	nowUnix := now.Unix()

	// Per-minute request limit
	minuteKey := fmt.Sprintf("proxy:%s:requests:minute", apiKey)
	s.Client.Do(ctx, s.Client.B().Zremrangebyscore().Key(minuteKey).Min(fmt.Sprintf("%d", 0)).Max(fmt.Sprintf("%d", nowUnix-60)).Build())
	count, err := s.Client.Do(ctx, s.Client.B().Zcard().Key(minuteKey).Build()).AsInt64()
	if err != nil {
		return false, time.Time{}, err
	}
	if count >= int64(limitPerMinute) {
		oldest, err := s.Client.Do(ctx, s.Client.B().Zrange().Key(minuteKey).Min(0).Max(0).Build()).AsStrSlice()
		if err != nil || len(oldest) == 0 {
			return false, time.Time{}, err
		}
		oldestTimestamp, _ := time.Parse(time.RFC3339, oldest)
		return false, oldestTimestamp.Add(60 * time.Second), nil
	}

	// Per-day request limit
	dailyKey := fmt.Sprintf("proxy:%s:requests:day", apiKey)
	s.Client.Do(ctx, s.Client.B().Zremrangebyscore().Key(dailyKey).Min(fmt.Sprintf("%d", 0)).Max(fmt.Sprintf("%d", nowUnix-86400)).Build())
	count, err = s.Client.Do(ctx, s.Client.B().Zcard().Key(dailyKey).Build()).AsInt64()
	if err != nil {
		return false, time.Time{}, err
	}
	if count >= int64(limitPerDay) {
		oldest, err := s.Client.Do(ctx, s.Client.B().Zrange().Key(dailyKey).Min(0).Max(0).Build()).AsStrSlice()
		if err != nil || len(oldest) == 0 {
			return false, time.Time{}, err
		}
		oldestTimestamp, _ := time.Parse(time.RFC3339, oldest)
		return false, oldestTimestamp.Add(86400 * time.Second), nil
	}


	// Add current request to sets
	s.Client.Do(ctx, s.Client.B().Zadd().Key(minuteKey).ScoreMember().ScoreMember(float64(nowUnix), now.Format(time.RFC3339)).Build())
	s.Client.Do(ctx, s.Client.B().Zadd().Key(dailyKey).ScoreMember().ScoreMember(float64(nowUnix), now.Format(time.RFC3339)).Build())

	return true, time.Time{}, nil
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