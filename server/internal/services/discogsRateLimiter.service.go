package services

import (
	"context"
	"fmt"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

const (
	DISCOGS_RATE_LIMIT_HASH = "discogs_rate_limit:%s" // %s = userID
	DISCOGS_RATE_LIMIT      = 5                       // 60 requests per 60 seconds
	DISCOGS_RATE_WINDOW     = 60 * time.Second        // 60 second window
	RATE_LIMIT_CHECK_SLEEP  = 1 * time.Second         // Sleep duration when rate limited
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

	for {
		// Check current rate limit status
		canProceed, err := d.checkAndAddRequest(ctx, userID)
		if err != nil {
			return log.Err("failed to check rate limit", err, "userID", userID)
		}

		if canProceed {
			log.Info("Rate limit check passed", "userID", userID)
			return nil
		}

		// Rate limited - wait and retry
		log.Info("Rate limited, waiting before retry",
			"userID", userID,
			"sleepDuration", RATE_LIMIT_CHECK_SLEEP)

		select {
		case <-ctx.Done():
			return log.Err(
				"context cancelled while waiting for rate limit",
				ctx.Err(),
				"userID",
				userID,
			)
		case <-time.After(RATE_LIMIT_CHECK_SLEEP):
			// Continue to next iteration
		}
	}
}

// checkAndAddRequest checks if user is under rate limit and adds a request if so
// Returns true if request was added, false if rate limited
func (d *DiscogsRateLimiterService) checkAndAddRequest(
	ctx context.Context,
	userID uuid.UUID,
) (bool, error) {
	log := d.log.Function("checkAndAddRequest")

	// Get current set size
	setSize, err := d.getSetSize(ctx, userID)
	if err != nil {
		return false, log.Err("failed to get set size", err, "userID", userID)
	}

	// Check if under rate limit
	if setSize >= DISCOGS_RATE_LIMIT {
		log.Info("User rate limited",
			"userID", userID,
			"currentRequests", setSize,
			"limit", DISCOGS_RATE_LIMIT)
		return false, nil
	}

	// Add request to set with TTL
	requestID := uuid.New().String()
	err = database.NewCacheBuilder(d.cache, userID.String()).
		WithHashPattern(DISCOGS_RATE_LIMIT_HASH).
		WithMember(requestID).
		WithContext(ctx).
		SetSadd()
	if err != nil {
		return false, log.Err("failed to add request to rate limit set", err,
			"userID", userID,
			"requestID", requestID)
	}

	// Set TTL on the member (this creates a self-expiring record)
	err = d.setMemberTTL(ctx, userID, requestID)
	if err != nil {
		log.Warn("Failed to set TTL on rate limit record",
			"error", err,
			"userID", userID,
			"requestID", requestID)
		// Don't fail the request for TTL issues
	}

	log.Info("Added request to rate limit tracker",
		"userID", userID,
		"requestID", requestID,
		"newSetSize", setSize+1,
		"limit", DISCOGS_RATE_LIMIT)

	return true, nil
}

// getSetSize returns the current size of the user's rate limit set
func (d *DiscogsRateLimiterService) getSetSize(ctx context.Context, userID uuid.UUID) (int, error) {
	members, err := database.NewCacheBuilder(d.cache, userID.String()).
		WithHashPattern(DISCOGS_RATE_LIMIT_HASH).
		WithContext(ctx).
		GetSetMembers()
	if err != nil {
		return 0, err
	}
	return len(members), nil
}

// setMemberTTL sets TTL on individual set members
// Note: Redis sets don't support per-member TTL, so we use a workaround
// We create individual keys for each member with TTL and clean them up
func (d *DiscogsRateLimiterService) setMemberTTL(
	ctx context.Context,
	userID uuid.UUID,
	memberID string,
) error {
	// Create a separate key for TTL tracking using hash pattern
	ttlKey := fmt.Sprintf("%s:ttl:%s", userID.String(), memberID)

	err := database.NewCacheBuilder(d.cache, ttlKey).
		WithHashPattern(DISCOGS_RATE_LIMIT_HASH).
		WithValue("1").
		WithTTL(DISCOGS_RATE_WINDOW).
		WithContext(ctx).
		Set()
	if err != nil {
		return err
	}

	// Schedule cleanup of the set member when TTL expires
	// We'll use a separate goroutine to remove the member after TTL
	go func() {
		time.Sleep(DISCOGS_RATE_WINDOW)
		_ = database.NewCacheBuilder(d.cache, userID.String()).
			WithHashPattern(DISCOGS_RATE_LIMIT_HASH).
			WithMember(memberID).
			WithContext(context.Background()).
			RemoveSetMember()
	}()

	return nil
}

// GetUserRateLimitStatus returns the current rate limit status for a user
func (d *DiscogsRateLimiterService) GetUserRateLimitStatus(
	ctx context.Context,
	userID uuid.UUID,
) (int, int, error) {
	currentRequests, err := d.getSetSize(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	return currentRequests, DISCOGS_RATE_LIMIT, nil
}

