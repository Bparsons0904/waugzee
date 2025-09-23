package models

import (
	"time"
	"waugzee/internal/utils"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Master struct {
	// Claude change to ID
	ID int64     `gorm:"type:bigint;primaryKey;not null"                          json:"discogsId"             validate:"required,gt=0"`
	CreatedAt time.Time `gorm:"autoCreateTime"                                           json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"                                           json:"updatedAt"`
	Title     string    `gorm:"type:text;not null;index:idx_masters_title"               json:"title"                 validate:"required"`
	// Claude Change to MainReleaseID
	MainReleaseID *int64 `gorm:"type:bigint"                                               json:"mainRelease,omitempty"`
	// Claude add MainReleaseResourceURL
	MainReleaseResourceURL *string `gorm:"type:text" json:"mainReleaseResourceUrl,omitempty"`
	// Claude add Most Recent ReleaseID
	MostRecentReleaseID *int64 `gorm:"type:bigint" json:"mostRecentReleaseId,omitempty"`
	// Claude add MostRecentReleaseResourceURL
	MostRecentReleaseResourceURL *string `gorm:"type:text" json:"mostRecentReleaseResourceUrl,omitempty"`
	Year        *int   `gorm:"type:int;index:idx_masters_year"                          json:"year,omitempty"`
	ContentHash string `gorm:"type:varchar(64);not null;index:idx_masters_content_hash" json:"contentHash"`

	// Claude add Data - Images, Videos
	Data datatypes.JSON `gorm:"type:jsonb" json:"data,omitempty"`

	// Relationships
	Releases []Release `gorm:"foreignKey:MasterID"       json:"releases,omitempty"`
	Genres   []Genre   `gorm:"many2many:master_genres;"  json:"genres,omitempty"`
	Artists  []Artist  `gorm:"many2many:master_artists;" json:"artists,omitempty"`
}

// Example response from Discogs for a master
// {
//   "id": 96559,
//   "main_release": 249504,
//   "most_recent_release": 3341754,
//   "resource_url": "https://api.discogs.com/masters/96559",
//   "uri": "https://www.discogs.com/master/96559-Rick-Astley-Never-Gonna-Give-You-Up",
//   "versions_url": "https://api.discogs.com/masters/96559/versions",
//   "main_release_url": "https://api.discogs.com/releases/249504",
//   "most_recent_release_url": "https://api.discogs.com/releases/3341754",
//   "num_for_sale": 776,
//   "lowest_price": 0.12,
//   "images": [
//     {
//       "type": "primary",
//       "uri": "",
//       "resource_url": "",
//       "uri150": "",
//       "width": 600,
//       "height": 600
//     },
//     {
//       "type": "secondary",
//       "uri": "",
//       "resource_url": "",
//       "uri150": "",
//       "width": 600,
//       "height": 600
//     },
//     {
//       "type": "secondary",
//       "uri": "",
//       "resource_url": "",
//       "uri150": "",
//       "width": 600,
//       "height": 600
//     },
//     {
//       "type": "secondary",
//       "uri": "",
//       "resource_url": "",
//       "uri150": "",
//       "width": 600,
//       "height": 600
//     }
//   ],
//   "genres": [
//     "Electronic",
//     "Pop"
//   ],
//   "styles": [
//     "Euro-Disco"
//   ],
//   "year": 1987,
//   "tracklist": [
//     {
//       "position": "A",
//       "type_": "track",
//       "title": "Never Gonna Give You Up",
//       "duration": "3:32"
//     },
//     {
//       "position": "B",
//       "type_": "track",
//       "title": "Never Gonna Give You Up (Instrumental)",
//       "duration": "3:30"
//     }
//   ],
//   "artists": [
//     {
//       "name": "Rick Astley",
//       "anv": "",
//       "join": "",
//       "role": "",
//       "tracks": "",
//       "id": 72872,
//       "resource_url": "https://api.discogs.com/artists/72872"
//     }
//   ],
//   "title": "Never Gonna Give You Up",
//   "data_quality": "Correct",
//   "videos": [
//     {
//       "uri": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
//       "title": "Rick Astley - Never Gonna Give You Up (Official Video) (4K Remaster)",
//       "description": "The official video for “Never Gonna Give You Up” by Rick Astley. \n\nNever: The Autobiography 📚 OUT NOW! \nFollow this link to get your copy and listen to Rick’s ‘Never’ playlist ❤️ #RickAstleyNever\nhttps://linktr.ee/rickastleynever\n\n“Never Gonna Give You Up",
//       "duration": 214,
//       "embed": true
//     },
//     {
//       "uri": "https://www.youtube.com/watch?v=HgmaLKpgmRQ",
//       "title": "Rick Astley - Never Gonna Give You Up (Cake Mix) (87)",
//       "description": "Rick Astley - Never Gonna Give You Up (Cake Mix) [08/08/87]\n\nProduced By Stock Aitken Waterman\r\n\r\nMixed By Pete Hammond For PWL\n\nStock Aitken Waterman's playlist.\nhttps://www.youtube.com/playlist?list=PLkazYlmM5uMIkiAJ_1Pl6V7HEhLTH5K_g\n\n#pwlmix\n#pwlremix\n",
//       "duration": 349,
//       "embed": true
//     },
//     {
//       "uri": "https://www.youtube.com/watch?v=NOFlBeINOdQ",
//       "title": "Never Gonna Give You Up (Instrumental) - Rick Astley",
//       "description": "From the maxi single Never Gonna Give You Up (1987)",
//       "duration": 379,
//       "embed": true
//     },
//     {
//       "uri": "https://www.youtube.com/watch?v=YtdfaDsZzrI",
//       "title": "Rick Astley - Never Gonna Give You Up (Escape To New York Mix) (87)",
//       "description": "Rick Astley - Never Gonna Give You Up (Escape To New York Mix) [08/08/87]\r\n\r\nProduced By Stock Aitken Waterman\r\n\r\nMixed By The Extra Beat Boys For PWL\n\nStock Aitken Waterman's playlist.\nhttps://www.youtube.com/playlist?list=PLkazYlmM5uMIkiAJ_1Pl6V7HEhLTH5",
//       "duration": 424,
//       "embed": true
//     },
//     {
//       "uri": "https://www.youtube.com/watch?v=liuHotzX4oE",
//       "title": "Rick Astley - Never Gonna Give You Up (Escape From Newton Mix) (87)",
//       "description": "Rick Astley - Never Gonna Give You Up (Escape From Newton Mix) [08/08/87]\r\n\r\nProduced By Stock Aitken Waterman\r\n\r\nMixed By Pete Hammond For PWL\n\nStock Aitken Waterman's playlist.\nhttps://www.youtube.com/playlist?list=PLkazYlmM5uMIkiAJ_1Pl6V7HEhLTH5K_g\n\n#p",
//       "duration": 386,
//       "embed": true
//     }
//   ]
// }

func (m *Master) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID <= 0 {
		return gorm.ErrInvalidValue
	}
	if m.Title == "" {
		return gorm.ErrInvalidValue
	}

	// Generate content hash
	hash, err := utils.GenerateEntityHash(m)
	if err != nil {
		return err
	}
	m.ContentHash = hash

	return nil
}

func (m *Master) BeforeUpdate(tx *gorm.DB) (err error) {
	if m.Title == "" {
		return gorm.ErrInvalidValue
	}

	// Regenerate content hash
	hash, err := utils.GenerateEntityHash(m)
	if err != nil {
		return err
	}
	m.ContentHash = hash

	return nil
}

// Hashable interface implementation
func (m *Master) GetHashableFields() map[string]any {
	return map[string]any{
		"Title":                          m.Title,
		"MainReleaseID":                  m.MainReleaseID,
		"MainReleaseResourceURL":         m.MainReleaseResourceURL,
		"MostRecentReleaseID":            m.MostRecentReleaseID,
		"MostRecentReleaseResourceURL":   m.MostRecentReleaseResourceURL,
		"Year":                           m.Year,
		"Data":                           m.Data,
	}
}

func (m *Master) SetContentHash(hash string) {
	m.ContentHash = hash
}

func (m *Master) GetContentHash() string {
	return m.ContentHash
}

func (m *Master) GetDiscogsID() int64 {
	return m.ID
}
