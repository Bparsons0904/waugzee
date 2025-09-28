package constants

import "time"

const (
	UserCachePrefix        = "user_oidc"        // Single cache by OIDC ID (CacheBuilder adds colon)
	UserFoldersCachePrefix = "user_folders"     // User folders cache by userID (CacheBuilder adds colon)
	UserCacheExpiry        = 7 * 24 * time.Hour // 7 days
)

