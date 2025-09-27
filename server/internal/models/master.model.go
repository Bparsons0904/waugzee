package models

import (
	"time"

	"gorm.io/datatypes"
)

type Master struct {
	BaseDiscogModel
	Title                        string     `gorm:"type:text"        json:"title"`
	LastSynced                   *time.Time `gorm:"type:timestamptz" json:"lastSynced,omitempty"`
	MainReleaseID                *int64     `gorm:"type:bigint"      json:"mainRelease,omitempty"`
	MainReleaseResourceURL       *string    `gorm:"type:text"        json:"mainReleaseResourceUrl,omitempty"`
	MostRecentReleaseID          *int64     `gorm:"type:bigint"      json:"mostRecentReleaseId,omitempty"`
	MostRecentReleaseResourceURL *string    `gorm:"type:text"        json:"mostRecentReleaseResourceUrl,omitempty"`
	Year                         *int       `gorm:"type:int"         json:"year,omitempty"`
	Uri                          string     `gorm:"type:text"        json:"uri"`
	ResourceURL                  string     `gorm:"type:text"        json:"resourceUrl"`

	// Claude add Data - Images, Videos
	Data datatypes.JSON `gorm:"type:jsonb" json:"data,omitempty"`

	// Relationships
	Releases []*Release `gorm:"-:migration"              json:"releases,omitempty"`
	Genres   []Genre    `gorm:"many2many:master_genres;"  json:"genres,omitempty"`
	Artists  []Artist   `gorm:"many2many:master_artists;" json:"artists,omitempty"`
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
