# Job Scheduling Refactor Recommendation

The current implementation of job scheduling in `server/internal/app/app.go` is not scalable. As more jobs are added, the `New()` function will become bloated with job registration logic, making it difficult to maintain.

To address this, I recommend the following refactor:

## The Goal

Decouple job registration from the main application setup in `app.go`. This will make the code cleaner, easier to maintain, and more scalable.

## The Plan

1.  **Create a new `jobs` package:** Create a new package `internal/jobs` that will be responsible for all things related to jobs.
2.  **Create a Job Registry:** Inside the new `jobs` package, create a `registry.go` file. This file will contain a function, `RegisterJobs`, that takes the `SchedulerService` and all necessary repositories and services as arguments. This function will be responsible for creating and registering all the jobs.
3.  **Move Job Creation Logic:** Move the job creation logic for `discogsDownloadJob` and `discogsProcessingJob` from `app.go` to the `RegisterJobs` function in `jobs/registry.go`.
4.  **Update `app.go`:** In `app.go`, replace the current job registration logic with a single call to the new `jobs.RegisterJobs` function.

This approach will centralize job management, making it much easier to add, remove, or modify jobs in the future without cluttering the main application setup.

## Other Suggestions for Improving `app.go`

Beyond job scheduling, the `app.go` file can be improved in several other ways to make it more manageable as the application grows.

### 1. Introduce a Dependency Injection (DI) Container

The `New()` function in `app.go` is currently responsible for manually creating and wiring all dependencies. This is a form of manual dependency injection, which can become very complex and error-prone in a large application.

**Recommendation:**

Adopt a dependency injection container like [Google's Wire](https://github.com/google/wire) or [Uber's Fx](https://github.com/uber-go/fx). These tools automate the process of dependency injection, making the application easier to develop, maintain, and test.

**Benefits:**

*   **Simplified `New()` function:** The `New()` function would be significantly simplified, as the DI container would be responsible for creating and injecting dependencies.
*   **Improved Testability:** DI containers make it easier to mock dependencies in tests.
*   **Clearer Dependencies:** The dependencies of each component are explicitly declared, making the application easier to understand.

### 2. Group Dependencies

The `App` struct currently has a flat list of all services and repositories. This can make the struct large and difficult to read.

**Recommendation:**

Group related dependencies into structs. For example, you could have a `Services` struct and a `Repositories` struct within the `App` struct.

**Example:**

```go
type App struct {
    Services     Services
    Repositories Repositories
    // ... other fields
}

type Services struct {
    TransactionService *services.TransactionService
    ZitadelService     *services.ZitadelService
    // ... other services
}

type Repositories struct {
    UserRepo repositories.UserRepository
    // ... other repositories
}
```

### 3. Service-Specific Initializers

Instead of initializing all services in the `New()` function, each service can have its own initializer function.

**Recommendation:**

Create a `New` function for each service that takes its dependencies as arguments and returns a new instance of the service. The `New()` function in `app.go` would then call these initializers.

**Example:**

```go
// in services/user_service.go
func NewUserService(userRepo repositories.UserRepository) *UserService {
    return &UserService{
        userRepo: userRepo,
    }
}

// in app.go
userService := services.NewUserService(userRepo)
```

These suggestions, combined with the job scheduling refactor, will help to make the `app.go` file more maintainable and scalable as the application grows.
