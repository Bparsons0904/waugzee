# XML Processing Infrastructure Removal Plan

## Overview
Remove all XML downloading and parsing infrastructure to return to the original on-demand approach for Discogs data integration.

## Files to Delete

### Core Services & Repositories
- `server/internal/services/xmlProcessing.service.go` - XML processing service
- `server/internal/services/download.service.go` - File download service
- `server/internal/services/download.service_test.go` - Download service tests
- `server/internal/services/discogsParser.service.go` - Discogs XML parser
- `server/internal/repositories/discogsDataProcessing.repository.go` - Processing repository
- `server/internal/models/discogsDataProcessing.model.go` - Processing models

### Job System
- `server/internal/jobs/discogsDownload.job.go` - Download job
- `server/internal/jobs/discogsProcessing.job.go` - Processing job
- `server/internal/jobs/scheduler.jog.go` - Job registration (entire jobs system)

### Documentation
- `docs/XML_PROCESSING_SERVICE.md` - XML processing architecture docs
- `docs/DISCOGS_PROCESSING_SIMPLIFICATION.md` - Processing docs
- `docs/CONCURRENT_PROCESSING_ANALYSIS.md` - Concurrent processing analysis

### Data Directory
- `server/discogs-data/` - XML data storage directory (entire directory)

## Code Modifications

### App Dependencies (`server/internal/app/app.go`)
**Remove from App struct:**
- `DiscogsParserService *services.DiscogsParserService`
- `DownloadService *services.DownloadService`
- `XMLProcessingService *services.XMLProcessingService`
- `DiscogsDataProcessingRepo repositories.DiscogsDataProcessingRepository`

**Remove from initialization:**
- All service/repository instantiation for removed components
- Job registration call: `jobs.RegisterAllJobs(...)`
- Validation checks for removed services
- Close() method cleanup for removed services

### Database Models (`server/internal/models/`)
**Remove imports/references:**
- Any imports of `discogsDataProcessing.model.go`
- Related model relationships if they exist

## Expected Impact

### Positive Outcomes
- Simplified codebase focused on core vinyl collection features
- Reduced complexity and maintenance burden
- Return to proven on-demand Discogs API integration
- Eliminated complex XML processing infrastructure

### Cleanup Required
- Database migration to remove processing tables (if any exist)
- Configuration cleanup for removed services
- Remove any environment variables related to XML processing

## Next Steps After Removal
1. Verify application builds and runs correctly
2. Update documentation to reflect simplified architecture
3. Focus development on core collection management features
4. Implement on-demand Discogs API integration as needed