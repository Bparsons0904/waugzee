# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## CRITICAL: Request Clarification Protocol

**If you cannot complete a request for ANY reason, STOP immediately and ask for clarification.**

- Don't make assumptions about unclear requirements
- Don't proceed with partial implementations
- Don't guess what the user wants
- Simply state what you don't understand and ask for specific clarification

This prevents wasted time and ensures accurate implementation.

## Project Overview

**Waugzee** is a vinyl play and cleaning logging application that helps users track when they play and clean their vinyl records. The app leverages users' existing Discogs collections as the data source and implements a client-as-proxy architecture for distributed API rate limiting. This project represents a complete rewrite of the Kleio system, focusing on minimal viable features with a clean, modern architecture.

## Project Plan

For comprehensive information about the migration strategy, architecture decisions, and implementation roadmap, see:

**[docs/PROJECT_PLAN.md](docs/PROJECT_PLAN.md)** - Complete migration strategy and implementation plan

Additional documentation available:

- **[docs/API_IMPLEMENTATION_GUIDE.md](docs/API_IMPLEMENTATION_GUIDE.md)** - API development guidelines
- **[docs/AUTH_STATUS.md](docs/AUTH_STATUS.md)** - Authentication implementation status
- **[docs/PGO_GUIDE.md](docs/PGO_GUIDE.md)** - Performance optimization guide
- **[docs/XML_PROCESSING_SERVICE.md](docs/XML_PROCESSING_SERVICE.md)** - Discogs XML processing service architecture

## Testing Philosophy

**CRITICAL RULE: Never add business logic to make tests pass - use mocks instead**

When writing or fixing tests, follow these principles:

- **Use mocks for external dependencies**: Database connections, cache clients, HTTP clients, file system operations
- **Never modify business logic to accommodate test requirements**: If a test needs specific behavior, mock the dependency rather than changing production code
- **Prefer unit tests with mocked dependencies over integration tests**: Integration tests should be minimal and focused on critical paths
- **Test behavior, not implementation**: Focus on what the code should do, not how it does it
- **Mock at service boundaries**: Mock database interfaces, external APIs, and other services rather than internal function calls

## Go Backend Standards

**CRITICAL: These patterns must be followed consistently to avoid architectural debt.**

### Database Operations

- **GORM Auto-Migration Only**: Never create manual SQL migrations - GORM's AutoMigrate handles all schema changes
- **Model Changes**: Update GORM models and run migration command - the system handles the rest
- **No Manual Schema**: SQL migrations are only for data transformations, not schema changes

### Repository Pattern

- **No Business Logic in Repositories**: Repositories handle ONLY database operations (CRUD)
- **Service Layer for Business Logic**: All business decisions happen in services, not repositories
- **Minimal Repository Methods**: Only create repository methods that are actually needed for current tasks
- **Single Responsibility**: Each repository method should have one clear database operation purpose

### Cache Operations

**CRITICAL: Manual cache key construction is ABSOLUTELY FORBIDDEN.**

- **NEVER construct cache keys manually**: Manual concatenation like `constants.SomePrefix + someValue` is FORBIDDEN
- **ALWAYS use CacheBuilder pattern**: Use `database.NewCacheBuilder(cache, identifier)` with builder methods
- **Use WithHash() for simple prefixes**: Most common pattern for prefix + identifier
- **Use WithHashPattern() only for complex patterns**: Reserved for truly complex scenarios
- **Consistent Set/Get operations**: Ensure identical patterns between cache writes and reads

**Required Patterns:**

```go
// ✅ CORRECT - Simple prefix (most common case)
var cachedResponse SomeResponse
found, err := database.NewCacheBuilder(cache, userID).
    WithContext(ctx).
    WithHash(constants.SomeCachePrefix).
    Get(&cachedResponse)

// ✅ CORRECT - Setting with same pattern
err := database.NewCacheBuilder(cache, userID).
    WithContext(ctx).
    WithHash(constants.SomeCachePrefix).
    Set(response, time.Hour)

// ❌ FORBIDDEN - Manual key construction
cacheKey := constants.SomeCachePrefix + userID
found, err := database.NewCacheBuilder(cache, cacheKey).Get(&cachedResponse)

// ❌ FORBIDDEN - Any form of manual concatenation
found, err := database.NewCacheBuilder(cache, prefix + ":" + id).Get(&cachedResponse)
```

### Service Architecture

- **Business Logic in Services**: Services contain all business decisions and orchestration
- **Repository Delegation**: Services call specific repository methods for data operations
- **No Cross-Service Business Logic**: Keep business logic within the appropriate service boundary
- **Clear Separation**: Services determine WHAT to do, repositories determine HOW to store/retrieve

### Common Anti-Patterns to Avoid

❌ **Manual SQL migrations for schema changes**
❌ **Business logic in repository methods**
❌ **MANUALLY CONSTRUCTED CACHE KEYS (ABSOLUTE ZERO TOLERANCE)**
❌ **Repository methods with complex business decisions**
❌ **Creating repository methods "just in case"**

✅ **GORM model updates + AutoMigrate**
✅ **Business logic in service layer**
✅ **CacheBuilder pattern with WithHash() for ALL cache operations**
✅ **Simple, focused repository methods**
✅ **Create methods only when needed**

## Technology Stack

### Backend (Go)

- **Framework**: Fiber web framework
- **Database**: PostgreSQL (primary) + Valkey/Redis (cache)
- **ORM**: GORM with UUID7 primary keys
- **Architecture**: Repository pattern with dependency injection
- **Authentication**: Zitadel OIDC integration
- **WebSockets**: Real-time communication support

### Frontend (SolidJS)

- **Framework**: SolidJS with TypeScript
- **Build Tool**: Vite
- **Styling**: SCSS with CSS Modules
- **State Management**: TanStack Query (Solid Query) + Context API
- **Authentication**: OIDC flow integration

**CRITICAL: Always use TanStack Query for API calls**

- **NEVER use `api.ts` directly in components or services** - it's ONLY for:
  - Internal use by `apiHooks.ts`
  - Authentication operations in `AuthContext.tsx`
- **ALWAYS use hooks from `@services/apiHooks`**:
  - `useApiQuery` for GET requests
  - `useApiPut` for PUT requests (with `invalidateQueries` for cache invalidation)
  - `useApiPost` for POST requests (with `invalidateQueries` for cache invalidation)
  - `useApiPatch` for PATCH requests (with `invalidateQueries` for cache invalidation)
  - `useApiDelete` for DELETE requests (with `invalidateQueries` for cache invalidation)
- **Use `invalidateQueries` option** to automatically refetch data after mutations
- **No manual cache management** - TanStack Query handles caching, loading states, and error states

**Example Pattern:**

```typescript
// ✅ CORRECT - Declarative pattern with callbacks (preferred)
const updateMutation = useApiPut<ResponseType, RequestType>(
  API_ENDPOINT,
  undefined,
  {
    invalidateQueries: [["queryKey"]], // Automatically refetch after success
    successMessage: "Update successful!",  // Auto toast notification
    errorMessage: "Update failed. Please try again.",  // Auto error toast
    onSuccess: (data) => {
      // Additional success logic (optional)
      console.log("Success:", data);
      someStateUpdate(data);
    },
    onError: (error) => {
      // Additional error handling (optional)
      console.error("Error:", error);
    },
  }
);

// Simple mutation call - no try/catch needed
updateMutation.mutate(data);

// ❌ AVOID - Manual try/catch (unnecessary with onSuccess/onError)
try {
  await updateMutation.mutateAsync(data);
  toast.showSuccess("Update successful!");
} catch (error) {
  toast.showError("Update failed");
}

// ❌ FORBIDDEN - Direct API usage in components
import { api } from "@services/api";
const response = await api.put(endpoint, data);
```

**Key Benefits of Declarative Pattern:**
- Automatic toast notifications via `successMessage` and `errorMessage`
- No manual try/catch blocks needed
- Cleaner, more readable code
- Consistent error handling across the app
- `onSuccess` and `onError` callbacks for additional logic

### Infrastructure

- **Development**: Tilt orchestration with Docker
- **Cache**: Valkey (Redis-compatible)
- **Database**: PostgreSQL with proper migrations

## Common Development Commands

### Development Environment

- **Start development**: `tilt up` (starts all services with hot reloading)
- **Stop development**: `tilt down`
- **View logs**: `tilt up --stream`
- **Tilt dashboard**: http://localhost:10350

### Testing & Linting

- **Server tests**: `tilt trigger server-tests` or `go test -C ./server ./...`
- **Server test coverage**: `go test -C ./server -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html`
- **Server linting**: `tilt trigger server-lint` or `\cd server && golangci-lint run`
- **Client linting**: `tilt trigger client-lint` or `npm run lint:check -C ./client`
- **Client tests**: `tilt trigger client-tests` or `npm run test -C ./client`
- **TypeScript check**: `npm run typecheck -C ./client`

### Database Operations

- **Run migrations**: `tilt trigger migrate-up`
- **Rollback migration**: `tilt trigger migrate-down`
- **Seed database**: `tilt trigger migrate-seed`
- **Valkey info**: `tilt trigger valkey-info`

### Manual Development (without Tilt)

- **Server**: `go run -C ./server cmd/api/main.go`
- **Client**: `npm run dev -C ./client`
- **Full stack**: `docker compose -f docker-compose.dev.yml up --build`

### Important Note: cd Command Aliasing

The `cd` command is aliased to zoxide and cannot be used directly in bash commands. When using bash commands, use one of these alternatives:

- **Use the -C flag**: `go test -C ./server ./...` (preferred)
- **Use builtin cd**: `\cd server && go test ./...` (escapes the alias)
- **Use absolute paths**: `go test /home/bobparsons/Development/waugzee/server/...`
- **NEVER use**: `cd server && go test ./...` (this will fail due to zoxide aliasing)

## Architecture Overview

### High-Level Structure

Full-stack vinyl record collection management application:

- **Backend**: Fiber framework with PostgreSQL + Valkey, Zitadel auth, WebSockets
- **Frontend**: SolidJS with TypeScript, Vite, CSS Modules, Solid Query
- **Cache**: Valkey (Redis-compatible) for sessions and caching
- **Orchestration**: Docker Compose + Tilt for development

### Key Ports (Development)

- Server API: http://localhost:8289 (WebSocket: ws://localhost:8289/ws)
- Client App: http://localhost:3021
- PostgreSQL: localhost:5432
- Valkey DB: localhost:6399

### Backend Architecture (Go)

- **Dependency Injection**: App struct (`internal/app/app.go`) contains all services
- **Repository Pattern**: Interface-based data access layer
- **Database**: Dual database setup - PostgreSQL (primary) + Valkey (cache)
- **Auth**: Zitadel OIDC integration with JWT tokens
- **WebSockets**: Manager pattern with hub for real-time communication

### Frontend Architecture (SolidJS)

- **State Management**: AuthContext + Solid Query for server state
- **API Layer**: Axios with interceptors for token management
- **WebSocket**: Auto-connecting WebSocket context with auth token header
- **Routing**: @solidjs/router with protected routes
- **Styling**: SCSS with CSS Modules pattern

### Database Layer

- **Primary**: PostgreSQL with GORM, UUID7 primary keys
- **Cache**: Valkey client for sessions and temporary data
- **Models**: GORM models with proper relationships
- **Migrations**: GORM-based migration system

### Authentication Flow (Zitadel OIDC)

1. OIDC flow with Zitadel for authentication
2. JWT tokens for API access with signature verification
3. Local user database with dual-layer caching (optimized 2025-09-10)
4. Session management via Valkey cache
5. WebSocket auth uses same token pattern
6. **NEW** Hybrid JWT validation middleware (optimized 2025-09-12)

**Performance Optimizations** ✅:

- `/auth/me` endpoint uses local database instead of external Zitadel API calls
- Dual-layer caching: User cache + OIDC ID mapping for sub-20ms response times
- **NEW** Sub-millisecond JWT validation (500x improvement: 500ms → <1ms)
- **NEW** Smart token detection with introspection fallback for backward compatibility
- Eliminated redundant external API dependencies for routine user operations

### Authentication Middleware Patterns

**NEW**: Handlers can now access authenticated users directly from middleware context:

```go
// Get the full User model from middleware context
user := middleware.GetUser(c)
if user == nil {
    return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
}

// User object includes all fields: ID, Email, FirstName, etc.
response := fiber.Map{"user": user.ToProfile()}
```

**Key Benefits:**

- **Performance**: User fetched once in middleware, cached in context
- **Simplicity**: No AuthInfo conversion needed in handlers
- **Type Safety**: Direct access to full User model with all fields and methods
- **Consistency**: Standardized pattern across all protected endpoints

**Legacy Pattern (No Longer Needed):**

```go
// OLD - Don't use this pattern anymore
authInfo := middleware.GetAuthInfo(c)
controllerAuthInfo := &userController.AuthInfo{...} // Manual conversion
```

## Development Philosophy

### Minimal Implementation Approach

**Core Principle**: Build only what the current implementation needs. Keep implementations minimal while allowing forward-thinking in planning.

**Guidelines**:
- ✅ **Implement current requirements**: Focus on play logging and cleaning tracking features
- ✅ **Forward-thinking planning**: Document future features and architecture decisions
- ❌ **Avoid over-engineering**: Don't build abstractions for future requirements that may never materialize
- ❌ **No premature optimization**: Implement simple solutions first, optimize when needed

**Examples**:
- **Good**: Simple play logging with user, release, timestamp, and notes
- **Good**: Planning for equipment tracking but implementing basic stylus reference first
- **Avoid**: Complex analytics engines before basic logging is complete
- **Avoid**: Over-abstracted repository patterns for single-use cases

## Development Notes

### Coding Standards

**Clean Code Principles:**

- **Self-documenting code**: Use descriptive variable and function names instead of comments
- **Comments only for critical areas**: Limit comments to complex business logic or hard-to-understand algorithms
- **No obvious comments**: Avoid comments that simply restate what the code does
- **SCSS variables over hardcoded values**: Always use design system variables instead of magic numbers
- **Consistent formatting**: Follow project linting and formatting rules

**Comment Guidelines:**

- ✅ **Good**: `// Fallback to introspection for legacy tokens without 'sub' claim`
- ✅ **Good**: `// CRITICAL: Reset loaded state when switching to fallback to prevent flashing`
- ❌ **Avoid**: `// Increased padding for larger cards`
- ❌ **Avoid**: `// Set background color to white`
- ❌ **Avoid**: `// Hero Section` or `// Features Section`

**SCSS/CSS Standards:**

- **Use design system variables**: `$spacing-xl` not `2rem`, `$text-default` not `#333`
- **Semantic class names**: `.featureCard` not `.socialCard`, `.heroImage` not `.cardImage`
- **Mobile-first responsive**: Use `@media (min-width: ...)` for larger screens
- **Consistent spacing**: Use spacing scale variables (`$spacing-xs` through `$spacing-3xl`)
- **No magic numbers**: All values should reference design system variables

**Component Development:**

- **Single responsibility**: Each component should have one clear purpose
- **Proper TypeScript**: Full type safety with interfaces for all props
- **Loading states**: Use skeleton loading for better UX
- **Error boundaries**: Handle error states gracefully with fallbacks
- **Accessibility**: Proper alt text, ARIA labels, keyboard navigation
- **Testing**: Comprehensive test coverage for component behavior

### Naming Conventions

**CRITICAL: All naming must follow consistent camelCase/PascalCase standards. NO kebab-case allowed.**

**File Naming:**
- **Services**: camelCase - `userService.ts`, `apiHooks.ts`, `discogsProxy.service.ts`
- **Components**: PascalCase - `Modal.tsx`, `Button.tsx`, `HomePage.tsx`
- **Utilities**: camelCase - `dateUtils.ts`, `formatHelpers.ts`
- **Types/Interfaces**: PascalCase - `User.ts`, `ApiResponse.ts`
- **CSS/SCSS**: camelCase - `button.module.scss`, `modal.module.scss`

**Route Naming:**
- **API Endpoints**: camelCase - `/syncCollection`, `/rateLimit`, `/getUserProfile`
- **Frontend Routes**: camelCase - `/silentCallback`, `/userDashboard`
- **NEVER use kebab-case**: ❌ `/sync-collection`, ❌ `/rate-limit`

**Variable & Function Naming:**
- **Variables**: camelCase - `userName`, `syncStatus`, `isLoading`
- **Functions**: camelCase - `handleSubmit`, `fetchUserData`, `validateToken`
- **Constants**: SCREAMING_SNAKE_CASE - `API_BASE_URL`, `MAX_RETRY_ATTEMPTS`
- **Components**: PascalCase - `UserProfile`, `SyncButton`, `ModalDialog`

**CSS Class Naming:**
- **CSS Classes**: camelCase - `.userProfile`, `.syncButton`, `.errorMessage`
- **CSS Variables**: kebab-case (exception) - `--primary-color`, `--font-size-large`
- **SCSS Mixins**: camelCase - `@mixin buttonStyles`, `@mixin cardLayout`

**Enforcement:**
- **Code Reviews**: All PRs must follow these naming conventions
- **Linting**: ESLint and Go linting rules enforce these standards
- **Immediate Fix Required**: Any kebab-case discovered should be fixed immediately
- **No Exceptions**: Only CSS variables may use kebab-case due to CSS specification requirements

### File Structure Guidelines

- **NEVER create index.js/ts files unless absolutely necessary** - Use direct imports instead
- Index files create confusion and make navigation harder as projects grow
- Prefer explicit imports like `import { Modal } from "./components/Modal/Modal"`
- **Component organization**: Each component in own directory with `.tsx` and `.module.scss`

### Reference Repository

**Legacy Code Reference**: The `/oldReferenceOnlyRepository` directory contains the complete legacy implementation for reference purposes:

- **Models & Logic**: Reference existing data models, business logic patterns, and API structures
- **Styling & UI**: Reference SCSS patterns, component structures, and design system elements
- **Implementation Patterns**: Reference proven patterns for features like collection management, equipment tracking, and user workflows

**Important**: This directory is for reference only - do not modify files in this location. Use it to understand existing patterns when implementing new features in the current codebase.

### Key Files to Understand

- `server/internal/app/app.go` - Main dependency injection container
- `client/src/context/AuthContext.tsx` - Auth state management
- `server/internal/handlers/router.go` - API route definitions
- `client/src/services/api/api.service.ts` - API client with interceptors
- `Tiltfile` - Development environment configuration
- `docs/PROJECT_PLAN.md` - Complete project roadmap and architecture decisions
- `/oldReferenceOnlyRepository/` - Legacy implementation for reference

### Current Development: Discogs Data Import Infrastructure

**Status**: ✅ **Phase 2 Complete** - Simplified JSONB processing implemented with deadlock resolution

**Processing Approach (Major Simplification - 2025-01-17):**

**Core Strategy**: Vinyl-only processing with JSONB storage and exact association processing.

**Data Architecture**:

- **Vinyl-Only Filtering**: Process only vinyl releases, skip CD/digital/cassette (~70-80% volume reduction)
- **JSONB Storage**: Store tracks, artists, genres as JSON in Release table (no separate Track table)
- **Master-Level Relationships**: Maintain searchable relationships only at Master level
- **Exact Association Processing**: Process specific master-artist pairs, never cross-products
- **Query Pattern**: Release → Master → Artists/Genres for searches, direct JSONB for display

**Key Changes**:

- ✅ **Eliminated Track Model**: Removed separate Track table and repository entirely
- ✅ **JSONB Columns**: Added TracksJSON, ArtistsJSON, GenresJSON to Release model using `gorm.io/datatypes`
- ✅ **Fixed Association Processing**: Eliminated cross-product bug that created millions of unwanted associations
- ✅ **Deadlock Resolution**: Added proper ordering and exact pair processing to prevent database deadlocks
- ✅ **Simplified Processing**: Single-threaded buffer processing with controlled batch sizes
- ✅ **Early Filtering**: Skip non-vinyl releases immediately after format detection

**Performance Impact**:

- **Processing Volume**: 70-80% reduction through vinyl-only filtering
- **Database Operations**: Eliminated cross-product associations (1M+ → 1K associations)
- **Deadlock Prevention**: Proper ordering and batch size limits prevent lock contention
- **Storage Efficiency**: JSONB replaces complex foreign key relationships
- **Processing Speed**: Simplified pipeline with exact association processing

**Implementation Files**:

- `server/internal/models/release.model.go` - JSONB columns added
- `server/internal/services/discogsParser.service.go` - Vinyl filtering and JSONB generation
- `server/internal/services/simplifiedXmlProcessing.service.go` - Fixed association processing
- `server/internal/repositories/master.repository.go` - Exact association pair processing
- `docs/XML_PROCESSING_SERVICE.md` - Complete service architecture documentation

### Environment Configuration

All environment variables in `.env` at project root, shared between services.

**Local Environment Overrides:**

- Copy `.env.local.example` to `.env.local` for local development overrides
- `.env.local` is git-ignored and will override values from `.env`

## Business Domain

### Core Features (Implemented/In Progress)

1. **Play Logging**: Track when vinyl records are played with equipment details and notes
2. **Cleaning Tracking**: Log cleaning sessions including deep cleaning with timestamps and notes
3. **Discogs Collection Integration**: Use user's existing Discogs collection as the data source via client-as-proxy API pattern

### Architecture: Client-as-Proxy Pattern

**Key Concept**: Each user makes their own Discogs API calls with their personal token, distributing rate limits across users while the server orchestrates complex sync logic.

**Benefits**:
- **Distributed Rate Limits**: Each user operates within their own Discogs API quota
- **Server Orchestration**: Backend manages sync workflows, state persistence, and business logic
- **Real-time Communication**: WebSocket enables immediate progress updates during sync
- **Scalability**: Performance scales naturally with user count

**Implementation**:
- Users provide their own Discogs tokens
- Frontend makes actual HTTP requests to Discogs API
- Backend receives responses via WebSocket and processes data
- Server tracks sync progress and manages database updates

### Data Models (Current Implementation)

- ✅ **Users**: Multi-tenant user management via Zitadel with Discogs token storage
- ✅ **PlayHistory**: Play sessions linking users, releases, stylus, and play timestamps
- ✅ **CleaningHistory**: Cleaning records with deep clean flags and user notes
- ✅ **UserRelease**: User's vinyl collection items synced from Discogs
- ✅ **Equipment Models**: Stylus tracking for play sessions
- ✅ **Discogs Data**: Artists, Labels, Masters, Releases from monthly XML processing

## MCP Tools Usage

**CRITICAL: Always prioritize MCP (Model Context Protocol) tools over bash commands when available.**

Available MCP tools and their preferred usage:

- **File Operations**: Use `mcp__filesystem__*` tools when available
  - `mcp__filesystem__read_file` instead of `cat`
  - `mcp__filesystem__write_file` for file creation
  - `mcp__filesystem__list_directory` instead of `ls`

## Migration Status

This project is currently in **Phase 2: Authentication & User Management**. See docs/PROJECT_PLAN.md for detailed progress and next steps.

**Recent Improvements** (2025-09-10):

- ✅ **Auth Performance Optimization**: Eliminated redundant Zitadel API calls for user info requests
- ✅ **Dual-Layer Caching**: Implemented OIDC ID mapping cache for faster user lookups
- ✅ **Database-First Approach**: `/auth/me` now uses local database with Valkey cache fallback

---

**Project Status**: 🚧 **Active Development** - Phase 2: Authentication & User Management

