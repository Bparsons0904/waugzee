# Waugzee Go Backend Standards and Best Practices

## Architecture Overview

The Waugzee backend follows a **clean architecture pattern** with clear separation of concerns across four distinct layers. The architecture is built around **dependency injection** using the main `App` struct that orchestrates all services, repositories, and controllers.

### Core Architecture Layers

```
┌─────────────────┐
│    Handlers     │ ← HTTP routing, middleware, request/response
├─────────────────┤
│   Controllers   │ ← Business logic coordination
├─────────────────┤
│   Repositories  │ ← Database and cache operations
├─────────────────┤
│     Models      │ ← Data definitions and validation
└─────────────────┘
```

## 1. Dependency Injection Container

**Pattern**: Central dependency injection container manages all service lifecycles

**File**: `internal/app/app.go`

**Key Characteristics**:
- Single `App` struct contains all services, repositories, controllers
- Constructor pattern with error handling
- Graceful shutdown with proper resource cleanup
- Clear initialization order

**Example Structure**:
```go
type App struct {
    Database   database.DB
    Middleware middleware.Middleware
    Websocket  *websockets.Manager
    EventBus   *events.EventBus
    Config     config.Config
    Services   services.Service
    Repos      repositories.Repository
    Controllers controllers.Controllers
}
```

## 2. Handler Standards

**Role**: HTTP concerns only - routing, middleware, request/response handling

### Handler Responsibilities

**✅ DO**:
- Handle HTTP routing and middleware application
- Parse request bodies and validate basic structure
- Extract authenticated users from middleware context
- Convert HTTP status codes and format responses
- Handle rate limiting and security headers

**❌ DON'T**:
- Implement business logic
- Directly access databases or caches
- Perform complex data transformations
- Handle authentication logic (use middleware)

### Handler Structure Pattern

```go
type ExampleHandler struct {
    Handler                    // Embedded base handler
    exampleController *Controller
}

func NewExampleHandler(app app.App, router fiber.Router) *ExampleHandler {
    return &ExampleHandler{
        exampleController: app.Controllers.Example,
        Handler: Handler{
            log:        logger.New("handlers").File("example_handler"),
            router:     router,
            middleware: app.Middleware,
        },
    }
}
```

### Request/Response Pattern

```go
func (h *ExampleHandler) HandleRequest(c *fiber.Ctx) error {
    // 1. Extract user from middleware context
    user := middleware.GetUser(c)
    if user == nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Authentication required",
        })
    }

    // 2. Parse request body
    var req ExampleRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    // 3. Delegate to controller
    result, err := h.exampleController.ProcessRequest(c.Context(), user, req)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    // 4. Return response
    return c.JSON(result)
}
```

## 3. Controller Standards

**Role**: Business logic coordination without direct database access

### Controller Responsibilities

**✅ DO**:
- Coordinate business logic across multiple repositories/services
- Handle data transformation and business rule validation
- Manage transaction boundaries when needed
- Process complex workflows and orchestration
- Convert between internal models and API responses

**❌ DON'T**:
- Directly execute SQL queries or cache operations
- Handle HTTP-specific concerns (status codes, headers)
- Implement authentication/authorization logic
- Manage database connections

### Controller Structure Pattern

```go
type ExampleController struct {
    exampleRepo repositories.ExampleRepository
    someService *services.SomeService
    config      config.Config
    log         logger.Logger
}

func New(
    repos repositories.Repository,
    services services.Service,
    config config.Config,
) *ExampleController {
    return &ExampleController{
        exampleRepo: repos.Example,
        someService: services.Some,
        config:      config,
        log:         logger.New("exampleController"),
    }
}
```

### Business Logic Pattern

```go
func (uc *ExampleController) ProcessBusinessLogic(
    ctx context.Context,
    user *User,
    req ExampleRequest,
) (*ExampleResponse, error) {
    log := uc.log.Function("ProcessBusinessLogic")

    // 1. Validate business rules
    if req.SomeField == "" {
        return nil, log.ErrMsg("required field missing")
    }

    // 2. Coordinate multiple repository/service calls
    data, err := uc.exampleRepo.GetByUserID(ctx, user.ID.String())
    if err != nil {
        return nil, log.Err("failed to fetch data", err)
    }

    // 3. Apply business logic transformations
    processed := uc.processData(data, req)

    // 4. Save results
    if err := uc.exampleRepo.Update(ctx, processed); err != nil {
        return nil, log.Err("failed to save results", err)
    }

    return &ExampleResponse{...}, nil
}
```

## 4. Repository Standards

**Role**: Database and caching operations only, minimal methods for current needs

### Repository Interface Pattern

```go
type ExampleRepository interface {
    GetByID(ctx context.Context, id string) (*Example, error)
    GetByUserID(ctx context.Context, userID string) ([]*Example, error)
    Update(ctx context.Context, example *Example) error
    Create(ctx context.Context, example *Example) (*Example, error)
    Delete(ctx context.Context, id string) error
}
```

### Repository Implementation Pattern

```go
type exampleRepository struct {
    db  database.DB
    log logger.Logger
}

func NewExampleRepository(db database.DB) ExampleRepository {
    return &exampleRepository{
        db:  db,
        log: logger.New("exampleRepository"),
    }
}
```

### Repository Responsibilities

**✅ DO**:
- Execute database queries with proper error handling
- Implement caching patterns with TTL management
- Handle database transactions when needed
- Provide minimal, focused methods for current requirements
- Use proper indexing and query optimization

**❌ DON'T**:
- Implement business logic or data transformation
- Handle HTTP concerns or user authentication
- Create methods for future requirements that don't exist yet
- Expose internal database implementation details

### Dual-Layer Caching Pattern

**Example from User Repository**:
```go
const (
    USER_CACHE_EXPIRY         = 7 * 24 * time.Hour
    USER_CACHE_PREFIX         = "user:"
    OIDC_MAPPING_CACHE_PREFIX = "oidc:"
)

func (r *userRepository) GetByOIDCUserID(ctx context.Context, oidcUserID string) (*User, error) {
    // 1. Try OIDC mapping cache first
    var userUUID string
    oidcCacheKey := OIDC_MAPPING_CACHE_PREFIX + oidcUserID
    found, err := database.NewCacheBuilder(r.db.Cache.User, oidcCacheKey).
        WithContext(ctx).Get(&userUUID)

    if err == nil && found {
        // 2. Use UUID to get from primary user cache
        var cachedUser User
        if err := r.getCacheByID(ctx, userUUID, &cachedUser); err == nil {
            return &cachedUser, nil
        }
    }

    // 3. Fallback to database query
    var user User
    if err := r.db.SQLWithContext(ctx).First(&user, "oidc_user_id = ?", oidcUserID).Error; err != nil {
        return nil, err
    }

    // 4. Cache both mappings for future requests
    r.addUserToCache(ctx, &user)
    r.cacheOIDCMapping(ctx, oidcUserID, user.ID.String())

    return &user, nil
}
```

**Benefits**:
- **Performance**: Sub-20ms response times for user lookups
- **Scalability**: Reduces database load with intelligent caching
- **Flexibility**: OIDC mapping cache enables efficient auth workflows

### CacheBuilder Usage Pattern

**Rule**: Let CacheBuilder handle key formatting internally - provide base pattern and key separately

**✅ CORRECT Usage**:
```go
// Set operation
if err := database.NewCacheBuilder(cache, requestID).
    WithHashPattern("api_request").
    WithStruct(metadata).
    WithTTL(APIRequestTTL).
    WithContext(ctx).
    Set(); err != nil {
    return err
}

// Get operation
var metadata RequestMetadata
found, err := database.NewCacheBuilder(cache, requestID).
    WithHashPattern("api_request").
    WithContext(ctx).
    Get(&metadata)
```

**❌ INCORRECT Usage**:
```go
// Don't format the key manually
cacheKey := fmt.Sprintf("api_request:%s", requestID) // ❌ DON'T DO THIS
if err := database.NewCacheBuilder(cache, cacheKey).
    WithHashPattern("api_request:%s"). // ❌ DON'T DO THIS
    Set(); err != nil {
    return err
}
```

**How CacheBuilder Works**:
- **Input**: `NewCacheBuilder(cache, "12345").WithHashPattern("api_request")`
- **Internal Processing**: CacheBuilder formats as `"api_request:12345"`
- **Result**: Consistent key formatting between Set and Get operations

**Key Benefits**:
- **Consistency**: Identical key formatting across Set/Get operations
- **Simplicity**: No manual string formatting required
- **Error Prevention**: Eliminates key mismatch bugs

## 5. Model Standards

**Role**: Data definitions that are descriptive and minimal

### Model Responsibilities

**✅ DO**:
- Define data structure with appropriate GORM tags
- Implement validation in BeforeCreate/BeforeUpdate hooks
- Use proper database types and constraints
- Include JSON tags for API serialization
- Use pointer types for optional fields

**❌ DON'T**:
- Include business logic in model methods
- Add methods that belong in services or repositories
- Create overly complex nested structures
- Include HTTP-specific or UI-specific fields

### Model Structure Pattern

```go
type Example struct {
    BaseUUIDModel                    // Embedded base with ID, timestamps
    Name        string              `gorm:"type:text;not null" json:"name"`
    Email       *string             `gorm:"type:text;uniqueIndex" json:"email,omitempty"`
    IsActive    bool                `gorm:"type:bool;default:true" json:"isActive"`
    Metadata    datatypes.JSON      `gorm:"type:jsonb" json:"metadata,omitempty"`

    // Relationships
    UserID      uuid.UUID           `gorm:"type:uuid;not null;index" json:"userId"`
    User        *User               `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
```

### Base Model Pattern

**UUID7 Primary Keys**:
```go
type BaseUUIDModel struct {
    ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
    CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
    UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
    DeletedAt gorm.DeletedAt `json:"deletedAt"`
}
```

### JSONB Data Pattern

**For Complex Data Storage**:
```go
type Release struct {
    // ... other fields
    TracksJSON  datatypes.JSON `gorm:"type:jsonb" json:"tracks,omitempty"`
    ArtistsJSON datatypes.JSON `gorm:"type:jsonb" json:"artists,omitempty"`
    GenresJSON  datatypes.JSON `gorm:"type:jsonb" json:"genres,omitempty"`
}
```

**Benefits**:
- Stores complex nested data (tracks, artists, images)
- Eliminates need for separate Track table
- Maintains PostgreSQL query capabilities
- Reduces foreign key relationship complexity

## 6. Authentication Middleware Pattern

### Enhanced Context Pattern

**Current Implementation**:
```go
func (m *Middleware) RequireAuth(zitadelService *services.ZitadelService) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // 1. Extract and validate token
        token := extractBearerToken(c)
        tokenInfo, _, err := zitadelService.ValidateTokenWithFallback(c.Context(), token)
        if err != nil {
            return unauthorizedResponse(c)
        }

        // 2. Fetch user once from database (with caching)
        user, err := m.userRepo.GetByOIDCUserID(c.Context(), tokenInfo.UserID)
        if err != nil {
            return unauthorizedResponse(c)
        }

        // 3. Store in both Fiber and Go contexts
        c.Locals(UserKeyFiber, user)
        ctx := context.WithValue(c.Context(), UserKey, user)
        c.SetUserContext(ctx)

        return c.Next()
    }
}
```

### Handler Usage Pattern

```go
func (h *Handler) SomeEndpoint(c *fiber.Ctx) error {
    user := middleware.GetUser(c) // Full User model, no conversion needed
    if user == nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Authentication required",
        })
    }

    // Use user directly - includes ID, Email, DisplayName, etc.
    response := fiber.Map{"user": user.ToProfile()}
    return c.JSON(response)
}
```

**Benefits**:
- **Performance**: User fetched once in middleware, cached in context
- **Simplicity**: No conversion needed in handlers
- **Type Safety**: Direct access to full User model
- **Consistency**: Standardized pattern across all protected endpoints

## 7. Code Quality Standards

### Minimal Comments Philosophy

**✅ GOOD Comments** (Critical/Non-obvious):
```go
// Fallback to introspection for legacy tokens without 'sub' claim
if tokenInfo.UserID == "" {
    return ac.validateWithIntrospection(ctx, token)
}

// CRITICAL: Reset loaded state when switching to fallback to prevent flashing
setLoadedState(false)
```

**❌ AVOID Comments** (Obvious):
```go
// Set background color to white
backgroundColor = "white"

// Loop through all users
for _, user := range users {
```

### Self-Documenting Code

**Prefer descriptive names over comments**:
```go
// ✅ Good - Self-describing
func (r *userRepository) addUserToCache(ctx context.Context, user *User) error

// ❌ Avoid - Needs comment
func (r *userRepository) add(ctx context.Context, u *User) error // adds user to cache
```

### No Defunct Code

**Clean up immediately when iterating**:
```go
// ❌ Bad - Commented out code
// func (h *DiscogsHandler) InitiateCollectionSync(c *fiber.Ctx) error {
//     // ... 100 lines of commented code
//     return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
// }

// ✅ Good - Working implementation or clear TODO
func (h *DiscogsHandler) InitiateCollectionSync(c *fiber.Ctx) error {
    // TODO: Implement collection sync workflow
    return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
        "error": "Collection sync not yet implemented",
    })
}
```

**Key Principle**: When iterating on work, remove defunct code entirely rather than adding new methods alongside old ones.

## 8. Testing Approach

### Testing Philosophy

**Rule**: Only implement tests when explicitly requested, then use mocks extensively

**Core Principle**: **NEVER modify business logic to make tests pass - use mocks instead**

### Mock-First Testing Pattern

```go
func TestUserController_UpdateDiscogsToken(t *testing.T) {
    // Arrange
    mockRepo := &mocks.MockUserRepository{}
    mockDiscogsService := &mocks.MockDiscogsService{}

    controller := &UserController{
        userRepo:       mockRepo,
        discogsService: mockDiscogsService,
        log:           logger.New("test"),
    }

    user := &User{ID: uuid.New()}
    request := UpdateDiscogsTokenRequest{Token: "valid-token"}

    // Mock expectations
    mockDiscogsService.On("GetUserIdentity", "valid-token").
        Return(&IdentityResponse{Username: "testuser"}, nil)
    mockRepo.On("Update", mock.Anything, user).Return(nil)

    // Act
    result, err := controller.UpdateDiscogsToken(context.Background(), user, request)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "valid-token", *result.DiscogsToken)
    mockRepo.AssertExpectations(t)
    mockDiscogsService.AssertExpectations(t)
}
```

### Testing Guidelines

**✅ DO**:
- Mock external dependencies (database, cache, HTTP clients, file system)
- Test behavior, not implementation
- Use mocks at service boundaries
- Focus on business logic validation
- Keep tests simple and focused

**❌ DON'T**:
- Modify business logic to accommodate test requirements
- Use real database connections in unit tests
- Test internal function calls or implementation details
- Create overly complex test setups

### Integration Test Strategy

**Minimal Integration Tests** (only when necessary):
- Critical path testing only
- Real database connections for database layer tests
- WebSocket connection testing
- Authentication flow end-to-end

## 9. Performance Patterns

### Caching Strategy

**Dual-Layer Caching**:
1. **Primary Cache**: Direct object caching with TTL
2. **Mapping Cache**: ID mapping for complex lookups (e.g., OIDC → UUID)

**Benefits**:
- Sub-20ms response times for common operations
- Reduced database load
- Intelligent cache invalidation

### Database Optimization

**UUID7 Primary Keys**:
- Better performance than UUID4 due to sequential ordering
- Time-sortable for debugging and analytics
- Reduces index fragmentation

**JSONB Usage**:
- Complex data storage without additional tables
- Maintains PostgreSQL query capabilities
- Reduces foreign key relationship complexity

### WebSocket Patterns

**Connection Management**:
```go
type Manager struct {
    clients    map[*websocket.Conn]*Client
    register   chan *Client
    unregister chan *Client
    broadcast  chan []byte
    mutex      sync.RWMutex
}
```

**Real-time Communication**:
- Event-driven architecture with EventBus
- Authenticated WebSocket connections
- Broadcast capability for real-time updates

## Summary

The Waugzee backend implements a **clean, minimal architecture** with:

1. **Clear Layer Separation**: Handlers→Controllers→Repositories→Models
2. **Dependency Injection**: Central App struct managing all dependencies
3. **Performance Optimization**: Dual-layer caching with OIDC mapping
4. **Minimal Implementation**: Build only what's needed now
5. **Quality Code**: Self-documenting with minimal comments
6. **Mock-Based Testing**: External dependencies mocked, not business logic modified
7. **No Defunct Code**: Clean up immediately when iterating

This architecture scales well while maintaining simplicity and allows for easy testing and maintenance as the application grows.

## Development Workflow

1. **Plan**: Define clear requirements before implementation
2. **Implement**: Follow layer responsibilities strictly
3. **Test**: Use mocks for external dependencies when testing is requested
4. **Clean**: Remove defunct code immediately during iteration
5. **Document**: Use self-documenting code over comments
6. **Optimize**: Implement caching and performance patterns as needed

This document serves as the definitive guide for maintaining consistency in Waugzee Go backend development.