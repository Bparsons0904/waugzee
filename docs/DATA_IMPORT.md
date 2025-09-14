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

## Ticket #2: Implement Cron Job Scheduling Service

**Priority:** High  
**Story Points:** 5

### Description

Create a scheduling service to manage periodic tasks for Discogs data downloading and processing.

### Acceptance Criteria

- [ ] Design and implement cron job scheduling architecture
- [ ] Create daily job for checking/downloading monthly data
- [ ] Create processing job for parsing downloaded files
- [ ] Add configuration for job timing and intervals
- [ ] Implement proper logging and monitoring for scheduled tasks
- [ ] Add graceful shutdown handling for running jobs
- [ ] Include job status reporting and health checks

### Technical Notes

- Consider using Go's built-in time package or cron library
- Jobs should be stateless and resumable
- Ensure jobs don't overlap or conflict with each other
- Add environment-based configuration for job scheduling

---

## Ticket #3: Implement Discogs Data Download Service

**Priority:** High  
**Story Points:** 8

### Description

Build service to automatically download monthly Discogs data dumps with proper error handling and retry logic.

### Acceptance Criteria

- [ ] Implement HTTP client for downloading large files from Discogs S3 bucket
- [ ] Add streaming download capability to handle multi-GB files efficiently
- [ ] Implement exponential backoff retry logic (max 5 attempts)
- [ ] Handle partial downloads and resume capability where possible
- [ ] Add progress tracking and logging for download operations
- [ ] Implement timeout handling for long-running downloads
- [ ] Update processing table status throughout download lifecycle

### Technical Notes

- Downloads: artists.xml.gz, labels.xml.gz, masters.xml.gz, releases.xml.gz, CHECKSUM.txt
- Use URL pattern: `https://discogs-data-dumps.s3-us-west-2.amazonaws.com/data/{YYYY}/discogs_{YYYYMMDD}_{type}.xml.gz`
- Files can be 1-5GB each, plan for appropriate timeouts
- Consider disk space management during downloads

---

## Ticket #4: Implement Download Validation Service

**Priority:** High  
**Story Points:** 3

### Description

Create validation service to verify downloaded files against Discogs-provided checksums before processing.

### Acceptance Criteria

- [ ] Download and parse CHECKSUM.txt file
- [ ] Implement checksum validation for each downloaded file
- [ ] Handle checksum mismatch scenarios (delete and retry)
- [ ] Add comprehensive error reporting for validation failures
- [ ] Update processing status based on validation results
- [ ] Log validation results and any discrepancies

### Technical Notes

- Validate all 4 data files against their checksums
- On validation failure, clean up invalid files and mark for re-download
- Store validated checksums in processing table for audit trail

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
