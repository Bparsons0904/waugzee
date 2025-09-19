package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"

	"github.com/google/uuid"
)

type DiscogsRateLimitService interface {
	UpdateRateLimit(ctx context.Context, userID uuid.UUID, headers map[string]string) error
	GetRateLimit(ctx context.Context, userID uuid.UUID) (*RateLimit, error)
	CanMakeRequest(ctx context.Context, userID uuid.UUID) (bool, time.Duration, error)
	CalculateRequestDelay(ctx context.Context, userID uuid.UUID) time.Duration
	ResetUserRateLimit(ctx context.Context, userID uuid.UUID) error
}

type RateLimit struct {
	UserID           uuid.UUID     `json:"userId"`
	Remaining        int           `json:"remaining"`
	Limit            int           `json:"limit"`
	WindowReset      time.Time     `json:"windowReset"`
	LastUpdated      time.Time     `json:"lastUpdated"`
	RecommendedDelay time.Duration `json:"recommendedDelay"`
}

type discogsRateLimitService struct {
	cache          database.CacheClient
	log            logger.Logger
	defaultLimit   int
	defaultWindow  time.Duration
	safetyBuffer   int           // Number of requests to keep as buffer
	minDelay       time.Duration // Minimum delay between requests
	maxDelay       time.Duration // Maximum delay when approaching limits
}

func NewDiscogsRateLimitService(cache database.CacheClient) DiscogsRateLimitService {
	return &discogsRateLimitService{
		cache:         cache,
		log:           logger.New("DiscogsRateLimitService"),
		defaultLimit:  60,              // Discogs default: 60 requests per minute
		defaultWindow: time.Minute,     // 1 minute window
		safetyBuffer:  5,               // Keep 5 requests as buffer
		minDelay:      1 * time.Second, // Minimum 1 second between requests
		maxDelay:      30 * time.Second, // Maximum 30 seconds delay
	}
}

func (s *discogsRateLimitService) UpdateRateLimit(ctx context.Context, userID uuid.UUID, headers map[string]string) error {
	log := s.log.Function("UpdateRateLimit")

	rateLimit := &RateLimit{
		UserID:      userID,
		LastUpdated: time.Now(),
	}

	// Parse rate limit headers from Discogs API response
	if remaining, exists := headers["x-discogs-ratelimit-remaining"]; exists {
		if val, err := strconv.Atoi(remaining); err == nil {
			rateLimit.Remaining = val
		}
	}

	if limit, exists := headers["x-discogs-ratelimit-limit"]; exists {
		if val, err := strconv.Atoi(limit); err == nil {
			rateLimit.Limit = val
		}
	}

	// Parse window reset time
	if window, exists := headers["x-discogs-ratelimit-window"]; exists {
		if seconds, err := strconv.Atoi(window); err == nil {
			rateLimit.WindowReset = time.Now().Add(time.Duration(seconds) * time.Second)
		}
	}

	// Set defaults if headers are missing
	if rateLimit.Limit == 0 {
		rateLimit.Limit = s.defaultLimit
	}
	if rateLimit.WindowReset.IsZero() {
		rateLimit.WindowReset = time.Now().Add(s.defaultWindow)
	}

	// Calculate recommended delay
	rateLimit.RecommendedDelay = s.calculateDelay(rateLimit)

	// Store in cache
	cacheKey := s.getRateLimitCacheKey(userID)
	rateLimitJSON, err := json.Marshal(rateLimit)
	if err != nil {
		return fmt.Errorf("failed to marshal rate limit: %w", err)
	}

	// Cache until window reset
	ttl := time.Until(rateLimit.WindowReset)
	if ttl < 0 {
		ttl = s.defaultWindow
	}

	if err := s.cache.Set(ctx, cacheKey, string(rateLimitJSON), ttl); err != nil {
		return fmt.Errorf("failed to cache rate limit: %w", err)
	}

	log.Info("Rate limit updated",
		"userID", userID,
		"remaining", rateLimit.Remaining,
		"limit", rateLimit.Limit,
		"windowReset", rateLimit.WindowReset,
		"recommendedDelay", rateLimit.RecommendedDelay,
	)

	return nil
}

func (s *discogsRateLimitService) GetRateLimit(ctx context.Context, userID uuid.UUID) (*RateLimit, error) {
	cacheKey := s.getRateLimitCacheKey(userID)

	rateLimitJSON, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		// Return default rate limit if not found
		return &RateLimit{
			UserID:           userID,
			Remaining:        s.defaultLimit,
			Limit:            s.defaultLimit,
			WindowReset:      time.Now().Add(s.defaultWindow),
			LastUpdated:      time.Now(),
			RecommendedDelay: s.minDelay,
		}, nil
	}

	var rateLimit RateLimit
	if err := json.Unmarshal([]byte(rateLimitJSON), &rateLimit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rate limit: %w", err)
	}

	// Check if window has reset
	if time.Now().After(rateLimit.WindowReset) {
		// Reset the rate limit
		rateLimit.Remaining = rateLimit.Limit
		rateLimit.WindowReset = time.Now().Add(s.defaultWindow)
		rateLimit.RecommendedDelay = s.minDelay

		// Update cache
		if err := s.UpdateRateLimit(ctx, userID, map[string]string{}); err != nil {
			s.log.Warn("Failed to update reset rate limit", "error", err)
		}
	}

	return &rateLimit, nil
}

func (s *discogsRateLimitService) CanMakeRequest(ctx context.Context, userID uuid.UUID) (bool, time.Duration, error) {
	rateLimit, err := s.GetRateLimit(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	// Check if we have enough requests remaining (including safety buffer)
	if rateLimit.Remaining <= s.safetyBuffer {
		// Calculate time until window reset
		delay := time.Until(rateLimit.WindowReset)
		if delay < 0 {
			delay = 0
		}
		return false, delay, nil
	}

	return true, 0, nil
}

func (s *discogsRateLimitService) CalculateRequestDelay(ctx context.Context, userID uuid.UUID) time.Duration {
	rateLimit, err := s.GetRateLimit(ctx, userID)
	if err != nil {
		return s.minDelay
	}

	return s.calculateDelay(rateLimit)
}

func (s *discogsRateLimitService) ResetUserRateLimit(ctx context.Context, userID uuid.UUID) error {
	cacheKey := s.getRateLimitCacheKey(userID)
	return s.cache.Delete(ctx, cacheKey)
}

// Helper methods

func (s *discogsRateLimitService) getRateLimitCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("discogs_rate_limit:%s", userID.String())
}

func (s *discogsRateLimitService) calculateDelay(rateLimit *RateLimit) time.Duration {
	if rateLimit.Remaining <= 0 {
		// If no requests remaining, wait until window reset
		return time.Until(rateLimit.WindowReset)
	}

	// Calculate percentage of requests used
	usedPercentage := float64(rateLimit.Limit-rateLimit.Remaining) / float64(rateLimit.Limit)

	// Progressive delay based on usage
	var delay time.Duration

	switch {
	case usedPercentage < 0.5: // Less than 50% used
		delay = s.minDelay
	case usedPercentage < 0.7: // 50-70% used
		delay = time.Duration(float64(s.minDelay) * 1.5)
	case usedPercentage < 0.85: // 70-85% used
		delay = time.Duration(float64(s.minDelay) * 2.5)
	case usedPercentage < 0.95: // 85-95% used
		delay = time.Duration(float64(s.minDelay) * 5)
	default: // 95%+ used
		delay = s.maxDelay
	}

	// Ensure delay doesn't exceed maximum
	if delay > s.maxDelay {
		delay = s.maxDelay
	}

	// If approaching window reset, distribute remaining requests
	timeUntilReset := time.Until(rateLimit.WindowReset)
	if timeUntilReset > 0 && rateLimit.Remaining > 0 {
		avgDelay := timeUntilReset / time.Duration(rateLimit.Remaining)
		if avgDelay > delay {
			delay = avgDelay
		}
	}

	return delay
}

// Rate limit tracking for concurrent requests
type RequestTracker struct {
	service DiscogsRateLimitService
	userID  uuid.UUID
}

func NewRequestTracker(service DiscogsRateLimitService, userID uuid.UUID) *RequestTracker {
	return &RequestTracker{
		service: service,
		userID:  userID,
	}
}

func (rt *RequestTracker) BeforeRequest(ctx context.Context) error {
	// Decrement available requests (optimistic)
	rateLimit, err := rt.service.GetRateLimit(ctx, rt.userID)
	if err != nil {
		return err
	}

	rateLimit.Remaining--
	headers := map[string]string{
		"x-discogs-ratelimit-remaining": strconv.Itoa(rateLimit.Remaining),
		"x-discogs-ratelimit-limit":     strconv.Itoa(rateLimit.Limit),
	}

	return rt.service.UpdateRateLimit(ctx, rt.userID, headers)
}

func (rt *RequestTracker) AfterRequest(ctx context.Context, responseHeaders map[string]string) error {
	// Update with actual rate limit from response
	return rt.service.UpdateRateLimit(ctx, rt.userID, responseHeaders)
}