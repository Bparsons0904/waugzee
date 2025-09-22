# Release Sync Implementation Plan

## Overview
Build a release sync system that fetches user's vinyl releases from Discogs folders with pagination, manages missing release processing, and stores user collection data efficiently.

## Architecture Strategy

### 1. **Service Layer Design**
- **`releases.service.go`** - Handle release sync orchestration
- **`userRelease.repository.go`** - Manage user collection data (needs creation)
- **Sync State Management** - Track progress through folders and pagination

### 2. **Data Flow Design**

**Phase 1: Collection Sync**
1. User triggers release sync for selected folder(s)
2. For each folder, paginate through Discogs API: `/users/{username}/collection/folders/{folder_id}/releases`
3. Store `UserRelease` records immediately (user's collection items)
4. Queue missing `Release` records for later processing

**Phase 2: Release Processing**
1. Process queued release IDs to fetch full release data
2. Batch insert/update `Release` records
3. Update sync status and progress

### 3. **Storage Strategy**

**Immediate Storage (UserRelease)**
- Store user collection items as soon as we get folder contents
- Fields: UserID, ReleaseID (Discogs ID), InstanceID, FolderID, Rating, Notes
- This preserves user's collection data even if release processing fails

**Queued Processing (Releases)**
- Store missing release IDs in **Valkey cache** as sorted sets
- Key pattern: `release_sync:{userID}:pending`
- Allows resumable processing and batch operations

**Configuration Storage**
- Sync state and pagination cursors in cache
- User preferences for batch sizes and sync scope

### 4. **Key Components to Build**

#### A. **UserRelease Repository** (Missing)
```go
type UserReleaseRepository interface {
    UpsertBatch(ctx context.Context, tx *gorm.DB, userReleases []*UserRelease) error
    GetByUserAndFolder(ctx context.Context, tx *gorm.DB, userID uuid.UUID, folderID int) ([]*UserRelease, error)
    DeleteOrphansByFolder(ctx context.Context, tx *gorm.DB, userID uuid.UUID, folderID int, keepInstanceIDs []int) error
}
```

#### B. **Releases Service**
```go
type ReleasesService struct {
    // Core sync methods
    RequestFolderReleases(ctx context.Context, user *User, folderID int, page int) (string, error)
    ProcessReleasesResponse(ctx context.Context, metadata RequestMetadata, responseData map[string]any) error

    // Queue management
    QueueMissingReleases(ctx context.Context, userID uuid.UUID, releaseIDs []int64) error
    ProcessQueuedReleases(ctx context.Context, userID uuid.UUID, batchSize int) error
}
```

#### C. **Sync Configuration**
```go
type SyncConfig struct {
    BatchSize        int           // API pagination size (default: 50)
    MaxConcurrency   int           // Concurrent folder processing (default: 2)
    ProcessBatchSize int           // Release processing batch size (default: 20)
    ResumableSync    bool          // Allow partial sync resumption (default: true)
    SyncTimeoutMin   int           // Overall sync timeout (default: 30)
}
```

### 5. **Pagination & State Management**

**Pagination Pattern**
- Discogs API: `?page=1&per_page=50` (standard pattern)
- Store current page in cache: `sync_state:{userID}:{folderID}:page`
- Track completion status per folder

**Resumable Sync Design**
- Cache sync progress: which folders completed, current page per folder
- Allow users to resume interrupted syncs
- Clear state only on successful completion

### 6. **Error Handling & Recovery**

**Graceful Degradation**
- UserRelease storage succeeds even if Release processing fails
- Retry logic for failed release fetches
- Partial sync completion tracking

**Queue Management**
- TTL on pending release queues (24 hours)
- Duplicate detection in queues
- Progress reporting via WebSocket

### 7. **Integration Points**

**Orchestration Service Updates**
- Add `"releases"` case to request type handling
- Delegate to `ReleasesService.ProcessReleasesResponse()`
- Maintain same event-driven pattern as folders

**Frontend Integration**
- Progress tracking for multi-folder syncs
- Resume/cancel sync capabilities
- Real-time sync status updates

## Implementation Order

1. **Create UserRelease Repository** - Essential data layer ✅ **COMPLETED**
2. **Extend Folders Service** - Core sync logic with pagination support ✅ **COMPLETED**
3. **Integrate with Orchestration** - Wire into existing event system ✅ **COMPLETED**
4. **Implement Option 2: Merge Folders 1+ Strategy** ✅ **COMPLETED**
   - Updated UserConfiguration to use Discogs folder IDs directly
   - Created database migration for column type change
   - Enhanced UserRelease repository with Create/Update/Delete methods
   - Added collection sync orchestration with state management
   - Implemented merge and differential sync logic
   - Updated folder processing to accumulate instead of immediate writes
5. **Add Sync Configuration** - User preferences and batch control
6. **Implement Queue Processing** - Background release data fetching
7. **Add Progress Tracking** - WebSocket status updates

## Key Benefits

✅ **Immediate Collection Preservation** - UserRelease data stored first
✅ **Resumable Operations** - Handle large collections gracefully
✅ **Scalable Architecture** - Separate collection sync from release processing
✅ **Memory Efficient** - Cache-based queuing instead of in-memory storage
✅ **User Control** - Configurable batch sizes and sync scope

This approach prioritizes user collection data while allowing flexible release processing in the background.