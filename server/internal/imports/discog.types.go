package imports

import (
	"encoding/xml"
	"time"
)

// Core Discogs entities
type Artist struct {
	ID       int           `xml:"id"                  json:"id"              db:"id"`
	Name     string        `xml:"name"                json:"name"            db:"name"`
	RealName string        `xml:"realname"            json:"real_name"       db:"real_name"`
	Profile  string        `xml:"profile"             json:"profile"         db:"profile"`
	URLs     []string      `xml:"urls>url"            json:"urls"            db:"urls"`
	NameVars []string      `xml:"namevariations>name" json:"name_variations" db:"name_variations"`
	Aliases  []Alias       `xml:"aliases>name"        json:"aliases"         db:"aliases"`
	Members  []Member      `xml:"members>name"        json:"members"         db:"members"`
	Groups   []Group       `xml:"groups>name"         json:"groups"          db:"groups"`
	Images   []DiscogsImage `xml:"images>image"        json:"images"          db:"images"`
}

type Label struct {
	ID          int      `xml:"id"              json:"id"           db:"id"`
	Name        string   `xml:"name"            json:"name"         db:"name"`
	ContactInfo string   `xml:"contactinfo"     json:"contact_info" db:"contact_info"`
	Profile     string   `xml:"profile"         json:"profile"      db:"profile"`
	ParentLabel string   `xml:"parentLabel"     json:"parent_label" db:"parent_label"`
	SubLabels   []string `xml:"sublabels>label" json:"sublabels"    db:"sublabels"`
	URLs        []string `xml:"urls>url"        json:"urls"         db:"urls"`
}

// ReleaseLabel represents a label as it appears within a release (with attributes)
type ReleaseLabel struct {
	ID        int    `xml:"id,attr"    json:"id"         db:"id"`
	Name      string `xml:"name,attr"  json:"name"       db:"name"`
	CatalogNo string `xml:"catno,attr" json:"catalog_no" db:"catalog_no"`
}

type Release struct {
	ID          int           `xml:"id,attr"         json:"id"           db:"id"`
	Status      string        `xml:"status,attr"     json:"status"       db:"status"`
	Title       string        `xml:"title"           json:"title"        db:"title"`
	Country     string        `xml:"country"         json:"country"      db:"country"`
	Released    string        `xml:"released"        json:"released"     db:"released"`
	Notes       string        `xml:"notes"           json:"notes"        db:"notes"`
	DataQuality string        `xml:"data_quality"    json:"data_quality" db:"data_quality"`
	MasterID    int           `xml:"master_id"       json:"master_id"    db:"master_id"`
	Artists     []Artist      `xml:"artists>artist"  json:"artists"      db:"artists"`
	Labels      []ReleaseLabel `xml:"labels>label"    json:"labels"       db:"labels"`
	Formats     []Format      `xml:"formats>format"  json:"formats"      db:"formats"`
	Genres      []string      `xml:"genres>genre"    json:"genres"       db:"genres"`
	Styles      []string      `xml:"styles>style"    json:"styles"       db:"styles"`
	Tracklist   []Track       `xml:"tracklist>track" json:"tracklist"    db:"tracklist"`
	Images      []DiscogsImage `xml:"images>image"    json:"images"       db:"images"`
}

// Supporting structs
type Alias struct {
	ID   int    `xml:"id,attr"   json:"id"   db:"id"`
	Name string `xml:",chardata" json:"name" db:"name"`
}

type Member struct {
	ID   int    `xml:"id,attr"   json:"id"   db:"id"`
	Name string `xml:",chardata" json:"name" db:"name"`
}

type Group struct {
	ID   int    `xml:"id,attr"   json:"id"   db:"id"`
	Name string `xml:",chardata" json:"name" db:"name"`
}

type Format struct {
	Name         string   `xml:"name,attr"                json:"name"         db:"name"`
	Qty          string   `xml:"qty,attr"                 json:"qty"          db:"qty"`
	Text         string   `xml:"text,attr"                json:"text"         db:"text"`
	Descriptions []string `xml:"descriptions>description" json:"descriptions" db:"descriptions"`
}

type Track struct {
	Position string `xml:"position" json:"position" db:"position"`
	Title    string `xml:"title"    json:"title"    db:"title"`
	Duration string `xml:"duration" json:"duration" db:"duration"`
}

type DownloadJob struct {
	ID             int        `json:"id"              db:"id"`
	Type           string     `json:"type"            db:"type"` // "artists", "labels", "releases", "masters"
	URL            string     `json:"url"             db:"url"`
	Filename       string     `json:"filename"        db:"filename"`
	Status         string     `json:"status"          db:"status"` // "pending", "downloading", "processing", "completed", "failed"
	StartedAt      time.Time  `json:"started_at"      db:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"    db:"completed_at"`
	FileSize       int64      `json:"file_size"       db:"file_size"`
	ProcessedCount int        `json:"processed_count" db:"processed_count"`
	ErrorMessage   string     `json:"error_message"   db:"error_message"`
}

type ProcessingStats struct {
	TotalRecords     int `json:"total_records"`
	ProcessedRecords int `json:"processed_records"`
	InsertedRecords  int `json:"inserted_records"`
	UpdatedRecords   int `json:"updated_records"`
	ErroredRecords   int `json:"errored_records"`
}

// Root containers for the XML files
type DiscogsArtists struct {
	XMLName xml.Name `xml:"artists"`
	Artists []Artist `xml:"artist"`
}

type DiscogsLabels struct {
	XMLName xml.Name `xml:"labels"`
	Labels  []Label  `xml:"label"`
}

type DiscogsReleases struct {
	XMLName  xml.Name  `xml:"releases"`
	Releases []Release `xml:"release"`
}

type DiscogsMasters struct {
	XMLName xml.Name `xml:"masters"`
	Masters []Master `xml:"master"`
}

type Master struct {
	ID          int           `xml:"id,attr"        json:"id"           db:"id"`
	MainRelease int           `xml:"main_release"   json:"main_release" db:"main_release"`
	Title       string        `xml:"title"          json:"title"        db:"title"`
	Year        int           `xml:"year"           json:"year"         db:"year"`
	Notes       string        `xml:"notes"          json:"notes"        db:"notes"`
	DataQuality string        `xml:"data_quality"   json:"data_quality" db:"data_quality"`
	Artists     []Artist      `xml:"artists>artist" json:"artists"      db:"artists"`
	Genres      []string      `xml:"genres>genre"   json:"genres"       db:"genres"`
	Styles      []string      `xml:"styles>style"   json:"styles"       db:"styles"`
	Videos      []Video       `xml:"videos>video"   json:"videos"       db:"videos"`
	Images      []DiscogsImage `xml:"images>image"   json:"images"       db:"images"`
}

type Video struct {
	Duration    int    `xml:"duration,attr" json:"duration"    db:"duration"`
	Embed       bool   `xml:"embed,attr"    json:"embed"       db:"embed"`
	Source      string `xml:"src,attr"      json:"source"      db:"source"`
	Title       string `xml:"title"         json:"title"       db:"title"`
	Description string `xml:"description"   json:"description" db:"description"`
	URI         string `xml:"uri"           json:"uri"         db:"uri"`
}

type DiscogsImage struct {
	Type   string `xml:"type,attr"   json:"type"    db:"type"`
	Width  int    `xml:"width,attr"  json:"width"   db:"width"`
	Height int    `xml:"height,attr" json:"height"  db:"height"`
	URI    string `xml:"uri,attr"    json:"uri"     db:"uri"`
	URI150 string `xml:"uri150,attr" json:"uri150"  db:"uri150"`
}
