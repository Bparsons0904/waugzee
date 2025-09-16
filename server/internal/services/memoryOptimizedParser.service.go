package services

import (
	"fmt"
	"runtime"
	"time"
	"waugzee/internal/logger"
)

// MemoryOptimizedParsingService provides memory usage tracking and optimization utilities
type MemoryOptimizedParsingService struct {
	log logger.Logger
}

func NewMemoryOptimizedParsingService() *MemoryOptimizedParsingService {
	return &MemoryOptimizedParsingService{
		log: logger.New("memoryOptimizedParsingService"),
	}
}

// MemorySnapshot represents memory usage at a point in time
type MemorySnapshot struct {
	Timestamp    time.Time
	AllocMB      uint64
	HeapAllocMB  uint64
	SysMB        uint64
	NumGC        uint32
	RecordCount  int
	Description  string
}

// TakeMemorySnapshot captures current memory usage
func (s *MemoryOptimizedParsingService) TakeMemorySnapshot(recordCount int, description string) *MemorySnapshot {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return &MemorySnapshot{
		Timestamp:    time.Now(),
		AllocMB:      memStats.Alloc / 1024 / 1024,
		HeapAllocMB:  memStats.HeapAlloc / 1024 / 1024,
		SysMB:        memStats.Sys / 1024 / 1024,
		NumGC:        memStats.NumGC,
		RecordCount:  recordCount,
		Description:  description,
	}
}

// LogMemoryComparison logs the difference between two memory snapshots
func (s *MemoryOptimizedParsingService) LogMemoryComparison(before, after *MemorySnapshot) {
	allocDelta := int64(after.AllocMB) - int64(before.AllocMB)
	heapDelta := int64(after.HeapAllocMB) - int64(before.HeapAllocMB)
	sysDelta := int64(after.SysMB) - int64(before.SysMB)
	recordDelta := after.RecordCount - before.RecordCount
	timeDelta := after.Timestamp.Sub(before.Timestamp)
	
	s.log.Info("Memory usage comparison",
		"beforeDesc", before.Description,
		"afterDesc", after.Description,
		"timeDeltaMs", timeDelta.Milliseconds(),
		"recordsDelta", recordDelta,
		"allocMB", after.AllocMB,
		"allocDeltaMB", allocDelta,
		"heapAllocMB", after.HeapAllocMB,
		"heapDeltaMB", heapDelta,
		"sysMB", after.SysMB,
		"sysDeltaMB", sysDelta,
		"gcRuns", after.NumGC - before.NumGC,
		"memoryPerRecord", s.calculateMemoryPerRecord(allocDelta, recordDelta))
}

// calculateMemoryPerRecord calculates average memory usage per record
func (s *MemoryOptimizedParsingService) calculateMemoryPerRecord(memoryDeltaMB int64, recordDelta int) string {
	if recordDelta <= 0 {
		return "N/A"
	}
	
	bytesPerRecord := (memoryDeltaMB * 1024 * 1024) / int64(recordDelta)
	if bytesPerRecord < 1024 {
		return fmt.Sprintf("%d bytes/record", bytesPerRecord)
	} else if bytesPerRecord < 1024*1024 {
		return fmt.Sprintf("%.1f KB/record", float64(bytesPerRecord)/1024)
	} else {
		return fmt.Sprintf("%.1f MB/record", float64(bytesPerRecord)/(1024*1024))
	}
}

// EstimateMemoryReduction estimates memory savings from optimization
func (s *MemoryOptimizedParsingService) EstimateMemoryReduction(entityType string, recordCount int) {
	var estimatedSavingsPerRecord int64 // in bytes
	
	switch entityType {
	case "labels":
		// Skip Profile (avg ~500 bytes) + Website (avg ~50 bytes)
		estimatedSavingsPerRecord = 550
	case "artists":
		// Skip Biography (avg ~1KB) + Images (avg ~200 bytes per image, ~2 images)
		estimatedSavingsPerRecord = 1400
	case "masters":
		// Skip Notes (avg ~800 bytes) + Genres (avg ~5 genres * 50 bytes) + Artists (avg ~2 artists * 500 bytes)
		estimatedSavingsPerRecord = 2050
	case "releases":
		// Skip Tracks (avg ~10 tracks * 150 bytes) + Artists (avg ~2 artists * 500 bytes) + Genres (avg ~3 genres * 50 bytes)
		estimatedSavingsPerRecord = 2650
	}
	
	totalSavingsMB := (estimatedSavingsPerRecord * int64(recordCount)) / (1024 * 1024)
	
	s.log.Info("Estimated memory savings from optimization",
		"entityType", entityType,
		"recordCount", recordCount,
		"avgSavingsPerRecord", fmt.Sprintf("%d bytes", estimatedSavingsPerRecord),
		"totalEstimatedSavingsMB", totalSavingsMB)
}

// MinimalEntitySizes returns the estimated size of minimal entities in bytes
func (s *MemoryOptimizedParsingService) GetMinimalEntitySizes() map[string]int {
	return map[string]int{
		"label":   100,  // UUID(16) + Name(~30) + DiscogsID(8) + overhead
		"artist":  110,  // UUID(16) + Name(~40) + DiscogsID(8) + IsActive(1) + overhead  
		"master":  150,  // UUID(16) + Title(~50) + DiscogsID(8) + MainRelease(8) + Year(4) + overhead
		"release": 200,  // UUID(16) + Title(~50) + DiscogsID(8) + basic fields + overhead
	}
}

// ForceGarbageCollection forces garbage collection and logs memory before/after
func (s *MemoryOptimizedParsingService) ForceGarbageCollection(description string) *MemorySnapshot {
	beforeGC := s.TakeMemorySnapshot(0, fmt.Sprintf("before GC - %s", description))
	
	runtime.GC()
	runtime.GC() // Run twice to ensure complete cleanup
	
	afterGC := s.TakeMemorySnapshot(0, fmt.Sprintf("after GC - %s", description))
	
	s.LogMemoryComparison(beforeGC, afterGC)
	
	return afterGC
}