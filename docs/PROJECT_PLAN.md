# Waugzee Project Plan

## Project Overview

**Waugzee** is a vinyl play and cleaning logging application that helps users track when they play and clean their vinyl records. The app leverages users' existing Discogs collections as the data source and implements a client-as-proxy architecture for distributed API rate limiting.

## Current Status

**Phase**: 3 - Core Data Models & Business Logic
**Status**: 🚀 In Active Development
**Last Updated**: 2025-01-03

## Completed Foundation ✅

- ✅ **Infrastructure**: PostgreSQL + Valkey, Docker + Tilt orchestration
- ✅ **Authentication**: Zitadel OIDC with JWT validation (sub-millisecond performance)
- ✅ **Security**: Enterprise-grade auth, proper token management, CSRF protection
- ✅ **Performance**: Optimized caching, dual-layer user lookup, hybrid JWT validation
- ✅ **Architecture**: Repository pattern, dependency injection, clean separation of concerns
- ✅ **Deployment**: Drone CI configured, production secrets set, Zitadel production instance ready

## Active Development

### Phase 3: Core Data Models ⭐ IN PROGRESS

**Current Focus**: Discogs data integration and collection management

**Completed**:
- ✅ Core entity models (Users, Artists, Labels, Masters, Releases)
- ✅ Equipment models (Turntables, Cartridges, Styluses)
- ✅ Play tracking models (PlayHistory, CleaningHistory)
- ✅ Repository interfaces and implementations
- ✅ GORM migrations with proper relationships

**In Progress**:
- [ ] Client-as-proxy Discogs API integration
- [ ] User collection sync from Discogs
- [ ] Play logging business logic
- [ ] Equipment tracking features

### Phase 4: Business Logic & API Layer

**Goals**: Complete REST API with full feature set

**Planned**:
- [ ] Collection sync controllers
- [ ] Play logging endpoints
- [ ] Equipment management API
- [ ] Search and filtering
- [ ] Analytics and statistics

### Phase 5: Frontend Development

**Goals**: Complete user interface with real-time features

**Planned**:
- [ ] Collection management UI
- [ ] Play logging interface
- [ ] Equipment tracking screens
- [ ] Analytics dashboard
- [ ] Real-time updates via WebSocket

## Technical Architecture

### Backend Stack
- **Language**: Go 1.25+
- **Framework**: Fiber v2
- **Database**: PostgreSQL 15+ (primary) + Valkey (cache)
- **ORM**: GORM with UUID7 primary keys
- **Authentication**: Zitadel OIDC with hybrid JWT validation
- **Architecture**: Repository pattern with dependency injection

### Frontend Stack
- **Framework**: SolidJS with TypeScript
- **Build Tool**: Vite
- **Styling**: SCSS with CSS Modules
- **State Management**: TanStack Query (Solid Query) + Context API
- **Authentication**: oidc-client-ts with secure in-memory tokens
- **Real-time**: WebSocket integration with auth

### Infrastructure
- **Development**: Tilt orchestration with hot reloading
- **Production**: Docker containers, Traefik reverse proxy, Let's Encrypt SSL
- **CI/CD**: Drone CI with automated builds and deployments
- **Cache Strategy**: Dual-layer caching (user + OIDC mapping)

## Key Design Decisions

### Client-as-Proxy Architecture
- Users provide their own Discogs API tokens
- Frontend makes Discogs API calls directly
- Backend orchestrates sync workflow and data persistence
- Distributed rate limiting across users
- Real-time progress updates via WebSocket

### Performance Optimizations
- **Sub-millisecond JWT validation** (500x improvement over introspection)
- **Dual-layer caching** for user lookups (sub-20ms response times)
- **Native PostgreSQL UPSERT** for batch operations
- **Streaming XML processing** for large data imports
- **Optimized batch sizes** (2K-5K records) for high throughput

### Security Features
- **Zitadel OIDC** with PKCE and state validation
- **In-memory token storage** (no localStorage XSS risk)
- **Automatic token refresh** with silent renewal
- **Hybrid JWT validation** with introspection fallback
- **Fail-fast configuration** (won't start without proper auth config)

## Development Workflow

### Start Development
```bash
tilt up                    # Start all services
# Access Tilt dashboard at http://localhost:10350
# Frontend at http://localhost:3021
# Backend API at http://localhost:8289
```

### Testing & Linting
```bash
tilt trigger server-tests  # Run Go tests
tilt trigger client-tests  # Run frontend tests
tilt trigger server-lint   # golangci-lint
tilt trigger client-lint   # Biome linter
```

### Database Operations
```bash
tilt trigger migrate-up    # Run migrations
tilt trigger migrate-seed  # Seed database
```

## Next Steps

### Immediate (This Week)
1. Implement Discogs collection sync (client-as-proxy pattern)
2. Add play logging business logic
3. Create collection management API endpoints
4. Build basic frontend collection view

### Short Term (Next 2 Weeks)
1. Complete play tracking feature
2. Add equipment management
3. Implement search and filtering
4. Build analytics foundation

### Medium Term (Next Month)
1. Complete frontend UI
2. Add real-time WebSocket features
3. Implement analytics dashboard
4. Production deployment testing

## Success Criteria

### MVP Ready When:
- [ ] Users can sync Discogs collections
- [ ] Users can log vinyl plays with equipment
- [ ] Users can track cleaning sessions
- [ ] Basic search and filtering works
- [ ] Production deployment stable

### Full Feature Set:
- [ ] Advanced analytics and statistics
- [ ] Equipment tracking and maintenance
- [ ] Multi-user collaboration features
- [ ] Mobile-responsive design
- [ ] Comprehensive test coverage (>80%)

---

**Status**: 🎯 Phase 3 in progress - Building collection sync and play logging features
