package adminController

import (
	"context"
	"fmt"
	"time"
	"waugzee/internal/logger"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

type AdminController struct {
	parser                    *services.DiscogsParserService
	xmlProcessing             *services.XMLProcessingService
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository
	transaction               *services.TransactionService
	log                       logger.Logger
}

func New(
	parser *services.DiscogsParserService,
	xmlProcessing *services.XMLProcessingService,
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository,
	transaction *services.TransactionService,
) *AdminController {
	return &AdminController{
		parser:                    parser,
		xmlProcessing:             xmlProcessing,
		discogsDataProcessingRepo: discogsDataProcessingRepo,
		transaction:               transaction,
		log:                       logger.New("adminController"),
	}
}

// Limits represents optional processing limits
type Limits struct {
	MaxRecords   int `json:"maxRecords,omitempty"`   // Optional: limit total records parsed (0 = no limit)
	MaxBatchSize int `json:"maxBatchSize,omitempty"` // Optional: batch size for processing (default: 2000)
}

// SimplifiedRequest represents the new unified request format for both endpoints
type SimplifiedRequest struct {
	FileTypes []string `json:"fileTypes" validate:"required,min=1"`
	Limits    *Limits  `json:"limits,omitempty"`
}

// LEGACY: ParseFileRequest represents a request to parse a single file (kept for compatibility)
type ParseFileRequest struct {
	FilePath   string `json:"filePath" validate:"required"`
	FileType   string `json:"fileType" validate:"required,oneof=labels artists masters releases"`
	BatchSize  int    `json:"batchSize,omitempty"`   // Optional: batch size for processing (default: 2000)
	MaxRecords int    `json:"maxRecords,omitempty"`  // Optional: limit total records parsed (0 = no limit)
}

// LEGACY: ProcessWorkflowRequest represents a request to run the full processing workflow (kept for compatibility)
type ProcessWorkflowRequest struct {
	YearMonth string `json:"yearMonth" validate:"required"`
}

// FileResult represents the result of processing a single file type
type FileResult struct {
	FileType         string   `json:"fileType"`
	FilePath         string   `json:"filePath"`
	Success          bool     `json:"success"`
	TotalRecords     int      `json:"totalRecords"`
	ProcessedRecords int      `json:"processedRecords"`
	ErroredRecords   int      `json:"erroredRecords"`
	Errors           []string `json:"errors"`
}

// SimplifiedParseResponse represents the response from the simplified parse endpoint
type SimplifiedParseResponse struct {
	Success     bool         `json:"success"`
	Message     string       `json:"message"`
	YearMonth   string       `json:"yearMonth"`
	FileResults []FileResult `json:"fileResults"`
	Summary     struct {
		TotalFiles       int `json:"totalFiles"`
		SuccessfulFiles  int `json:"successfulFiles"`
		TotalRecords     int `json:"totalRecords"`
		ProcessedRecords int `json:"processedRecords"`
		ErroredRecords   int `json:"erroredRecords"`
	} `json:"summary"`
}

// SimplifiedProcessResponse represents the response from the simplified process endpoint
type SimplifiedProcessResponse struct {
	Success          bool         `json:"success"`
	Message          string       `json:"message"`
	YearMonth        string       `json:"yearMonth"`
	ProcessingID     string       `json:"processingId,omitempty"`
	FileResults      []FileResult `json:"fileResults"`
	Summary          struct {
		TotalFiles       int `json:"totalFiles"`
		SuccessfulFiles  int `json:"successfulFiles"`
		TotalRecords     int `json:"totalRecords"`
		ProcessedRecords int `json:"processedRecords"`
		InsertedRecords  int `json:"insertedRecords"`
		UpdatedRecords   int `json:"updatedRecords"`
		ErroredRecords   int `json:"erroredRecords"`
	} `json:"summary"`
}

// LEGACY: ParseFileResponse represents the response from parsing a file (kept for compatibility)
type ParseFileResponse struct {
	Success          bool                         `json:"success"`
	Message          string                       `json:"message"`
	TotalRecords     int                          `json:"totalRecords"`
	ProcessedRecords int                          `json:"processedRecords"`
	ErroredRecords   int                          `json:"erroredRecords"`
	Errors           []string                     `json:"errors"`
	ParsedData       *services.ParseResult        `json:"parsedData,omitempty"` // Optional: include parsed data
}

// LEGACY: ProcessWorkflowResponse represents the response from running the full workflow (kept for compatibility)
type ProcessWorkflowResponse struct {
	Success          bool     `json:"success"`
	Message          string   `json:"message"`
	ProcessingID     string   `json:"processingId,omitempty"`
	TotalRecords     int      `json:"totalRecords"`
	ProcessedRecords int      `json:"processedRecords"`
	InsertedRecords  int      `json:"insertedRecords"`
	UpdatedRecords   int      `json:"updatedRecords"`
	ErroredRecords   int      `json:"erroredRecords"`
	Errors           []string `json:"errors"`
}

// getCurrentYearMonth returns the current year-month in YYYY-MM format
func (ac *AdminController) getCurrentYearMonth() string {
	return time.Now().Format("2006-01")
}

// buildFilePath constructs the file path for a given file type using current year-month
func (ac *AdminController) buildFilePath(fileType string) string {
	yearMonth := ac.getCurrentYearMonth()
	return fmt.Sprintf("/app/discogs-data/%s/%s.xml.gz", yearMonth, fileType)
}

// validateFileTypes checks if all provided file types are valid
func (ac *AdminController) validateFileTypes(fileTypes []string) error {
	validFileTypes := map[string]bool{
		"labels":   true,
		"artists":  true,
		"masters":  true,
		"releases": true,
	}

	for _, fileType := range fileTypes {
		if !validFileTypes[fileType] {
			return fmt.Errorf("invalid fileType '%s': must be one of: labels, artists, masters, releases", fileType)
		}
	}

	return nil
}

// SimplifiedParse performs direct parsing of multiple files without database operations
func (ac *AdminController) SimplifiedParse(ctx context.Context, req SimplifiedRequest) (*SimplifiedParseResponse, error) {
	log := ac.log.Function("SimplifiedParse")

	// Validate file types
	if err := ac.validateFileTypes(req.FileTypes); err != nil {
		return nil, err
	}

	yearMonth := ac.getCurrentYearMonth()

	log.Info("Starting simplified parsing for multiple files",
		"yearMonth", yearMonth,
		"fileTypes", req.FileTypes,
		"limits", req.Limits)

	var fileResults []FileResult
	var summary struct {
		TotalFiles       int
		SuccessfulFiles  int
		TotalRecords     int
		ProcessedRecords int
		ErroredRecords   int
	}

	summary.TotalFiles = len(req.FileTypes)

	// Process each file type
	for _, fileType := range req.FileTypes {
		filePath := ac.buildFilePath(fileType)

		log.Info("Processing file",
			"fileType", fileType,
			"filePath", filePath)

		// Configure parsing options
		options := services.ParseOptions{
			FilePath: filePath,
			FileType: fileType,
		}

		// Apply limits if provided
		if req.Limits != nil {
			if req.Limits.MaxRecords > 0 {
				options.MaxRecords = req.Limits.MaxRecords
			}
			if req.Limits.MaxBatchSize > 0 {
				options.BatchSize = req.Limits.MaxBatchSize
			}
		}

		// Parse the file
		result, err := ac.parser.ParseFile(ctx, options)

		fileResult := FileResult{
			FileType: fileType,
			FilePath: filePath,
		}

		if err != nil {
			fileResult.Success = false
			fileResult.Errors = []string{fmt.Sprintf("Failed to parse file: %v", err)}
			_ = log.Error("File parsing failed",
				"error", err,
				"fileType", fileType,
				"filePath", filePath)
		} else {
			fileResult.Success = true
			fileResult.TotalRecords = result.TotalRecords
			fileResult.ProcessedRecords = result.ProcessedRecords
			fileResult.ErroredRecords = result.ErroredRecords
			fileResult.Errors = result.Errors

			summary.SuccessfulFiles++
			summary.TotalRecords += result.TotalRecords
			summary.ProcessedRecords += result.ProcessedRecords
			summary.ErroredRecords += result.ErroredRecords

			log.Info("File parsing completed",
				"fileType", fileType,
				"totalRecords", result.TotalRecords,
				"processedRecords", result.ProcessedRecords,
				"erroredRecords", result.ErroredRecords)
		}

		fileResults = append(fileResults, fileResult)
	}

	success := summary.SuccessfulFiles == summary.TotalFiles
	message := fmt.Sprintf("Parsed %d/%d files successfully", summary.SuccessfulFiles, summary.TotalFiles)

	response := &SimplifiedParseResponse{
		Success:     success,
		Message:     message,
		YearMonth:   yearMonth,
		FileResults: fileResults,
	}
	response.Summary.TotalFiles = summary.TotalFiles
	response.Summary.SuccessfulFiles = summary.SuccessfulFiles
	response.Summary.TotalRecords = summary.TotalRecords
	response.Summary.ProcessedRecords = summary.ProcessedRecords
	response.Summary.ErroredRecords = summary.ErroredRecords

	log.Info("Simplified parsing completed",
		"yearMonth", yearMonth,
		"totalFiles", summary.TotalFiles,
		"successfulFiles", summary.SuccessfulFiles,
		"totalRecords", summary.TotalRecords,
		"processedRecords", summary.ProcessedRecords,
		"erroredRecords", summary.ErroredRecords)

	return response, nil
}

// SimplifiedProcess performs the full processing workflow with current date
func (ac *AdminController) SimplifiedProcess(ctx context.Context, req SimplifiedRequest) (*SimplifiedProcessResponse, error) {
	log := ac.log.Function("SimplifiedProcess")

	// Validate file types
	if err := ac.validateFileTypes(req.FileTypes); err != nil {
		return nil, err
	}

	yearMonth := ac.getCurrentYearMonth()

	log.Info("Starting simplified full processing workflow",
		"yearMonth", yearMonth,
		"fileTypes", req.FileTypes,
		"limits", req.Limits)

	// Find or create the processing record for current year-month
	processing, err := ac.discogsDataProcessingRepo.GetByYearMonth(ctx, yearMonth)
	if err != nil {
		return &SimplifiedProcessResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to find processing record for %s: %v", yearMonth, err),
			YearMonth: yearMonth,
		}, log.Err("failed to find processing record", err, "yearMonth", yearMonth)
	}

	var fileResults []FileResult
	var summary struct {
		TotalFiles       int
		SuccessfulFiles  int
		TotalRecords     int
		ProcessedRecords int
		InsertedRecords  int
		UpdatedRecords   int
		ErroredRecords   int
	}

	summary.TotalFiles = len(req.FileTypes)

	// Map file types to their processing methods with limits support
	processingMethods := map[string]func(context.Context, string, string, *services.ProcessingLimits) (*services.ProcessingResult, error){
		"labels":   ac.xmlProcessing.ProcessLabelsFileWithLimits,
		"artists":  ac.xmlProcessing.ProcessArtistsFileWithLimits,
		"masters":  ac.xmlProcessing.ProcessMastersFileWithLimits,
		"releases": ac.xmlProcessing.ProcessReleasesFileWithLimits,
	}

	// Process each requested file type
	for _, fileType := range req.FileTypes {
		filePath := ac.buildFilePath(fileType)
		method, exists := processingMethods[fileType]
		if !exists {
			continue // This should not happen due to validation, but safety first
		}

		log.Info("Starting XML processing",
			"fileType", fileType,
			"yearMonth", yearMonth,
			"filePath", filePath)

		fileResult := FileResult{
			FileType: fileType,
			FilePath: filePath,
		}

		// Convert limits to service format
		var serviceLimits *services.ProcessingLimits
		if req.Limits != nil {
			serviceLimits = &services.ProcessingLimits{
				MaxRecords:   req.Limits.MaxRecords,
				MaxBatchSize: req.Limits.MaxBatchSize,
			}
		}

		// Process the XML file with limits
		result, err := method(ctx, filePath, processing.ID.String(), serviceLimits)
		if err != nil {
			fileResult.Success = false
			fileResult.Errors = []string{fmt.Sprintf("Failed to process file: %v", err)}
			_ = log.Error("XML processing failed",
				"error", err,
				"fileType", fileType,
				"filePath", filePath)
		} else {
			fileResult.Success = true
			fileResult.TotalRecords = result.TotalRecords
			fileResult.ProcessedRecords = result.ProcessedRecords
			fileResult.ErroredRecords = result.ErroredRecords
			fileResult.Errors = result.Errors

			summary.SuccessfulFiles++
			summary.TotalRecords += result.TotalRecords
			summary.ProcessedRecords += result.ProcessedRecords
			summary.InsertedRecords += result.InsertedRecords
			summary.UpdatedRecords += result.UpdatedRecords
			summary.ErroredRecords += result.ErroredRecords

			log.Info("XML processing completed",
				"fileType", fileType,
				"yearMonth", yearMonth,
				"totalRecords", result.TotalRecords,
				"processedRecords", result.ProcessedRecords,
				"insertedRecords", result.InsertedRecords,
				"updatedRecords", result.UpdatedRecords,
				"erroredRecords", result.ErroredRecords)
		}

		fileResults = append(fileResults, fileResult)
	}

	success := summary.SuccessfulFiles == summary.TotalFiles
	message := fmt.Sprintf("Processed %d/%d files successfully", summary.SuccessfulFiles, summary.TotalFiles)

	response := &SimplifiedProcessResponse{
		Success:      success,
		Message:      message,
		YearMonth:    yearMonth,
		ProcessingID: processing.ID.String(),
		FileResults:  fileResults,
	}
	response.Summary.TotalFiles = summary.TotalFiles
	response.Summary.SuccessfulFiles = summary.SuccessfulFiles
	response.Summary.TotalRecords = summary.TotalRecords
	response.Summary.ProcessedRecords = summary.ProcessedRecords
	response.Summary.InsertedRecords = summary.InsertedRecords
	response.Summary.UpdatedRecords = summary.UpdatedRecords
	response.Summary.ErroredRecords = summary.ErroredRecords

	log.Info("Simplified full processing workflow completed",
		"yearMonth", yearMonth,
		"processingID", processing.ID.String(),
		"totalFiles", summary.TotalFiles,
		"successfulFiles", summary.SuccessfulFiles,
		"totalRecords", summary.TotalRecords,
		"processedRecords", summary.ProcessedRecords,
		"insertedRecords", summary.InsertedRecords,
		"updatedRecords", summary.UpdatedRecords,
		"erroredRecords", summary.ErroredRecords)

	return response, nil
}

// LEGACY METHODS (kept for backward compatibility)

// ParseFile performs direct parsing without database operations or workflow management
func (ac *AdminController) ParseFile(ctx context.Context, req ParseFileRequest) (*ParseFileResponse, error) {
	log := ac.log.Function("ParseFile")

	log.Info("Starting direct file parsing",
		"filePath", req.FilePath,
		"fileType", req.FileType,
		"batchSize", req.BatchSize,
		"maxRecords", req.MaxRecords)

	// Configure parsing options
	options := services.ParseOptions{
		FilePath:   req.FilePath,
		FileType:   req.FileType,
		BatchSize:  req.BatchSize,
		MaxRecords: req.MaxRecords,
	}

	// Parse the file using the pure parser service
	result, err := ac.parser.ParseFile(ctx, options)
	if err != nil {
		return &ParseFileResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse file: %v", err),
		}, log.Err("parsing failed", err)
	}

	response := &ParseFileResponse{
		Success:          true,
		Message:          "File parsed successfully",
		TotalRecords:     result.TotalRecords,
		ProcessedRecords: result.ProcessedRecords,
		ErroredRecords:   result.ErroredRecords,
		Errors:           result.Errors,
		ParsedData:       result, // Include full parsed data for inspection
	}

	log.Info("Direct file parsing completed",
		"filePath", req.FilePath,
		"fileType", req.FileType,
		"totalRecords", result.TotalRecords,
		"processedRecords", result.ProcessedRecords,
		"erroredRecords", result.ErroredRecords)

	return response, nil
}

// ProcessWorkflow runs the full processing workflow (existing functionality)
func (ac *AdminController) ProcessWorkflow(ctx context.Context, req ProcessWorkflowRequest) (*ProcessWorkflowResponse, error) {
	log := ac.log.Function("ProcessWorkflow")

	log.Info("Starting full processing workflow", "yearMonth", req.YearMonth)

	// Find the processing record for this year-month
	processing, err := ac.discogsDataProcessingRepo.GetByYearMonth(ctx, req.YearMonth)
	if err != nil {
		return &ProcessWorkflowResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to find processing record for %s: %v", req.YearMonth, err),
		}, log.Err("failed to find processing record", err, "yearMonth", req.YearMonth)
	}

	// Determine download directory
	downloadDir := fmt.Sprintf("/app/discogs-data/%s", req.YearMonth)

	// Process all XML files in dependency order: Labels → Artists → Masters → Releases
	fileTypes := []struct {
		name   string
		method func(context.Context, string, string) (*services.ProcessingResult, error)
	}{
		{"labels", ac.xmlProcessing.ProcessLabelsFile},
		{"artists", ac.xmlProcessing.ProcessArtistsFile},
		{"masters", ac.xmlProcessing.ProcessMastersFile},
		{"releases", ac.xmlProcessing.ProcessReleasesFile},
	}

	var totalProcessed int
	var totalInserted int
	var totalUpdated int
	var totalErrors int
	var errors []string

	// Process each file type in dependency order
	for _, fileType := range fileTypes {
		filePath := fmt.Sprintf("%s/%s.xml.gz", downloadDir, fileType.name)

		log.Info("Starting XML processing",
			"fileType", fileType.name,
			"yearMonth", req.YearMonth,
			"filePath", filePath)

		// Process the XML file
		result, err := fileType.method(ctx, filePath, processing.ID.String())
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to process %s file: %v", fileType.name, err)
			errors = append(errors, errorMsg)
			_ = log.Error("XML processing failed",
				"error", err,
				"fileType", fileType.name,
				"filePath", filePath)
			continue
		}

		log.Info("XML processing completed",
			"fileType", fileType.name,
			"yearMonth", req.YearMonth,
			"totalRecords", result.TotalRecords,
			"processedRecords", result.ProcessedRecords,
			"insertedRecords", result.InsertedRecords,
			"updatedRecords", result.UpdatedRecords,
			"erroredRecords", result.ErroredRecords)

		// Accumulate totals
		totalProcessed += result.ProcessedRecords
		totalInserted += result.InsertedRecords
		totalUpdated += result.UpdatedRecords
		totalErrors += result.ErroredRecords
		errors = append(errors, result.Errors...)
	}

	success := len(errors) == 0
	message := "Workflow completed successfully"
	if !success {
		message = fmt.Sprintf("Workflow completed with %d errors", len(errors))
	}

	response := &ProcessWorkflowResponse{
		Success:          success,
		Message:          message,
		ProcessingID:     processing.ID.String(),
		TotalRecords:     totalProcessed + totalErrors, // Approximation
		ProcessedRecords: totalProcessed,
		InsertedRecords:  totalInserted,
		UpdatedRecords:   totalUpdated,
		ErroredRecords:   totalErrors,
		Errors:           errors,
	}

	log.Info("Full processing workflow completed",
		"yearMonth", req.YearMonth,
		"success", success,
		"totalProcessed", totalProcessed,
		"totalInserted", totalInserted,
		"totalUpdated", totalUpdated,
		"totalErrors", totalErrors)

	return response, nil
}

// SimplifiedProcessWithTracks performs the full processing workflow with track support for releases
func (ac *AdminController) SimplifiedProcessWithTracks(ctx context.Context, req SimplifiedRequest) (*SimplifiedProcessResponse, error) {
	log := ac.log.Function("SimplifiedProcessWithTracks")

	// Validate file types
	if err := ac.validateFileTypes(req.FileTypes); err != nil {
		return nil, err
	}

	yearMonth := ac.getCurrentYearMonth()

	log.Info("Starting simplified full processing workflow with tracks",
		"yearMonth", yearMonth,
		"fileTypes", req.FileTypes,
		"limits", req.Limits)

	// Find or create the processing record for current year-month
	processing, err := ac.discogsDataProcessingRepo.GetByYearMonth(ctx, yearMonth)
	if err != nil {
		return &SimplifiedProcessResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to find processing record for %s: %v", yearMonth, err),
			YearMonth: yearMonth,
		}, log.Err("failed to find processing record", err, "yearMonth", yearMonth)
	}

	var fileResults []FileResult
	var summary struct {
		TotalFiles       int
		SuccessfulFiles  int
		TotalRecords     int
		ProcessedRecords int
		InsertedRecords  int
		UpdatedRecords   int
		ErroredRecords   int
	}

	summary.TotalFiles = len(req.FileTypes)

	// Map file types to their processing methods (enhanced for releases)
	processingMethods := map[string]func(context.Context, string, string, *services.ProcessingLimits) (*services.ProcessingResult, error){
		"labels":   ac.xmlProcessing.ProcessLabelsFileWithLimits,
		"artists":  ac.xmlProcessing.ProcessArtistsFileWithLimits,
		"masters":  ac.xmlProcessing.ProcessMastersFileWithLimits,
		"releases": ac.xmlProcessing.ProcessReleasesFileWithTracksAndLimits, // Enhanced with track support
	}

	// Process each requested file type
	for _, fileType := range req.FileTypes {
		filePath := ac.buildFilePath(fileType)
		method, exists := processingMethods[fileType]
		if !exists {
			continue // This should not happen due to validation, but safety first
		}

		log.Info("Starting XML processing with track support",
			"fileType", fileType,
			"yearMonth", yearMonth,
			"filePath", filePath,
			"tracksEnabled", fileType == "releases")

		fileResult := FileResult{
			FileType: fileType,
			FilePath: filePath,
		}

		// Convert limits to service format
		var serviceLimits *services.ProcessingLimits
		if req.Limits != nil {
			serviceLimits = &services.ProcessingLimits{
				MaxRecords:   req.Limits.MaxRecords,
				MaxBatchSize: req.Limits.MaxBatchSize,
			}
		}

		// Process the XML file with limits
		result, err := method(ctx, filePath, processing.ID.String(), serviceLimits)
		if err != nil {
			fileResult.Success = false
			fileResult.Errors = []string{fmt.Sprintf("Failed to process file: %v", err)}
			_ = log.Error("XML processing failed",
				"error", err,
				"fileType", fileType,
				"filePath", filePath)
		} else {
			fileResult.Success = true
			fileResult.TotalRecords = result.TotalRecords
			fileResult.ProcessedRecords = result.ProcessedRecords
			fileResult.ErroredRecords = result.ErroredRecords
			fileResult.Errors = result.Errors

			summary.SuccessfulFiles++
			summary.TotalRecords += result.TotalRecords
			summary.ProcessedRecords += result.ProcessedRecords
			summary.InsertedRecords += result.InsertedRecords
			summary.UpdatedRecords += result.UpdatedRecords
			summary.ErroredRecords += result.ErroredRecords

			log.Info("XML processing completed with track support",
				"fileType", fileType,
				"yearMonth", yearMonth,
				"totalRecords", result.TotalRecords,
				"processedRecords", result.ProcessedRecords,
				"insertedRecords", result.InsertedRecords,
				"updatedRecords", result.UpdatedRecords,
				"erroredRecords", result.ErroredRecords,
				"tracksEnabled", fileType == "releases")
		}

		fileResults = append(fileResults, fileResult)
	}

	success := summary.SuccessfulFiles == summary.TotalFiles
	message := fmt.Sprintf("Processed %d/%d files successfully with track support", summary.SuccessfulFiles, summary.TotalFiles)

	response := &SimplifiedProcessResponse{
		Success:      success,
		Message:      message,
		YearMonth:    yearMonth,
		ProcessingID: processing.ID.String(),
		FileResults:  fileResults,
	}
	response.Summary.TotalFiles = summary.TotalFiles
	response.Summary.SuccessfulFiles = summary.SuccessfulFiles
	response.Summary.TotalRecords = summary.TotalRecords
	response.Summary.ProcessedRecords = summary.ProcessedRecords
	response.Summary.InsertedRecords = summary.InsertedRecords
	response.Summary.UpdatedRecords = summary.UpdatedRecords
	response.Summary.ErroredRecords = summary.ErroredRecords

	log.Info("Simplified full processing workflow with tracks completed",
		"yearMonth", yearMonth,
		"processingID", processing.ID.String(),
		"totalFiles", summary.TotalFiles,
		"successfulFiles", summary.SuccessfulFiles,
		"totalRecords", summary.TotalRecords,
		"processedRecords", summary.ProcessedRecords,
		"insertedRecords", summary.InsertedRecords,
		"updatedRecords", summary.UpdatedRecords,
		"erroredRecords", summary.ErroredRecords)

	return response, nil
}