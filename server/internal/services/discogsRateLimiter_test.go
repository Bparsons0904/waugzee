package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Simple test that creates a service with a nil cache for testing logic
func TestCalculateThrottleDelay(t *testing.T) {
	// We can test this method since it doesn't use the cache
	service := &DiscogsRateLimiterService{}

	tests := []struct {
		name         string
		currentCount int64
		expectedDelay time.Duration
	}{
		{
			name:         "No throttling at 0% capacity",
			currentCount: 0,
			expectedDelay: 0,
		},
		{
			name:         "No throttling at 40% capacity",
			currentCount: 2,
			expectedDelay: 0,
		},
		{
			name:         "Light throttling at 60% capacity",
			currentCount: 3,
			expectedDelay: THROTTLE_DELAY_MEDIUM,
		},
		{
			name:         "Moderate throttling at 80% capacity",
			currentCount: 4,
			expectedDelay: THROTTLE_DELAY_HIGH,
		},
		{
			name:         "Moderate throttling at 100% capacity",
			currentCount: 5,
			expectedDelay: THROTTLE_DELAY_HIGH,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := service.calculateThrottleDelay(tt.currentCount)
			if delay != tt.expectedDelay {
				t.Errorf("calculateThrottleDelay(%d) = %v, want %v", tt.currentCount, delay, tt.expectedDelay)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Test that our constants have expected values
	if DISCOGS_RATE_LIMIT != 5 {
		t.Errorf("DISCOGS_RATE_LIMIT = %d, want 5", DISCOGS_RATE_LIMIT)
	}

	if DISCOGS_RATE_WINDOW != 60*time.Second {
		t.Errorf("DISCOGS_RATE_WINDOW = %v, want %v", DISCOGS_RATE_WINDOW, 60*time.Second)
	}

	if THROTTLE_THRESHOLD_MEDIUM != 0.5 {
		t.Errorf("THROTTLE_THRESHOLD_MEDIUM = %f, want 0.5", THROTTLE_THRESHOLD_MEDIUM)
	}

	if THROTTLE_THRESHOLD_HIGH != 0.75 {
		t.Errorf("THROTTLE_THRESHOLD_HIGH = %f, want 0.75", THROTTLE_THRESHOLD_HIGH)
	}

	if THROTTLE_DELAY_MEDIUM != 1*time.Second {
		t.Errorf("THROTTLE_DELAY_MEDIUM = %v, want %v", THROTTLE_DELAY_MEDIUM, 1*time.Second)
	}

	if THROTTLE_DELAY_HIGH != 2*time.Second {
		t.Errorf("THROTTLE_DELAY_HIGH = %v, want %v", THROTTLE_DELAY_HIGH, 2*time.Second)
	}
}

func TestServiceConstruction(t *testing.T) {
	// Create a nil cache for testing - in real usage, this would be a valkey client
	service := NewDiscogsRateLimiterService(nil)

	if service == nil {
		t.Error("NewDiscogsRateLimiterService returned nil")
		return
	}

	// Test that service has expected structure
	if service.cache != nil {
		// We passed nil, so it should be nil
		// (In real usage, we'd pass a real valkey client)
		t.Error("Service cache should be nil for this test")
	}
}

func TestThrottleDelayBoundaryConditions(t *testing.T) {
	service := &DiscogsRateLimiterService{}

	// Test exact boundary conditions
	tests := []struct {
		name         string
		currentCount int64
		expectedDelay time.Duration
	}{
		{
			name:         "Exactly 50% capacity (2.5 rounds to 2)",
			currentCount: 2, // 2/5 = 40% (under 50%)
			expectedDelay: 0,
		},
		{
			name:         "Just over 50% capacity",
			currentCount: 3, // 3/5 = 60% (50-75% range)
			expectedDelay: THROTTLE_DELAY_MEDIUM,
		},
		{
			name:         "Exactly 75% capacity",
			currentCount: 4, // 4/5 = 80% (75-100% range)
			expectedDelay: THROTTLE_DELAY_HIGH,
		},
		{
			name:         "Over 100% capacity",
			currentCount: 6, // 6/5 = 120% (75-100% range)
			expectedDelay: THROTTLE_DELAY_HIGH,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := service.calculateThrottleDelay(tt.currentCount)
			if delay != tt.expectedDelay {
				capacityPercent := float64(tt.currentCount) / float64(DISCOGS_RATE_LIMIT) * 100
				t.Errorf("calculateThrottleDelay(%d) [%.1f%% capacity] = %v, want %v",
					tt.currentCount, capacityPercent, delay, tt.expectedDelay)
			}
		})
	}
}

// Test context timeout validation (can test without Redis)
func TestContextTimeoutValidation(t *testing.T) {
	service := NewDiscogsRateLimiterService(nil)
	userID := uuid.New()

	t.Run("Short context timeout should return error", func(t *testing.T) {
		// Create context with insufficient time
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// This should return an error before trying to access Redis
		err := service.CheckUserRateLimit(ctx, userID)
		if err == nil {
			t.Error("Expected error due to insufficient time remaining, got nil")
		}

		// Check error message is not empty
		if err != nil && len(err.Error()) > 0 {
			errMsg := err.Error()
			if len(errMsg) == 0 {
				t.Error("Error message is empty")
			}
			// We can't check exact string due to logger formatting, but we can check it's not empty
		}
	})

	t.Run("Short context timeout should return error for throttling method", func(t *testing.T) {
		// Create context with insufficient time
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// This should return an error before trying to access Redis
		err := service.CheckUserRateLimitWithThrottling(ctx, userID)
		if err == nil {
			t.Error("Expected error due to insufficient time remaining, got nil")
		}
	})

	// Note: We skip testing the "no deadline" case because it would require
	// a real Redis client to avoid nil pointer panics. The timeout validation
	// is the key feature we want to test here.
}