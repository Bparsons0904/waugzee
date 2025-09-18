# Simplified XML Processing Service Architecture

## Overview

The Simplified XML Processing Service is responsible for parsing large Discogs XML data files and efficiently storing the data in PostgreSQL. The service uses a streaming, buffered channel architecture to handle millions of records while maintaining controlled memory usage and preventing database deadlocks.

## Core Architecture

### Single-Threaded Buffer Processing

The service operates on a **single-threaded buffer processing model** with the following key principles:

1. **One goroutine per entity type** - Each entity type (artists, masters, releases, etc.) has exactly one processing goroutine
2. **Fixed batch sizes** - Each processor maintains strict batch size limits to control transaction sizes
3. **No cross-products** - Associations are processed as exact pairs, never as cross-joins
4. **Ordered processing** - All database operations use consistent ordering to prevent deadlocks

### Entity Processing Flow

```
XML File → Parser → Entity Channels → Buffer Processors → Repository Batches → Database
```

#### 1. XML Parsing
- Streams XML data without loading entire file into memory
- Sends parsed entities to appropriate buffered channels
- Processes vinyl releases only (filters out CD/digital/cassette)

#### 2. Entity Channels
- **Labels**: 5000 record buffer
- **Artists**: 1000 record buffer
- **Masters**: 5000 record buffer
- **Releases**: 1000 record buffer (smaller due to JSONB data)
- **Images**: 5000 record buffer
- **Genres**: 5000 record buffer

#### 3. Association Channels
- **Master-Artist**: 1000 association buffer
- **Master-Genre**: 1000 association buffer

## Association Processing Architecture

### Master-Artist Associations

**Critical Design**: Associations are processed as **exact pairs**, not cross-products.

#### How It Works:
1. **XML Parsing**: When processing a Master record, extract each artist relationship
2. **Association Creation**: Create `MasterArtistAssociation{MasterDiscogsID: 123, ArtistDiscogsID: 456}`
3. **Buffer Accumulation**: Collect exactly 1000 association pairs
4. **Batch Processing**: Insert the exact 1000 associations (no joins)

#### What We Avoid:
```go
// ❌ WRONG: Cross-product approach (creates millions of unwanted associations)
allMasters := []int64{1, 2, 3, ...}      // 1000 masters
allArtists := []int64{100, 101, 102, ...} // 1077 artists
// Results in: 1000 × 1077 = 1,077,000 associations

// ✅ CORRECT: Exact association approach
associations := []MasterArtistAssociation{
    {MasterDiscogsID: 1, ArtistDiscogsID: 100},
    {MasterDiscogsID: 1, ArtistDiscogsID: 101},
    {MasterDiscogsID: 2, ArtistDiscogsID: 100},
    // ... exactly 1000 real associations
}
```

### Deadlock Prevention Strategy

#### 1. Consistent Ordering
All database operations use `ORDER BY` to ensure consistent lock acquisition:
```sql
-- Repository operations always order IDs
INSERT INTO master_artists (master_discogs_id, artist_discogs_id)
SELECT master_id, artist_id
FROM unnest($1::bigint[], $2::bigint[]) AS t(master_id, artist_id)
ORDER BY master_id, artist_id
ON CONFLICT DO NOTHING
```

#### 2. Controlled Batch Sizes
- **Artists**: 1000 records per batch
- **Masters**: 5000 records per batch
- **Releases**: 1000 records per batch (JSONB data is larger)
- **Associations**: 1000 pairs per batch

#### 3. Single-Threaded Processing
- One goroutine per entity type eliminates race conditions
- No concurrent access to the same database tables
- Predictable lock acquisition patterns

## Data Flow Examples

### Processing a Master Record

```go
// 1. XML contains master with 3 artists
rawMaster := &imports.Master{
    ID: 123,
    Title: "Abbey Road",
    Artists: []imports.Artist{
        {ID: 456, Name: "The Beatles"},
        {ID: 789, Name: "George Martin"},
        {ID: 101, Name: "Geoff Emerick"},
    },
}

// 2. Service creates exact associations
associations := []MasterArtistAssociation{
    {MasterDiscogsID: 123, ArtistDiscogsID: 456},
    {MasterDiscogsID: 123, ArtistDiscogsID: 789},
    {MasterDiscogsID: 123, ArtistDiscogsID: 101},
}

// 3. Buffer accumulates until 1000 associations collected
// 4. Repository inserts exactly 1000 association pairs
```

### Release Processing with JSONB

```go
// 1. XML contains release with track data
rawRelease := &imports.Release{
    ID: 456,
    Title: "Abbey Road",
    TrackList: []imports.Track{
        {Position: "A1", Title: "Come Together", Duration: "4:20"},
        {Position: "A2", Title: "Something", Duration: "3:03"},
    },
    Artists: []imports.Artist{...},
    Genres: []string{"Rock", "Pop"},
}

// 2. Service converts to Release model with JSONB fields
release := &models.Release{
    DiscogsID: 456,
    Title: "Abbey Road",
    TracksJSON: `[{"position":"A1","title":"Come Together","duration":"4:20"}...]`,
    ArtistsJSON: `[{"id":456,"name":"The Beatles"}...]`,
    GenresJSON: `["Rock","Pop"]`,
}

// 3. No separate Track table - all stored as JSONB
```

## Performance Characteristics

### Memory Usage
- **Streaming Processing**: No complete file loading into memory
- **Fixed Buffer Sizes**: Predictable memory footprint
- **JSONB Storage**: Eliminates need for separate Track table and relationships

### Database Performance
- **Batch Operations**: All inserts/updates use batch processing
- **Hash-Based Upserts**: Efficient change detection using content hashes
- **Minimal Associations**: Only Master-level relationships maintained

### Processing Speed
- **Vinyl-Only Filtering**: 70-80% volume reduction by skipping non-vinyl formats
- **Concurrent Buffers**: Multiple entity types processed simultaneously
- **Optimized SQL**: Proper indexing and ordered operations

## Configuration

### Buffer Sizes
```go
const (
    ImageBufferSize = 10000
    GenreBufferSize = 10000
    ArtistBufferSize = 10000
    LabelBufferSize = 10000
    MasterBufferSize = 10000
    ReleaseBufferSize = 10000
    MasterArtistAssociationBufferSize = 10000
    MasterGenreAssociationBufferSize = 10000
)
```

### Batch Processing Limits
```go
const (
    ArtistBatchSize = 1000
    MasterBatchSize = 5000
    ReleaseBatchSize = 1000  // Smaller due to JSONB
    AssociationBatchSize = 1000
)
```

## Error Handling

### Deadlock Recovery
- Repository operations include retry logic for deadlock detection
- Consistent ordering prevents most deadlock scenarios
- Batch size limits reduce lock contention

### Processing Errors
- Invalid records are logged and skipped
- Processing continues with valid records
- Final statistics include error counts

### Memory Management
- Buffers are properly closed after processing
- Maps are reset after each batch to prevent memory leaks
- Goroutines are properly cleaned up with WaitGroup

## Monitoring and Observability

### Logging
- Batch processing progress with counts
- Error rates and types
- Memory usage statistics
- Processing duration metrics

### Metrics
- Records processed per entity type
- Batch sizes and processing times
- Error rates and retry attempts
- Memory allocation patterns

This architecture ensures reliable, efficient processing of large Discogs datasets while maintaining data integrity and preventing database deadlocks.