package services

import (
	"context"
	"errors"
	"fmt"
	"time"
	"waugzee/internal/logger"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

const (
	DISCOGS_RATE_LIMIT_HASH = "discogs_rate_limit:%s" // %s = userID
	DISCOGS_RATE_LIMIT      = 50
	DISCOGS_RATE_WINDOW     = 60 * time.Second // 60 second window

	// Throttling configuration
	THROTTLE_THRESHOLD_MEDIUM = 0.5             // 50% capacity - start light throttling
	THROTTLE_THRESHOLD_HIGH   = 0.75            // 75% capacity - increase throttling
	THROTTLE_DELAY_MEDIUM     = 1 * time.Second // 50-75% capacity delay
	THROTTLE_DELAY_HIGH       = 2 * time.Second // 75-100% capacity delay
)

type DiscogsRateLimiterService struct {
	log   logger.Logger
	cache valkey.Client
}

func NewDiscogsRateLimiterService(cache valkey.Client) *DiscogsRateLimiterService {
	log := logger.New("DiscogsRateLimiterService")
	return &DiscogsRateLimiterService{
		log:   log,
		cache: cache,
	}
}

// CheckUserRateLimit checks if the user can make a Discogs API request
// If rate limited, it will sleep and retry until a slot becomes available
func (d *DiscogsRateLimiterService) CheckUserRateLimit(
	ctx context.Context,
	userID uuid.UUID,
) error {
	log := d.log.Function("CheckUserRateLimit")

	// Check if context has a reasonable deadline
	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) < 5*time.Second {
			err := errors.New("insufficient time remaining for rate limit processing")
			return log.Err(
				"insufficient time remaining for rate limit processing",
				err,
				"userID",
				userID,
				"timeRemaining",
				time.Until(deadline),
			)
		}
	}

	for {
		// Check current rate limit status
		canProceed, _, err := d.checkAndAddRequest(ctx, userID)
		if err != nil {
			return log.Err("failed to check rate limit", err, "userID", userID)
		}

		if canProceed {
			log.Info("Rate limit check passed", "userID", userID)
			return nil
		}

		// Rate limited - calculate when next slot becomes available
		retryAfter, err := d.calculateNextSlotAvailable(ctx, userID)
		if err != nil {
			return log.Err("failed to calculate retry delay", err, "userID", userID)
		}

		log.Info("Rate limited, waiting for next slot",
			"userID", userID,
			"retryAfter", retryAfter)

		select {
		case <-ctx.Done():
			return log.Err(
				"context cancelled while waiting for rate limit",
				ctx.Err(),
				"userID",
				userID,
			)
		case <-time.After(retryAfter):
			// Continue to next iteration
		}
	}
}

// CheckUserRateLimitWithThrottling checks rate limit with proactive throttling based on capacity usage
// Applies sliding scale delays: 0-50% (no delay), 50-75% (1s), 75-100% (2s)
func (d *DiscogsRateLimiterService) CheckUserRateLimitWithThrottling(
	ctx context.Context,
	userID uuid.UUID,
) error {
	log := d.log.Function("CheckUserRateLimitWithThrottling")

	// Check if context has a reasonable deadline
	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) < 5*time.Second {
			err := errors.New("insufficient time remaining for rate limit processing")
			return log.Err(
				"insufficient time remaining for rate limit processing",
				err,
				"userID",
				userID,
				"timeRemaining",
				time.Until(deadline),
			)
		}
	}

	for {
		// Check current rate limit status and get current count
		canProceed, currentCount, err := d.checkAndAddRequest(ctx, userID)
		if err != nil {
			return log.Err("failed to check rate limit", err, "userID", userID)
		}

		if canProceed {
			log.Info(
				"Rate limit check passed with throttling",
				"userID",
				userID,
				"currentCount",
				currentCount,
				"capacity",
				fmt.Sprintf("%.1f%%", float64(currentCount)/float64(DISCOGS_RATE_LIMIT)*100),
			)
			return nil
		}

		// Apply proactive throttling delay based on current capacity
		throttleDelay := d.calculateThrottleDelay(currentCount)
		if throttleDelay > 0 {
			log.Info(
				"Applying proactive throttling delay",
				"userID",
				userID,
				"currentCount",
				currentCount,
				"capacity",
				fmt.Sprintf("%.1f%%", float64(currentCount)/float64(DISCOGS_RATE_LIMIT)*100),
				"throttleDelay",
				throttleDelay,
			)

			select {
			case <-ctx.Done():
				return log.Err(
					"context cancelled during throttling delay",
					ctx.Err(),
					"userID",
					userID,
				)
			case <-time.After(throttleDelay):
				// Continue after throttling delay
			}
		}

		// Rate limited - calculate when next slot becomes available
		retryAfter, err := d.calculateNextSlotAvailable(ctx, userID)
		if err != nil {
			return log.Err("failed to calculate retry delay", err, "userID", userID)
		}

		log.Info("Rate limited, waiting for next slot",
			"userID", userID,
			"retryAfter", retryAfter)

		select {
		case <-ctx.Done():
			return log.Err(
				"context cancelled while waiting for rate limit",
				ctx.Err(),
				"userID",
				userID,
			)
		case <-time.After(retryAfter):
			// Continue to next iteration
		}
	}
}

// checkAndAddRequest checks if user is under rate limit and adds a request if so
// Returns (canProceed, currentCount, error)
func (d *DiscogsRateLimiterService) checkAndAddRequest(
	ctx context.Context,
	userID uuid.UUID,
) (bool, int64, error) {
	log := d.log.Function("checkAndAddRequest")
	key := fmt.Sprintf(DISCOGS_RATE_LIMIT_HASH, userID.String())
	now := time.Now().Unix()
	windowStart := now - int64(DISCOGS_RATE_WINDOW.Seconds())

	// 1. Clean up expired entries using sorted set operations
	err := d.cache.Do(ctx, d.cache.B().Zremrangebyscore().Key(key).Min("-inf").Max(fmt.Sprintf("%d", windowStart)).Build()).
		Error()
	if err != nil {
		return false, 0, log.Err(
			"failed to clean up expired rate limit entries",
			err,
			"userID",
			userID,
		)
	}

	// 2. Check current count using sorted set cardinality
	count, err := d.cache.Do(ctx, d.cache.B().Zcard().Key(key).Build()).AsInt64()
	if err != nil {
		return false, 0, log.Err("failed to get current rate limit count", err, "userID", userID)
	}

	if count >= DISCOGS_RATE_LIMIT {
		log.Info("User rate limited",
			"userID", userID,
			"currentRequests", count,
			"limit", DISCOGS_RATE_LIMIT)
		return false, count, nil
	}

	// 3. Add new request with current timestamp as score
	requestID := uuid.New().String()
	err = d.cache.Do(ctx, d.cache.B().Zadd().Key(key).ScoreMember().ScoreMember(float64(now), requestID).Build()).
		Error()
	if err != nil {
		return false, count, log.Err("failed to add request to rate limit tracker", err,
			"userID", userID,
			"requestID", requestID)
	}

	// Optional: Set key expiry for cleanup when unused (prevents memory buildup)
	d.cache.Do(
		ctx,
		d.cache.B().Expire().Key(key).Seconds(int64(DISCOGS_RATE_WINDOW.Seconds()*2)).Build(),
	)

	log.Info("Added request to rate limit tracker",
		"userID", userID,
		"requestID", requestID,
		"newCount", count+1,
		"limit", DISCOGS_RATE_LIMIT)

	return true, count + 1, nil
}

// calculateNextSlotAvailable calculates when the next rate limit slot becomes available
func (d *DiscogsRateLimiterService) calculateNextSlotAvailable(
	ctx context.Context,
	userID uuid.UUID,
) (time.Duration, error) {
	key := fmt.Sprintf(DISCOGS_RATE_LIMIT_HASH, userID.String())

	// Get the oldest entry in the sorted set (earliest timestamp)
	result, err := d.cache.Do(ctx, d.cache.B().Zrange().Key(key).Min("0").Max("0").Withscores().Build()).
		AsZScores()
	if err != nil {
		return time.Second, err // Fallback to 1 second on error
	}

	if len(result) == 0 {
		return 0, nil // No entries, slot should be available immediately
	}

	// Calculate when the oldest entry expires
	oldestTimestamp := time.Unix(int64(result[0].Score), 0)
	expiresAt := oldestTimestamp.Add(DISCOGS_RATE_WINDOW)
	now := time.Now()

	if expiresAt.Before(now) {
		return 0, nil // Should be available immediately
	}

	// Return time until oldest entry expires, plus small buffer
	retryAfter := expiresAt.Sub(now) + (100 * time.Millisecond)

	// Cap the retry time to prevent excessive waiting
	maxWait := 30 * time.Second
	if retryAfter > maxWait {
		retryAfter = maxWait
	}

	return retryAfter, nil
}

// calculateThrottleDelay calculates proactive throttling delay based on current capacity usage
func (d *DiscogsRateLimiterService) calculateThrottleDelay(currentCount int64) time.Duration {
	capacityUsed := float64(currentCount) / float64(DISCOGS_RATE_LIMIT)

	switch {
	case capacityUsed < THROTTLE_THRESHOLD_MEDIUM: // 0-50%: No throttling
		return 0
	case capacityUsed < THROTTLE_THRESHOLD_HIGH: // 50-75%: Light throttling
		return THROTTLE_DELAY_MEDIUM
	default: // 75-100%: Moderate throttling
		return THROTTLE_DELAY_HIGH
	}
}

// GetUserRateLimitStatus returns the current rate limit status for a user
func (d *DiscogsRateLimiterService) GetUserRateLimitStatus(
	ctx context.Context,
	userID uuid.UUID,
) (int, int, error) {
	key := fmt.Sprintf(DISCOGS_RATE_LIMIT_HASH, userID.String())
	now := time.Now().Unix()
	windowStart := now - int64(DISCOGS_RATE_WINDOW.Seconds())

	// Clean up expired entries first
	err := d.cache.Do(ctx, d.cache.B().Zremrangebyscore().Key(key).Min("-inf").Max(fmt.Sprintf("%d", windowStart)).Build()).
		Error()
	if err != nil {
		return 0, 0, err
	}

	// Get current count using sorted set cardinality
	count, err := d.cache.Do(ctx, d.cache.B().Zcard().Key(key).Build()).AsInt64()
	if err != nil {
		return 0, 0, err
	}

	return int(count), DISCOGS_RATE_LIMIT, nil
}
