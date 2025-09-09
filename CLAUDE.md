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

**Waugzee** is a modern vinyl record collection management system built as a fresh implementation using proven architectural patterns. This project represents a complete rewrite of the Kleio system, leveraging the robust foundation from the Vim project architecture.

## Project Plan

For comprehensive information about the migration strategy, architecture decisions, and implementation roadmap, see:

**[PROJECT_PLAN.md](PROJECT_PLAN.md)** - Complete migration strategy and implementation plan

## Testing Philosophy

**CRITICAL RULE: Never add business logic to make tests pass - use mocks instead**

When writing or fixing tests, follow these principles:

- **Use mocks for external dependencies**: Database connections, cache clients, HTTP clients, file system operations
- **Never modify business logic to accommodate test requirements**: If a test needs specific behavior, mock the dependency rather than changing production code
- **Prefer unit tests with mocked dependencies over integration tests**: Integration tests should be minimal and focused on critical paths
- **Test behavior, not implementation**: Focus on what the code should do, not how it does it
- **Mock at service boundaries**: Mock database interfaces, external APIs, and other services rather than internal function calls

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
- **State Management**: Solid Query + Context API
- **Authentication**: OIDC flow integration

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
- **Server linting**: `tilt trigger server-lint` or `golangci-lint run -C ./server`
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

- Server API: http://localhost:8288 (WebSocket: ws://localhost:8288/ws)
- Client App: http://localhost:3020
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
2. JWT tokens for API access
3. Session management via Valkey cache
4. WebSocket auth uses same token pattern
5. Middleware validates tokens on protected routes

## Development Notes

### File Structure Guidelines

- **NEVER create index.js/ts files unless absolutely necessary** - Use direct imports instead
- Index files create confusion and make navigation harder as projects grow
- Prefer explicit imports like `import { Modal } from "./components/Modal/Modal"`

### Key Files to Understand

- `server/internal/app/app.go` - Main dependency injection container
- `client/src/context/AuthContext.tsx` - Auth state management
- `server/internal/handlers/router.go` - API route definitions
- `client/src/services/api/api.service.ts` - API client with interceptors
- `Tiltfile` - Development environment configuration
- `PROJECT_PLAN.md` - Complete project roadmap and architecture decisions

### Environment Configuration

All environment variables in `.env` at project root, shared between services.

**Local Environment Overrides:**

- Copy `.env.local.example` to `.env.local` for local development overrides
- `.env.local` is git-ignored and will override values from `.env`

## Business Domain

### Core Features (To Be Implemented)

1. **Multi-User Collection Management**: User-scoped vinyl record collections
2. **Discogs Integration**: Automatic collection sync and metadata
3. **Play Tracking**: Log listening sessions with equipment details
4. **Equipment Management**: Track turntables, cartridges, and styluses
5. **Maintenance Tracking**: Record cleaning and maintenance history
6. **Analytics Dashboard**: Listening patterns and collection insights

### Data Models (Planned)

- **Users**: Multi-tenant user management via Zitadel
- **Collections**: User-owned vinyl records with Discogs integration
- **Equipment**: Turntables, cartridges, styluses with usage tracking
- **Sessions**: Play sessions linking records, equipment, and user notes
- **Maintenance**: Cleaning and care records for collection items

## MCP Tools Usage

**CRITICAL: Always prioritize MCP (Model Context Protocol) tools over bash commands when available.**

Available MCP tools and their preferred usage:

- **File Operations**: Use `mcp__filesystem__*` tools when available
  - `mcp__filesystem__read_file` instead of `cat`
  - `mcp__filesystem__write_file` for file creation
  - `mcp__filesystem__list_directory` instead of `ls`

## Migration Status

This project is currently in **Phase 1: Foundation Setup**. See PROJECT_PLAN.md for detailed progress and next steps.

---

**Project Status**: 🚧 **Active Development** - Foundation phase in progress