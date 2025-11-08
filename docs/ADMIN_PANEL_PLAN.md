# Admin Panel Implementation Plan

## Overview

This document outlines the implementation plan for the Waugzee admin panel, a single-page dashboard for managing system operations, background jobs, users, and cache.

## Architecture Decision: Admin Authorization

### Recommended Approach: Database Column with Future Zitadel Integration Path

**Why this is a good balance:**
- ✅ **Security**: Database-driven authorization with middleware validation is industry-standard
- ✅ **Best Practice**: Follows existing codebase patterns (similar to `IsActive` field)
- ✅ **Flexibility**: Can add Zitadel role checking later without breaking changes
- ✅ **Simple Initial Setup**: Can manually set `is_admin=true` in database or create admin promotion endpoint

**Implementation:**
1. Use existing `User.IsAdmin` boolean field (already in model at `server/internal/models/user.model.go:17`)
2. Create `RequireAdmin` middleware that checks both auth + admin status
3. Add audit logging for all admin actions
4. **Future enhancement**: Add Zitadel role sync that updates `IsAdmin` field

## Phase 1: Core Admin Dashboard

### Dashboard Layout

Single dashboard page with collapsible sections:

```
┌─────────────────────────────────────┐
│ Admin Dashboard Header              │
├─────────────────────────────────────┤
│ 🔽 Monthly Downloads Status         │
│   - Current month processing        │
│   - Progress bars for each step     │
│   - [Reset] [Reprocess] buttons     │
│   - File checksums & sizes          │
├─────────────────────────────────────┤
│ 🔽 Background Jobs                  │
│   - Jobs table with schedules       │
│   - Next run times countdown        │
│   - [Trigger] [Pause/Resume] btns   │
├─────────────────────────────────────┤
│ 🔽 User Management                  │
│   - Users table with admin toggle   │
│   - Active/inactive status          │
│   - Quick stats (total, active)     │
├─────────────────────────────────────┤
│ 🔽 Cache Management                 │
│   - Stats by namespace              │
│   - Search keys by pattern          │
│   - Inspect/delete operations       │
│   - ⚠️ Dangerous bulk operations    │
└─────────────────────────────────────┘
```

### Priority Features (Based on User Feedback)

1. **Monthly Downloads Management** (HIGHEST PRIORITY)
2. **Background Jobs Control**
3. **User Management**
4. **Cache Management**

## Backend Implementation

### 1. Admin Middleware

**File**: `server/internal/handlers/middleware/admin.middleware.go`

**Purpose**: Validate admin access for protected routes

**Implementation**:
```go
func (m *Middleware) RequireAdmin() fiber.Handler {
    return func(c *fiber.Ctx) error {
        user := middleware.GetUser(c)
        if user == nil || !user.IsAdmin {
            return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
                "error": "Admin access required",
            })
        }
        return c.Next()
    }
}
```

**Usage**: Chain after `RequireAuth` for admin routes

### 2. Admin Handler

**File**: `server/internal/handlers/admin.handler.go`

#### Monthly Downloads Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/admin/downloads/status` | Get current processing status with detailed steps |
| POST | `/admin/downloads/reset` | Delete current processing record and trigger fresh download |
| POST | `/admin/downloads/reprocess` | Keep downloads, reset processing steps only |
| DELETE | `/admin/downloads/:yearMonth` | Delete specific month's processing record |

**User Requirement**:
- "Generally want to be able to restart. I almost think we delete the row and initiate a new processing. Either just triggering the reprocessing or redownload and reprocess."

**Implementation Notes**:
- **Reset**: Delete `DiscogsDataProcessing` record for current month, trigger download job
- **Reprocess**: Update status to `ready_for_processing`, reset processing stats, trigger parser job
- Both operations should validate no in-progress operations before proceeding

#### Background Jobs Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/admin/jobs` | List all scheduled jobs with next run times |
| POST | `/admin/jobs/:jobName/trigger` | Manually trigger job execution |
| POST | `/admin/jobs/pause` | Pause scheduler |
| POST | `/admin/jobs/resume` | Resume scheduler |

**Available Jobs** (from `server/internal/jobs/`):
- `DiscogsDownloadJob`: Downloads monthly XML dumps
- `DiscogsXMLParserJob`: Processes downloaded XML files

#### User Management Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/admin/users` | List all users with admin status, activity |
| PATCH | `/admin/users/:id/admin` | Toggle admin status |
| PATCH | `/admin/users/:id/active` | Activate/deactivate user |

#### Cache Management Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/admin/cache/stats` | Cache statistics by namespace |
| GET | `/admin/cache/keys/:pattern` | List keys matching pattern (URL-encoded) |
| GET | `/admin/cache/value/:key` | Inspect specific cache value |
| DELETE | `/admin/cache/key/:key` | Delete specific cache key (with confirmation) |
| DELETE | `/admin/cache/pattern/:pattern` | Bulk delete by pattern (DANGEROUS) |

**Cache Namespaces** (from `server/internal/services/constants.go`):
- `api_request` - API request metadata
- `collection_sync` - Collection sync state
- `release_queue` - Release processing queue
- `discogs_rate_limit:%s` - Per-user rate limiting

**CRITICAL**: Must use CacheBuilder pattern for all cache operations:
```go
// ✅ CORRECT
database.NewCacheBuilder(cache, identifier).
    WithContext(ctx).
    WithHash(constants.SomeCachePrefix).
    Get(&result)

// ❌ FORBIDDEN
cacheKey := constants.SomeCachePrefix + identifier
```

### 3. Admin Service

**File**: `server/internal/services/admin.service.go`

**Purpose**: Encapsulate admin business logic

**Key Functions**:
```go
type AdminService struct {
    db                    *gorm.DB
    cache                 *database.ValkeyClient
    downloadService       *DownloadService
    xmlProcessingService  *SimplifiedXMLProcessingService
    schedulerService      *SchedulerService
    userRepo             *repositories.UserRepository
    processingRepo       *repositories.DiscogsDataProcessingRepository
    auditLogRepo         *repositories.AdminActionLogRepository
}

// Monthly Downloads
func (s *AdminService) GetDownloadStatus(ctx context.Context) (*DownloadStatusResponse, error)
func (s *AdminService) ResetDownload(ctx context.Context, adminUserID string) error
func (s *AdminService) ReprocessDownload(ctx context.Context, adminUserID string) error
func (s *AdminService) DeleteProcessingRecord(ctx context.Context, yearMonth string, adminUserID string) error

// Background Jobs
func (s *AdminService) GetJobsList(ctx context.Context) ([]JobInfo, error)
func (s *AdminService) TriggerJob(ctx context.Context, jobName string, adminUserID string) error
func (s *AdminService) PauseScheduler(ctx context.Context, adminUserID string) error
func (s *AdminService) ResumeScheduler(ctx context.Context, adminUserID string) error

// User Management
func (s *AdminService) GetUsersList(ctx context.Context) ([]UserInfo, error)
func (s *AdminService) ToggleAdminStatus(ctx context.Context, userID string, adminUserID string) error
func (s *AdminService) ToggleActiveStatus(ctx context.Context, userID string, adminUserID string) error

// Cache Management
func (s *AdminService) GetCacheStats(ctx context.Context) (*CacheStats, error)
func (s *AdminService) GetKeysByPattern(ctx context.Context, pattern string) ([]string, error)
func (s *AdminService) GetCacheValue(ctx context.Context, key string) (interface{}, error)
func (s *AdminService) DeleteCacheKey(ctx context.Context, key string, adminUserID string) error
func (s *AdminService) DeleteCachePattern(ctx context.Context, pattern string, adminUserID string) error
```

### 4. Audit Logging

**File**: `server/internal/models/adminActionLog.model.go`

**Purpose**: Track all admin actions for security and debugging

**Model**:
```go
type AdminActionLog struct {
    ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v7()"`
    AdminUserID uuid.UUID `gorm:"type:uuid;not null;index"`
    Action      string    `gorm:"type:varchar(100);not null"` // reset_download, toggle_admin, clear_cache, etc.
    Target      *string   `gorm:"type:varchar(255)"` // user ID, cache key, job name, etc.
    Success     bool      `gorm:"not null;default:false"`
    ErrorMessage *string  `gorm:"type:text"`
    CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`

    // Relationships
    AdminUser   User `gorm:"foreignKey:AdminUserID"`
}
```

**Repository**: `server/internal/repositories/adminActionLog.repository.go`

**Action Types**:
- `reset_download` - Full download reset
- `reprocess_download` - Processing-only reset
- `delete_processing_record` - Delete specific month
- `trigger_job` - Manual job trigger
- `pause_scheduler` - Pause background jobs
- `resume_scheduler` - Resume background jobs
- `toggle_admin` - Change admin status
- `toggle_active` - Change active status
- `delete_cache_key` - Single key deletion
- `delete_cache_pattern` - Bulk key deletion

### 5. Router Configuration

**File**: `server/internal/handlers/router.go`

**Admin Routes**:
```go
// Admin routes - protected by RequireAuth + RequireAdmin
adminRoutes := app.Group("/admin", middleware.RequireAuth(), middleware.RequireAdmin())

// Dashboard
adminRoutes.Get("/dashboard", adminHandler.GetDashboard)

// Monthly Downloads
adminRoutes.Get("/downloads/status", adminHandler.GetDownloadStatus)
adminRoutes.Post("/downloads/reset", adminHandler.ResetDownload)
adminRoutes.Post("/downloads/reprocess", adminHandler.ReprocessDownload)
adminRoutes.Delete("/downloads/:yearMonth", adminHandler.DeleteProcessingRecord)

// Background Jobs
adminRoutes.Get("/jobs", adminHandler.GetJobsList)
adminRoutes.Post("/jobs/:jobName/trigger", adminHandler.TriggerJob)
adminRoutes.Post("/jobs/pause", adminHandler.PauseScheduler)
adminRoutes.Post("/jobs/resume", adminHandler.ResumeScheduler)

// User Management
adminRoutes.Get("/users", adminHandler.GetUsersList)
adminRoutes.Patch("/users/:id/admin", adminHandler.ToggleAdminStatus)
adminRoutes.Patch("/users/:id/active", adminHandler.ToggleActiveStatus)

// Cache Management
adminRoutes.Get("/cache/stats", adminHandler.GetCacheStats)
adminRoutes.Get("/cache/keys/:pattern", adminHandler.GetKeysByPattern)
adminRoutes.Get("/cache/value/:key", adminHandler.GetCacheValue)
adminRoutes.Delete("/cache/key/:key", adminHandler.DeleteCacheKey)
adminRoutes.Delete("/cache/pattern/:pattern", adminHandler.DeleteCachePattern)
```

## Frontend Implementation

### 1. Admin Page Component

**File**: `client/src/pages/AdminPage/AdminPage.tsx`

**Structure**:
```tsx
export default function AdminPage() {
  return (
    <div class={styles.adminPage}>
      <header class={styles.header}>
        <h1>Admin Dashboard</h1>
      </header>

      <MonthlyDownloadsSection />
      <BackgroundJobsSection />
      <UserManagementSection />
      <CacheManagementSection />
    </div>
  );
}
```

### 2. Section Components

**Directory**: `client/src/components/admin/`

**Components to Create**:
- `MonthlyDownloadsSection.tsx` - Download status and controls
- `BackgroundJobsSection.tsx` - Jobs list and scheduler controls
- `UserManagementSection.tsx` - Users table with toggles
- `CacheManagementSection.tsx` - Cache inspection and management
- `ConfirmationModal.tsx` - Reusable confirmation dialog

### 3. API Hooks

**File**: `client/src/services/apiHooks.ts`

**Hooks to Add**:
```typescript
// Monthly Downloads
export const useDownloadStatus = () =>
  useApiQuery<DownloadStatusResponse>(['admin', 'downloads', 'status'], '/admin/downloads/status');

export const useResetDownload = () =>
  useApiPost<void, void>('/admin/downloads/reset', undefined, {
    invalidateQueries: [['admin', 'downloads', 'status']],
    successMessage: 'Download reset initiated',
    errorMessage: 'Failed to reset download'
  });

export const useReprocessDownload = () =>
  useApiPost<void, void>('/admin/downloads/reprocess', undefined, {
    invalidateQueries: [['admin', 'downloads', 'status']],
    successMessage: 'Reprocessing initiated',
    errorMessage: 'Failed to start reprocessing'
  });

// Background Jobs
export const useJobsList = () =>
  useApiQuery<JobInfo[]>(['admin', 'jobs'], '/admin/jobs');

export const useTriggerJob = () =>
  useApiPost<void, { jobName: string }>('/admin/jobs/:jobName/trigger', undefined, {
    invalidateQueries: [['admin', 'jobs']],
    successMessage: 'Job triggered successfully',
    errorMessage: 'Failed to trigger job'
  });

// User Management
export const useUsersList = () =>
  useApiQuery<UserInfo[]>(['admin', 'users'], '/admin/users');

export const useToggleAdmin = () =>
  useApiPatch<void, { userID: string }>('/admin/users/:id/admin', undefined, {
    invalidateQueries: [['admin', 'users']],
    successMessage: 'Admin status updated',
    errorMessage: 'Failed to update admin status'
  });

// Cache Management
export const useCacheStats = () =>
  useApiQuery<CacheStats>(['admin', 'cache', 'stats'], '/admin/cache/stats');

export const useDeleteCacheKey = () =>
  useApiDelete<void>('/admin/cache/key/:key', undefined, {
    invalidateQueries: [['admin', 'cache', 'stats']],
    successMessage: 'Cache key deleted',
    errorMessage: 'Failed to delete cache key'
  });
```

### 4. Route Protection

**File**: `client/src/App.tsx`

**Admin Route**:
```tsx
<Route path="/admin" component={() => {
  const { user } = useAuth();

  // Redirect non-admin users
  if (!user?.isAdmin) {
    return <Navigate href="/" />;
  }

  return <AdminPage />;
}} />
```

**Navbar Update** (`client/src/components/layout/Navbar/Navbar.tsx`):
```tsx
<Show when={user()?.isAdmin}>
  <A href="/admin" class={styles.navLink}>
    Admin
  </A>
</Show>
```

### 5. Confirmation Modal Pattern

**File**: `client/src/components/admin/ConfirmationModal.tsx`

**Usage**:
```tsx
<ConfirmationModal
  isOpen={showResetModal()}
  onClose={() => setShowResetModal(false)}
  onConfirm={handleResetDownload}
  title="Reset Download?"
  message="This will delete the current processing record and start fresh. All progress will be lost."
  confirmText="Reset"
  confirmVariant="danger"
/>
```

## Security Implementation

### Middleware Chain
- **All admin routes**: `RequireAuth` → `RequireAdmin`
- **Rate limiting**: Stricter limits for admin endpoints (e.g., 30 requests/minute instead of 60)
- **Input validation**: Validate all parameters (user IDs, cache keys, patterns)

### Audit Trail
- Every admin action logged to `AdminActionLog` table
- Include: admin user, action type, target, success/failure, timestamp
- Failed attempts logged for security monitoring

### Confirmation Requirements
- **Single confirmation**: Reset download, trigger job, toggle user status
- **Double confirmation**: Bulk cache deletion (requires typing pattern to confirm)

### Frontend Defense in Depth
- Hide admin UI elements for non-admin users
- Client-side route guard (redirect to home)
- Backend enforcement is primary security (never trust client)

## Testing Strategy

### Backend Tests

**Admin Middleware Tests** (`server/internal/handlers/middleware/admin.middleware_test.go`):
- Admin user can access admin routes
- Non-admin user receives 403
- Unauthenticated user receives 401

**Download Management Tests** (`server/internal/services/admin.service_test.go`):
- Reset download deletes record and triggers new download
- Reprocess updates status and triggers parser
- Cannot reset/reprocess while download in progress
- Audit log created for all operations

**Job Management Tests**:
- Job trigger executes job immediately
- Pause/resume scheduler works correctly
- Cannot trigger non-existent job

**User Management Tests**:
- Toggle admin status updates user record
- Cannot remove own admin status
- Audit log tracks all changes

**Cache Management Tests**:
- Stats calculation correct for each namespace
- Key listing respects pattern matching
- Delete operations work correctly
- CacheBuilder pattern enforced

### Frontend Tests

**Admin Page Tests** (`client/src/pages/AdminPage/AdminPage.test.tsx`):
- Page renders for admin users
- Non-admin users redirected
- All sections render correctly
- Loading states display properly

**Confirmation Modal Tests**:
- Modal opens/closes correctly
- Confirmation triggers action
- Cancel prevents action
- Dangerous actions show warning styling

**API Hooks Tests**:
- Hooks use correct endpoints
- Mutations invalidate correct queries
- Success/error messages display
- Loading states managed properly

## Implementation Order

### Day 1: Backend Foundation & Monthly Downloads
1. Create admin middleware
2. Create admin handler skeleton
3. Create admin service with download operations
4. Create audit log model + repository
5. Implement download endpoints
6. Add admin routes to router
7. Test download management

### Day 2: Background Jobs & User Management
1. Implement jobs endpoints in admin handler
2. Add job management to admin service
3. Implement user management endpoints
4. Test jobs and user management
5. Start frontend admin page scaffold
6. Create MonthlyDownloadsSection component
7. Create API hooks for downloads

### Day 3: Cache Management & Frontend Components
1. Implement cache endpoints in admin handler
2. Add cache management to admin service
3. Test cache operations
4. Create BackgroundJobsSection component
5. Create UserManagementSection component
6. Create API hooks for jobs and users
7. Add route protection

### Day 4: Polish & Final Testing
1. Create CacheManagementSection component
2. Create ConfirmationModal component
3. Add API hooks for cache operations
4. Add admin link to navbar
5. Comprehensive integration testing
6. Security testing (authorization, audit logs)
7. Update documentation
8. Manual testing of all features

## Files to Create

### Backend
- `server/internal/handlers/middleware/admin.middleware.go`
- `server/internal/handlers/admin.handler.go`
- `server/internal/services/admin.service.go`
- `server/internal/models/adminActionLog.model.go`
- `server/internal/repositories/adminActionLog.repository.go`

### Frontend
- `client/src/pages/AdminPage/AdminPage.tsx`
- `client/src/pages/AdminPage/AdminPage.module.scss`
- `client/src/components/admin/MonthlyDownloadsSection.tsx`
- `client/src/components/admin/BackgroundJobsSection.tsx`
- `client/src/components/admin/UserManagementSection.tsx`
- `client/src/components/admin/CacheManagementSection.tsx`
- `client/src/components/admin/ConfirmationModal.tsx`
- `client/src/components/admin/admin.module.scss`

## Files to Modify

### Backend
- `server/internal/handlers/router.go` - Add admin routes
- `server/internal/app/app.go` - Register admin service

### Frontend
- `client/src/App.tsx` - Add admin route with protection
- `client/src/services/apiHooks.ts` - Add admin hooks
- `client/src/components/layout/Navbar/Navbar.tsx` - Add admin link

## Future Enhancements

### Zitadel Role Integration
- Sync admin role from Zitadel to `User.IsAdmin` field
- Check for admin role in JWT claims
- Auto-update admin status on login

### System Health Dashboard
- Database connection status
- Cache connection status
- Disk space monitoring
- Memory usage tracking
- Go runtime statistics

### WebSocket Monitoring
- Active connections count
- Per-user connection tracking
- Connection duration statistics
- Force disconnect capability

### Advanced Analytics
- Download success/failure rates
- Processing duration trends
- User activity metrics
- Cache hit/miss rates

### Audit Log Viewer
- Admin can view their own actions
- Filter by action type, date range
- Export audit logs
- Search functionality

## Notes & Considerations

### Cache Management Safety
- **DANGEROUS**: Bulk cache deletion can break active sessions
- Require double-confirmation: user must type exact pattern to confirm
- Show warning about potential impact
- Log all cache deletions for troubleshooting

### Job Trigger Safety
- **WARNING**: Manual job triggers can cause race conditions
- Check if job is already running before triggering
- Show last execution time and status
- Consider adding "force" flag for emergency situations

### Download Reset vs Reprocess
- **Reset**: Use when files are corrupted or incomplete
- **Reprocess**: Use when files are good but processing failed
- Both should check for in-progress operations first
- Consider adding "force" flag for stuck operations

### Admin Self-Management
- Prevent admin from removing their own admin status (requires another admin)
- Allow viewing own audit log actions
- Show warning when performing actions on own account

### Rate Limiting
- Admin endpoints should have stricter limits than public endpoints
- Consider separate rate limit buckets for admin vs regular API
- Log rate limit violations for security monitoring

### Error Handling
- All admin operations should return detailed error messages
- Include specific reason for failure (validation, permission, state, etc.)
- Log errors server-side for debugging
- Show user-friendly errors client-side

### Database Migrations
- `AdminActionLog` table will be created via GORM AutoMigrate
- No manual SQL migrations needed (per project standards)
- Run migration: `tilt trigger migrate-up` or direct Go execution
