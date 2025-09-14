# Discogs Monthly Data Processing - Work Tickets

## Epic Overview

Implement automated monthly processing of Discogs data dumps to populate the application's core music database with artists, labels, masters, and releases data.

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

## Ticket #3: Implement Discogs Data Download Service ✅ **PHASE 1 COMPLETED**

**Priority:** High  
**Story Points:** 8  
**Status:** ✅ Phase 1 Complete - Checksum Download (2025-09-13)

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

**Phase 2 - Data File Downloads (Pending):**
- [ ] Add streaming download capability to handle multi-GB files efficiently
- [ ] Handle partial downloads and resume capability where possible
- [ ] Download XML data files: artists.xml.gz, labels.xml.gz, masters.xml.gz, releases.xml.gz

### Implementation Details

**Files Created:**
- `server/internal/services/download.service.go` - Core download service with HTTP client
- `server/internal/services/download.service_test.go` - Comprehensive unit tests

**Files Modified:**
- `server/config/config.go` - Added Discogs download configuration fields
- `server/internal/jobs/discogsDownload.job.go` - Integrated download service with job
- `server/internal/app/app.go` - Added download service to dependency injection
- `server/.env` - Added Discogs configuration (base URL, timeout, retries, download directory)

**Key Features Implemented:**

- **HTTP Client**: Configurable timeout (300s default), proper User-Agent, connection pooling
- **Exponential Backoff**: Retry schedule: immediate, 5min, 25min, 75min, 375min (max 5 attempts)
- **Progress Tracking**: Real-time logging with download progress updates every 30 seconds  
- **File Management**: Downloads to `/tmp/discogs-{year-month}/` with original Discogs filenames
- **Checksum Processing**: Parses CHECKSUM.txt and stores in `FileChecksums` JSONB field
- **Status Management**: Proper transitions (`not_started` → `downloading` → `ready_for_processing`/`failed`)
- **Concurrent Support**: Multiple month processing simultaneously (no Discogs limits)
- **Context Cancellation**: Full support for graceful shutdown and cancellation
- **Transaction Safety**: All database operations wrapped in transactions

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

- **Current Implementation**: CHECKSUM.txt download and parsing fully functional
- **URL Pattern**: `https://discogs-data-dumps.s3-us-west-2.amazonaws.com/data/{YYYY}/discogs_{YYYYMMDD}_CHECKSUM.txt`
- **Date Logic**: Always uses current year-month (`time.Now().UTC().Format("2006-01")`)
- **File Storage**: Container temp storage (`/tmp/discogs-{year-month}/`)
- **Next Phase**: Ready to extend for XML data file downloads (artists, labels, masters, releases)
- **Performance**: Sub-second checksum downloads with comprehensive error handling

---

## Ticket #4: Implement Download Validation Service ✅ **PHASE 1 COMPLETED**

**Priority:** High  
**Story Points:** 3  
**Status:** ✅ Phase 1 Complete - Checksum Management (2025-09-13)

### Description

Create validation service to verify downloaded files against Discogs-provided checksums before processing.

### Acceptance Criteria

**Phase 1 - Checksum Management (Completed):**
- [x] Download and parse CHECKSUM.txt file
- [x] Store validated checksums in processing table for audit trail
- [x] Add comprehensive error reporting for validation failures
- [x] Update processing status based on validation results
- [x] Log validation results and any discrepancies

**Phase 2 - File Validation (Pending):**
- [ ] Implement checksum validation for each downloaded XML data file
- [ ] Handle checksum mismatch scenarios (delete and retry)
- [ ] Validate all 4 data files (artists, labels, masters, releases) against their checksums
- [ ] On validation failure, clean up invalid files and mark for re-download

### Implementation Details

**Current Implementation (Phase 1):**
- ✅ **Checksum Download**: Successfully downloads CHECKSUM.txt from Discogs S3
- ✅ **Parsing Logic**: Parses checksum file format and extracts MD5 hashes for all data files
- ✅ **Database Storage**: Stores checksums in `FileChecksums` JSONB field for audit trail
- ✅ **Error Handling**: Comprehensive error reporting for download and parsing failures
- ✅ **Status Management**: Updates processing status based on checksum download results
- ✅ **Logging**: Detailed logging of validation results and any discrepancies

**Integration with Ticket #3:**
- The checksum validation functionality has been integrated into the download service
- `ParseChecksumFile()` method handles checksum extraction and validation
- Database updates occur within the same transaction as the download process

### Technical Notes

- **Phase 1 Complete**: Checksum download and storage fully functional
- **Phase 2 Ready**: Infrastructure in place for XML file validation when data files are downloaded
- **Validation Strategy**: Compare downloaded file MD5 hashes against stored checksums
- **Error Recovery**: Failed validation will trigger cleanup and re-download process

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
