# Cache Strategy

## Overview

Waugzee implements a comprehensive caching strategy to minimize database queries and improve response times for the `/users/me` endpoint and related user data operations. This document outlines the caching patterns, invalidation strategies, and design decisions.

## Valkey Database Index Organization

Waugzee uses Valkey (Redis-compatible) with **logical database separation** to organize caches by category. All user-related data is consolidated into **DB 2 (USER_CACHE_INDEX)** for simplicity and consistency.

### Database Index Assignments

| DB Index | Name | Purpose | Usage |
|----------|------|---------|-------|
| **0** | General | General purpose caching | Miscellaneous cache operations |
| **1** | Session | Session management | User sessions, authentication tokens |
| **2** | **User** | **All user-related data** | User profiles, folders, releases, history, styluses, recommendations, streaks |
| **3** | Events | Event-driven data | Event sourcing, notifications, real-time updates |
| **4** | ClientAPI | External API responses | Cached responses from Discogs and other external services |

### Why DB 2 for All User Data?

**Consolidation Benefits**:
- **Simplicity**: All user-related caches in one place
- **Easy Flushing**: Clear all user data with single `FLUSHDB` command on DB 2
- **Debuggability**: Inspect all user caches without switching DB indexes
- **Consistency**: No confusion about which DB index to use for user operations

**What's in DB 2**:
- User profiles and OIDC ID mappings (`user_oidc:{oidcUserID}`)
- User folders (`user_folders:{userID}`, `user_folder:{userID}:{folderID}`)
- User releases (`user_releases:{userID}:{folderID}`)
- Play history (`play_history:{userID}`)
- Cleaning history (`cleaning_history:{userID}`)
- Styluses (`user_styluses:{userID}`)
- Daily recommendations (`daily_recommendations:{userID}`, `recent_recommendation:{userID}`)
- User streaks (`user_streak:{userID}`)

## Caching Pattern

All cached repository methods follow a consistent **cache-first pattern**:

1. **Check cache** for requested data
2. If **cache miss**, query database
3. **Update cache** with fresh data
4. **Clear cache** when data is modified

This pattern is implemented using the `CacheBuilder` utility with `WithHash()` for cache key management.

## Cache Inventory

### User Data Caches

| Cache Key | Repository Method | TTL | Cleared On |
|-----------|------------------|-----|------------|
| `user_oidc:{oidcUserID}` | `GetByOIDCUserID` | 7 days | User updates, config updates |
| `user_folders:{userID}` | `GetUserFolders` | 7 days | Folder upsert, folder deletion |
| `user_folder:{userID}:{folderID}` | `GetFolderByID` | 7 days | Folder upsert, folder deletion |
| `user_releases:{userID}:{folderID}` | `GetUserReleasesByFolderID` | 24 hours | Release mutations, play/clean history mutations |
| `user_styluses:{userID}` | `GetUserStyluses` | 7 days | Stylus create/update/delete |
| `play_history:{userID}` | `GetUserPlayHistory` | 24 hours | Play history mutations |
| `cleaning_history:{userID}` | `GetUserCleaningHistory` | 24 hours | Cleaning history mutations |
| `daily_recommendations:{userID}` | `GetTodayRecommendation` | 24 hours | Recommendation create, mark as listened |
| `recent_recommendation:{userID}` | `GetMostRecentRecommendation` | 24 hours | Recommendation create, mark as listened |
| `user_streak:{userID}` | `CalculateUserStreaks` | 24 hours | Play history mutations, mark recommendation as listened |

## Cache Invalidation Strategy

### Direct Invalidations

Each mutation method clears its own cache immediately after database modification:

```go
// Example: PlayHistory creation
func (r *historyRepository) CreatePlayHistory(...) error {
    // 1. Create in database
    err := gorm.G[PlayHistory](tx).Create(ctx, playHistory)

    // 2. Clear own cache
    r.clearUserPlayHistoryCache(ctx, playHistory.UserID)

    // 3. Clear dependent caches (cascade)
    r.clearAllUserReleasesCache(ctx, playHistory.UserID)
    r.clearUserStreakCache(ctx, playHistory.UserID)

    return nil
}
```

### Cascade Invalidations

Some mutations affect multiple caches due to data dependencies. The following cascade patterns are implemented:

#### PlayHistory Mutations → 3 Caches

**Cleared:**
- `play_history:{userID}` (direct)
- `user_releases:{userID}:*` (cascade - UserReleases preload PlayHistory)
- `user_streak:{userID}` (cascade - streaks calculated from play history)

**Triggers:**
- `CreatePlayHistory`
- `UpdatePlayHistory`
- `DeletePlayHistory`

**Rationale:** UserReleases include PlayHistory data in preloads. Streaks are calculated based on PlayHistory records.

#### CleaningHistory Mutations → 2 Caches

**Cleared:**
- `cleaning_history:{userID}` (direct)
- `user_releases:{userID}:*` (cascade - UserReleases preload CleaningHistory)

**Triggers:**
- `CreateCleaningHistory`
- `UpdateCleaningHistory`
- `DeleteCleaningHistory`

**Rationale:** UserReleases include CleaningHistory data in preloads.

#### DailyRecommendation.MarkAsListened → 3 Caches

**Cleared:**
- `daily_recommendations:{userID}` (direct)
- `recent_recommendation:{userID}` (related)
- `user_streak:{userID}` (cascade - streaks depend on listened_at)

**Trigger:**
- `MarkAsListened`

**Rationale:** Marking a recommendation as listened affects streak calculations.

#### UserConfiguration Mutations → 1 Cache

**Cleared:**
- `user_oidc:{oidcUserID}` (User includes Configuration in preload)

**Triggers:**
- `Update`
- `CreateOrUpdate`

**Rationale:** User objects preload Configuration data.

#### Folder Mutations → 2 Caches

**Cleared:**
- `user_folders:{userID}` (list cache)
- `user_folder:{userID}:{folderID}` (individual folder caches)

**Triggers:**
- `UpsertFolders`
- `DeleteOrphanFolders`

**Rationale:** Keep list and individual folder caches in sync.

#### UserRelease Mutations → 1 Cache

**Cleared:**
- `user_releases:{userID}:{folderID}` (specific folder)
- OR `user_releases:{userID}:*` (all folders, on delete)

**Triggers:**
- `CreateBatch` (clears specific folder)
- `UpdateBatch` (clears specific folder)
- `DeleteBatch` (clears all folder caches)

**Rationale:** DeleteBatch doesn't specify folder, so clear all variations to ensure consistency.

## TTL Rationale

### 7-Day TTL
Used for relatively static data that changes infrequently:
- User data
- Folders
- Styluses

**Reasoning:** These change only when user explicitly updates them. Long TTL reduces database load while maintaining reasonable freshness.

### 24-Hour TTL
Used for dynamic data that changes frequently:
- User releases (includes play/clean history)
- Play history
- Cleaning history
- Recommendations
- Streaks

**Reasoning:** Balances cache hit rate with data freshness. Even if invalidation misses, data refreshes within a day.

## Cache Pattern Limitations

### User Releases Wildcard Pattern

**Challenge:** When PlayHistory or CleaningHistory changes, we need to clear ALL `user_releases:{userID}:*` caches (one per folder) because we don't know which folder the affected release belongs to.

**Current Solution:** Log a warning but accept eventual consistency. The 24-hour TTL ensures stale data doesn't persist long.

```go
func (r *historyRepository) clearAllUserReleasesCache(ctx context.Context, userID uuid.UUID) {
    cachePattern := fmt.Sprintf("%s:*", userID.String())
    r.log.Debug("clearing all user_releases caches", "userID", userID, "pattern", cachePattern)
    // Note: Pattern-based cache clearing not yet implemented
    // Relying on 24-hour TTL for eventual consistency
}
```

**Future Improvement:** Implement pattern-based cache deletion in the cache layer if needed.

## Client-Server Cache Flow

### Current Flow (Recommended)

```
┌─────────┐                    ┌─────────┐                    ┌──────────┐
│ Client  │                    │ Server  │                    │ Database │
└────┬────┘                    └────┬────┘                    └────┬─────┘
     │                              │                              │
     │  Mutation Request            │                              │
     │─────────────────────────────>│                              │
     │                              │                              │
     │                              │  Execute DB Mutation         │
     │                              │─────────────────────────────>│
     │                              │                              │
     │                              │  Invalidate Server Caches    │
     │                              │──────────┐                   │
     │                              │          │                   │
     │                              │<─────────┘                   │
     │                              │                              │
     │  Success Response            │                              │
     │<─────────────────────────────│                              │
     │                              │                              │
     │  Invalidate TanStack Cache   │                              │
     │──────────┐                   │                              │
     │          │                   │                              │
     │<─────────┘                   │                              │
     │                              │                              │
     │  Refetch /users/me           │                              │
     │─────────────────────────────>│                              │
     │                              │                              │
     │                              │  Check Server Cache (MISS)   │
     │                              │──────────┐                   │
     │                              │          │                   │
     │                              │<─────────┘                   │
     │                              │                              │
     │                              │  Query Database              │
     │                              │─────────────────────────────>│
     │                              │                              │
     │                              │  Update Server Cache         │
     │                              │──────────┐                   │
     │                              │          │                   │
     │                              │<─────────┘                   │
     │                              │                              │
     │  Fresh Data                  │                              │
     │<─────────────────────────────│                              │
     │                              │                              │
```

### Why This Pattern?

1. **Simplicity**: Easy to understand and debug
2. **TanStack Query Integration**: Works perfectly with client-side cache management
3. **Flexibility**: Client controls when to refetch (can batch mutations)
4. **Correctness**: No risk of returning stale data
5. **Clean Separation**: Server manages server cache, client manages client cache

### Alternative Patterns Considered

#### Query & Cache Before Return
**Rejected:** Slows down mutations, creates over-fetching, doesn't align with TanStack Query patterns.

#### Async Cache Warming (Goroutine)
**Rejected:** Race conditions, unpredictable timing, complex error handling, minimal benefit.

## Performance Metrics

### Cache Hit Rate Goals

- **User data (7-day TTL)**: 95%+ hit rate expected
- **Dynamic data (24-hour TTL)**: 70-80% hit rate expected for active users

### `/users/me` Endpoint

**Without Caching:**
- 8-10 database queries per request
- ~100-150ms response time

**With Caching (cache hit):**
- 0 database queries
- ~10-20ms response time

**With Caching (cache miss):**
- 8-10 database queries
- ~100-150ms response time (plus cache warming overhead)

### Play Logging Flow

**Most Common User Action:** Logging a play

**Cache Invalidations:**
- 3 cache clears (play_history, user_releases pattern, streak)
- Minimal performance impact (<1ms for cache deletions)
- Next `/users/me` request: cache miss, fresh data from DB

**Optimization:** The play logging mutation itself is fast. The cache miss on next request is acceptable given the low latency of the refetch.

## Best Practices

### Do's

✅ **Always use `CacheBuilder` with `WithHash()`** - Never manually construct cache keys
✅ **Clear caches immediately after DB mutations** - Don't defer to avoid missed invalidations
✅ **Clear dependent caches** - Follow cascade invalidation patterns
✅ **Log cache operations** - Use Debug level for cache hits, Warn for failures
✅ **Set appropriate TTLs** - Use documented TTL patterns

### Don'ts

❌ **Don't manually concatenate cache keys** - Use CacheBuilder
❌ **Don't skip cache invalidation** - Always clear affected caches
❌ **Don't over-invalidate** - Only clear what's actually affected
❌ **Don't cache errors** - Only cache successful DB queries
❌ **Don't implement cache warming in mutations** - Keep mutations fast

## Monitoring & Debugging

### Cache Hit Rate Logging

All cache operations log at Debug level:
```
[DEBUG] user_releases retrieved from cache userID=... folderID=... count=...
[DEBUG] clearing all user_releases caches userID=... pattern=...
```

### Cache Misses

Cache misses result in INFO level logs:
```
[INFO] User releases retrieved from database and cached userID=... folderID=... count=...
```

### Common Issues

**Stale data after mutation:**
- Check if cascade invalidation is implemented
- Verify cache is being cleared in mutation method
- Check TTL hasn't expired between mutation and refetch

**High database load:**
- Check cache hit rate logs
- Verify TTLs are appropriate
- Look for missing cache implementations

## Future Improvements

1. **Pattern-based cache deletion**: Implement wildcard cache clearing for `user_releases:{userID}:*`
2. **Cache metrics**: Add prometheus metrics for cache hit/miss rates
3. **Selective cache warming**: Strategic cache warming for specific high-traffic scenarios
4. **Cache compression**: Compress large cached objects (like user_releases with preloads)

## Related Documentation

- [API Implementation Guide](API_IMPLEMENTATION_GUIDE.md)
- [Project Plan](PROJECT_PLAN.md)
- [CLAUDE.md](../CLAUDE.md) - Backend development standards
