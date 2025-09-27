package constants

import "time"

// User cache constants
const (
	UserCachePrefix = "user_oidc:"               // Single cache by OIDC ID
	UserCacheExpiry = 7 * 24 * time.Hour        // 7 days
)