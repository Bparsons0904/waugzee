# Discogs Monthly Data Processing - Work Tickets

## Epic Overview

Implement automated monthly processing of Discogs data dumps to populate the application's core music database with artists, labels, masters, and releases data.

### 🎉 **Current Status: Phase 1 Complete (2025-09-14)**

**✅ Production-Ready Infrastructure:**
- **Hourly Cron Jobs**: Automated scheduling with gocron
- **Download System**: Streaming downloads with SHA256 validation
- **State Management**: File-level tracking with recovery capabilities
- **Error Handling**: Exponential backoff and transaction safety
- **Tech Lead Approved**: Ready for production deployment

**✅ Completed Tickets:**
- **Ticket #1**: Database tracking model ✅
- **Ticket #2**: Cron job scheduling service ✅
- **Ticket #4**: Validation service (SHA256 verification) ✅

**✅ Recently Completed:**
- **Ticket #3**: Download service **PRODUCTION VERIFIED** ✅ (2025-09-14: 4 files totaling 11.9GB successfully downloaded)

**📋 Next Phase:**
- **Ticket #5**: XML parsing and database population (artists/labels)

---

## Ticket #1: Create Discogs Data Processing Tracking Model ✅ **COMPLETED**

**Priority:** High  
**Story Points:** 3  
**Status:** ✅ Completed (2025-09-13)

### Description

Create a database model to track the state and progress of monthly Discogs data processing operations.

### Acceptance Criteria

- [x] Create `discogs_data_processing` table with proper schema
- [x] Include fields for year_month, status, timestamps, checksums, retry count, and error handling
- [x] Implement status enum (not_started, downloading, ready_for_processing, processing, completed, failed)
- [x] Add GORM model with proper relationships and validation
- [x] Create database migration for the new table
- [x] Add repository interface and implementation

### Implementation Details

**Files Created:**

- `server/internal/models/discogsDataProcessingModel.go` - GORM model with validation
- `server/internal/repositories/discogsDataProcessing.repository.go` - Repository interface & implementation

**Key Features Implemented:**

- **UUID7 Primary Key**: Following project consistency standards
- **ProcessingStatus Enum**: All 6 required statuses with proper constants
- **JSONB Fields**: FileChecksums and ProcessingStats for structured data storage
- **Input Validation**: YearMonth regex validation (YYYY-MM format) in GORM hooks
- **Status Transition Validation**: CanTransitionTo() and UpdateStatus() methods with comprehensive transition rules
- **Nullable Timestamps**: StartedAt, DownloadCompletedAt, ProcessingCompletedAt, CompletedAt
- **Database Indexing**: Unique index on year_month, regular index on status
- **Repository Pattern**: Complete CRUD operations with transaction support
- **Context-Aware Operations**: All repository methods support database transactions

**Security & Performance:**

- Parameterized queries prevent SQL injection
- Context-aware database operations prevent race conditions
- Proper error handling and logging throughout
- Efficient query patterns with appropriate indexing

**Integration:**

- Added to migration system (`MODELS_TO_MIGRATE`)
- Integrated with app dependency injection
- Follows established project patterns and conventions

### Technical Notes

- ✅ UUID7 primary key implemented for consistency
- ✅ File checksums stored as JSONB for validation support
- ✅ Retry count tracking with validation hooks
- ✅ Proper indexing on year_month (unique) and status fields
- ✅ Code review completed - all critical issues addressed

---

## Ticket #2: Implement Cron Job Scheduling Service ✅ **COMPLETED**

**Priority:** High  
**Story Points:** 5  
**Status:** ✅ Completed (2025-09-13)

### Description

Create a scheduling service to manage periodic tasks for Discogs data downloading and processing.

### Acceptance Criteria

- [x] Design and implement cron job scheduling architecture
- [x] Create daily job for checking/downloading monthly data
- [x] Create processing job for parsing downloaded files
- [x] Add configuration for job timing and intervals
- [x] Implement proper logging and monitoring for scheduled tasks
- [x] Add graceful shutdown handling for running jobs
- [x] Include job status reporting and health checks

### Implementation Details

**Files Created:**

- `server/internal/services/scheduler.service.go` - Main scheduler service with gocron integration
- `server/internal/jobs/discogsDownload.job.go` - Daily Discogs data processing check job
- `server/internal/jobs/discogsDownload.job_test.go` - Unit tests for job logic

**Files Modified:**

- `server/config/config.go` - Added `SchedulerEnabled` configuration field
- `server/internal/app/app.go` - Integrated scheduler service into dependency injection
- `server/cmd/api/main.go` - Added scheduler startup and graceful shutdown
- `server/.env` - Added `SCHEDULER_ENABLED=true` configuration
- `server/go.mod` - Added gocron dependency

**Key Features Implemented:**

- **gocron Integration**: Uses gocron library for robust job scheduling (https://github.com/go-co-op/gocron)
- **Daily Execution**: Jobs run daily at 2:00 AM UTC
- **Lifecycle Management**: Proper start/stop with context-aware cancellation
- **Thread Safety**: Mutex-protected operations for concurrent access
- **Transaction Safety**: All database operations wrapped in transactions to prevent race conditions
- **Graceful Shutdown**: Cancellable context support for proper job termination
- **Robust Error Handling**: Uses proper GORM error checking with `errors.Is()`
- **Status Management**: Follows business rules for processing state transitions using model validation
- **Job Interface**: Clean interface for registering schedulable jobs
- **Environment Control**: Controlled via `SCHEDULER_ENABLED` environment variable

**Architecture Benefits:**

- **Production Ready**: All critical issues resolved through comprehensive code review
- **Context Management**: Jobs receive cancellable contexts for graceful shutdown
- **Dependency Injection**: Follows existing App struct pattern for service management
- **Clean Separation**: Jobs package contains concrete implementations, services handles scheduling
- **Existing Patterns**: Follows all established codebase patterns and conventions

### Technical Notes

- ✅ Uses gocron library (https://github.com/go-co-op/gocron) as requested
- ✅ Jobs are stateless and resumable with proper transaction handling
- ✅ No job overlap - daily execution with proper locking mechanisms
- ✅ Environment-based configuration for enabling/disabling scheduler
- ✅ Comprehensive code review completed - all critical issues resolved
- ✅ Unit tests implemented for job execution logic

---

## Ticket #3: Implement Discogs Data Download Service ✅ **COMPLETED**

**Priority:** High
**Story Points:** 8
**Status:** ✅ Complete - All Files with Concurrent Downloads (2025-09-14)

### Description

Build service to automatically download monthly Discogs data dumps with proper error handling and retry logic.

### Acceptance Criteria

**Phase 1 - Checksum Download (Completed):**
- [x] Implement HTTP client for downloading files from Discogs S3 bucket
- [x] Implement exponential backoff retry logic (immediate, 5min, 25min, 75min, 375min - max 5 attempts)
- [x] Add progress tracking and logging for download operations
- [x] Implement timeout handling for long-running downloads
- [x] Update processing table status throughout download lifecycle
- [x] Download and parse CHECKSUM.txt file
- [x] Store validated checksums in processing table for audit trail

**Phase 2 - Data File Downloads (Complete):**
- [x] Add streaming download capability to handle multi-GB files efficiently
- [x] Handle partial downloads and resume capability where possible
- [x] Download XML data files: artists.xml.gz, labels.xml.gz
- [x] Download XML data files: masters.xml.gz, releases.xml.gz
- [x] **NEW** Concurrent downloads using goroutines for 4x performance improvement

### Implementation Details

**Files Created:**
- `server/internal/services/download.service.go` - Core download service with HTTP client and validation
- `server/internal/services/download.service_test.go` - Comprehensive unit tests

**Files Modified:**
- `server/config/config.go` - **Cleaned up** - Removed Discogs environment variables
- `server/internal/jobs/discogsDownload.job.go` - Complete workflow with XML file downloads
- `server/internal/app/app.go` - Added download service to dependency injection
- `server/.env` - **Cleaned up** - Removed Discogs configuration bloat
- `server/internal/models/discogsDataProcessing.model.go` - Added GORM Scanner/Valuer interfaces and enhanced state tracking

**Key Features Implemented:**

- **Configuration Cleanup**: Replaced environment variables with package constants for immutable values
- **HTTP Client**: 300s timeout, proper User-Agent, connection pooling (MaxIdleConns: 10)
- **Exponential Backoff**: Retry schedule: immediate, 5min, 25min, 75min, 375min (max 5 attempts)
- **Streaming Downloads**: Memory-efficient with 32KB buffers for multi-GB files
- **SHA256 Validation**: Proper checksum verification against Discogs-provided hashes
- **File-Level State Tracking**: Individual file status for granular recovery
- **Smart Recovery**: Validates existing files on restart, skips re-downloading valid files
- **Progress Tracking**: Real-time logging with download progress updates every 30 seconds
- **File Management**: Downloads to `/tmp/discogs-{year-month}/` with intelligent cleanup
- **Status Management**: Robust state machine with validated transitions
- **GORM JSONB Support**: Fixed scanning issues with proper Scanner/Valuer interfaces
- **Transaction Safety**: All database operations wrapped in transactions
- **Production Ready**: Tech lead reviewed and approved for production deployment
- **🚀 NEW: Concurrent Downloads**: 4 files download in parallel using goroutines with proper error handling and synchronization
- **🚀 NEW: Progress-Based Timeout**: Smart timeout system allows unlimited time with progress, fails quickly on stalls (5min detection)
- **🚀 NEW: Docker Volume Storage**: Files persist across container restarts in `/app/discogs-data/` volume

**Architecture Integration:**

- **Dependency Injection**: Follows existing App struct pattern
- **Repository Pattern**: Uses existing `DiscogsDataProcessingRepository`
- **Logger Integration**: Uses project's structured logging package
- **Error Handling**: Follows existing patterns with proper error wrapping
- **Config Integration**: Uses `DiscogsTimeoutSec`, `DiscogsBaseURL`, etc.

**Test Coverage:**
- ✅ Service initialization with default and custom timeouts
- ✅ Checksum file parsing with various scenarios (valid, empty, missing files)
- ✅ Directory creation and file management
- ✅ Input validation and error handling
- ✅ Year-month format validation
- ✅ Edge cases and error scenarios

### Technical Notes

- **✅ PRODUCTION VERIFIED**: All 4 files successfully downloaded and validated (2025-09-14)
  - Artists: 441MB ✅ | Labels: 84MB ✅ | Masters: 578MB ✅ | Releases: 10.8GB ✅
- **Complete Implementation**: All 4 files (artists, labels, masters, releases) with full workflow (checksum → download → validation)
- **URL Patterns**:
  - Checksum: `https://discogs-data-dumps.s3-us-west-2.amazonaws.com/data/{YYYY}/discogs_{YYYYMMDD}_CHECKSUM.txt`
  - XML Files: `https://discogs-data-dumps.s3-us-west-2.amazonaws.com/data/{YYYY}/discogs_{YYYYMMDD}_{TYPE}.xml.gz`
- **Date Logic**: Uses current year-month (`time.Now().UTC().Format("2006-01")`) for URL construction
- **File Storage**: Docker volume persistent storage (`/app/discogs-data/{year-month}/`) - survives container restarts
- **Recovery**: Server restarts resume from last valid state, no unnecessary re-downloads
- **🚀 Performance Optimizations**:
  - **Concurrent Downloads**: 4 files download in parallel with goroutines and proper error handling
  - **Progress-Based Timeout**: 5-minute stall detection with unlimited time if progress continues
  - **Smart HTTP Timeout**: 1-hour safety net allows large files while detecting stalls quickly
  - **Streaming**: Memory-efficient 32KB buffers handle multi-GB files (10.8GB releases file verified)
- **Production Status**: Complete implementation verified working with real Discogs data

---

## Ticket #4: Implement Download Validation Service ✅ **COMPLETED**

**Priority:** High
**Story Points:** 3
**Status:** ✅ Complete - All Phases (2025-09-14)

### Description

Create validation service to verify downloaded files against Discogs-provided checksums before processing.

### Acceptance Criteria

**Phase 1 - Checksum Management (Completed):**
- [x] Download and parse CHECKSUM.txt file
- [x] Store validated checksums in processing table for audit trail
- [x] Add comprehensive error reporting for validation failures
- [x] Update processing status based on validation results
- [x] Log validation results and any discrepancies

**Phase 2 - File Validation (Completed):**
- [x] Implement checksum validation for each downloaded XML data file
- [x] Handle checksum mismatch scenarios (delete and retry)
- [x] Validate artists.xml.gz and labels.xml.gz against their checksums (masters/releases ready for future)
- [x] On validation failure, clean up invalid files and mark for re-download
- [x] **CRITICAL FIX**: Changed from MD5 to SHA256 validation to match Discogs checksums

### Implementation Details

**Complete Implementation (Both Phases):**
- ✅ **Checksum Download**: Successfully downloads CHECKSUM.txt from Discogs S3
- ✅ **SHA256 Validation**: **CRITICAL FIX** - Changed from MD5 to SHA256 to match Discogs format
- ✅ **Parsing Logic**: Parses checksum file format and extracts SHA256 hashes for all data files
- ✅ **Database Storage**: Stores checksums in `FileChecksums` JSONB field for audit trail
- ✅ **File Validation**: Implements `ValidateFileChecksum()` method with streaming SHA256 computation
- ✅ **Error Handling**: Comprehensive error reporting with computed vs expected checksum values
- ✅ **Cleanup Logic**: Automatically removes invalid files on checksum mismatch
- ✅ **Status Management**: Updates processing status based on validation results
- ✅ **Recovery Support**: Validates existing files on restart to prevent unnecessary re-downloads

**Integration Architecture:**
- Fully integrated into the download service workflow
- `ParseChecksumFile()` handles checksum extraction and validation
- `ValidateFileChecksum()` performs SHA256 verification of downloaded XML files
- Database updates occur within transactions for consistency
- File cleanup and status updates happen atomically

### Technical Notes

- **Complete Implementation**: Full checksum download, parsing, and file validation pipeline
- **SHA256 Algorithm**: **CRITICAL FIX** - Uses SHA256 (not MD5) to match Discogs checksum format
- **Streaming Validation**: Memory-efficient checksum computation for large files
- **Smart Recovery**: Existing valid files detected and reused on server restart
- **Error Recovery**: Failed validation triggers automatic cleanup and re-download on next run
- **Production Status**: Validated working with real Discogs data files (artists.xml.gz, labels.xml.gz)

---

## Ticket #5: Implement XML Data Processing Service

**Priority:** High  
**Story Points:** 13

### Description

Build service to parse and process Discogs XML data files, updating the database with artists, labels, masters, and releases information.

### Acceptance Criteria

- [ ] Implement streaming XML parser for large files (memory efficient)
- [ ] Create processing pipeline: Labels → Artists → Masters → Releases
- [ ] Implement upsert logic for all entity types based on Discogs IDs
- [ ] Handle foreign key relationships properly (releases → masters → artists/labels)
- [ ] Add batch processing for database operations (performance optimization)
- [ ] Implement progress tracking and status updates during processing
- [ ] Add comprehensive error handling and rollback capabilities
- [ ] Handle many-to-many relationships (release_artists, release_genres)

### Technical Notes

- Use Go's streaming XML parser to avoid loading entire files in memory
- Process files in dependency order to ensure foreign keys exist
- Consider using Go channels for concurrent processing where appropriate
- Files contain millions of records, optimize for batch database operations
- Plan for processing time of several hours for full monthly update

---

## Ticket #6: Implement File Cleanup Service

**Priority:** Medium  
**Story Points:** 2

### Description

Create service to clean up downloaded files after successful processing to manage disk space.

### Acceptance Criteria

- [ ] Automatically delete processed XML files after successful completion
- [ ] Implement configurable retention policy for downloaded files
- [ ] Add safety checks to prevent deletion of files during processing
- [ ] Handle cleanup failures gracefully without affecting main processing
- [ ] Add option to preserve files for debugging/manual inspection
- [ ] Log cleanup operations and any failures

### Technical Notes

- Only clean up files marked as successfully processed
- Consider keeping recent month's files for troubleshooting
- Add configuration option to disable cleanup for development/testing

---

## Dependencies & Integration Points

### Prerequisites

- Database models for Artists, Labels, Masters, Releases must exist
- Repository pattern implementations for all entities
- Basic CRUD operations for core entities

### External Dependencies

- Discogs S3 bucket accessibility
- Sufficient disk space (15-20GB temporary storage)
- Database performance for batch operations

### Testing Considerations

- Unit tests for each service component
- Integration tests with sample XML data
- Performance testing with large datasets
- Error scenario testing (network failures, corrupted files, disk space)

---

## Deployment & Monitoring

### Configuration Needed

- Cron job schedules
- Download retry limits and timeouts
- File storage locations
- Database batch sizes
- Cleanup retention policies

### Monitoring Requirements

- Job execution status and duration
- Download progress and failures
- Processing progress and performance metrics
- Disk space utilization
- Database performance during batch operations

### Rollback Strategy

- Ability to reprocess failed months
- Database transaction rollback for failed processing
- File retention for re-processing scenarios
