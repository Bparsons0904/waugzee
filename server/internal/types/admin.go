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
