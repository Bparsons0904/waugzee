package types

type DownloadProgressEvent struct {
	YearMonth    string  `json:"yearMonth"`
	Status       string  `json:"status"`
	FileType     string  `json:"fileType"`
	Stage        string  `json:"stage"`
	Downloaded   int64   `json:"downloaded"`
	Total        int64   `json:"total"`
	Percentage   float64 `json:"percentage"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

type ProcessingProgressEvent struct {
	YearMonth      string  `json:"yearMonth"`
	Status         string  `json:"status"`
	FileType       string  `json:"fileType"`
	Step           string  `json:"step"`
	Stage          string  `json:"stage"`
	FilesProcessed int64   `json:"filesProcessed"`
	TotalFiles     int64   `json:"totalFiles"`
	Percentage     float64 `json:"percentage"`
	ErrorMessage   *string `json:"errorMessage,omitempty"`
}
