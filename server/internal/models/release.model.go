package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ReleaseFormat string

const (
	FormatVinyl    ReleaseFormat = "vinyl"
	FormatCD       ReleaseFormat = "cd"
	FormatCassette ReleaseFormat = "cassette"
	FormatDigital  ReleaseFormat = "digital"
	FormatOther    ReleaseFormat = "other"
)

type Data struct {
	Styles []string `json:"styles"`
	Genres []string `json:"genres"`
}

type Release struct {
	ID          int64         `gorm:"type:bigint;primaryKey;not null"                                                     json:"discogsId"             validate:"required,gt=0"`
	CreatedAt   time.Time     `gorm:"autoCreateTime"                                                                      json:"createdAt"`
	UpdatedAt   time.Time     `gorm:"autoUpdateTime"                                                                      json:"updatedAt"`
	Title       string        `gorm:"type:text;not null;index:idx_releases_title"                                         json:"title"                 validate:"required"`
	LabelID     *int64        `gorm:"type:bigint;index:idx_releases_label"                                                json:"labelId,omitempty"`
	MasterID    *int64        `gorm:"type:bigint;index:idx_releases_master;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"masterId,omitempty"`
	Year        *int          `gorm:"type:int;index:idx_releases_year"                                                    json:"year,omitempty"`
	Country     *string       `gorm:"type:text"                                                                           json:"country,omitempty"`
	Format      ReleaseFormat `gorm:"type:text;default:'vinyl';index:idx_releases_format"                                 json:"format"`
	TrackCount  *int          `gorm:"type:int"                                                                            json:"trackCount,omitempty"`
	Notes       *string       `gorm:"type:text"                                                                           json:"notes,omitempty"`
	ResourceURL *string       `gorm:"type:text"                                                                           json:"resourceUrl,omitempty"`
	URI         *string       `gorm:"type:text"                                                                           json:"uri,omitempty"`
	DateAdded   *time.Time    `gorm:"type:timestamptz"                                                                    json:"dateAdded,omitempty"`
	DateChanged *time.Time    `gorm:"type:timestamptz"                                                                    json:"dateChanged,omitempty"`
	LastSynced  *time.Time    `gorm:"type:timestamptz"                                                                    json:"lastSynced,omitempty"`
	Thumb       *string       `gorm:"type:text"                                                                           json:"thumb,omitempty"`
	CoverImage  *string       `gorm:"type:text"                                                                           json:"coverImage,omitempty"`

	// JSONB column containing embedded display data: tracks, styles, images, videos
	// Claude we eventually need to properly define these with a struct
	Data datatypes.JSON `gorm:"type:jsonb" json:"data,omitempty"`

	Master  *Master  `gorm:"foreignKey:MasterID"                                                    json:"master,omitempty"`
	Artists []Artist `gorm:"many2many:release_artists;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"artists,omitempty"`
	Labels  []Label  `gorm:"many2many:release_labels;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"  json:"labels,omitempty"`
	Genres  []Genre  `gorm:"many2many:release_genres;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"  json:"genres,omitempty"`
}

// For Reference per docs release payload https://www.discogs.com/developers?gad_source=1&gad_campaignid=823995355&gbraid=0AAAAADmy1_qz72zU5htXZz3lK6Y3ullFL&gclid=CjwKCAjwobnGBhBNEiwAu2mpFHcf8cHDn0K2FJBEyUUOfD427IsCWbYaAxZJC0XueuPQU7VwnLvGtBoCGIsQAvD_BwE#page:database,header:database-release
// {
//     "title": "Never Gonna Give You Up",
//     "id": 249504,
//     "artists": [
//         {
//             "anv": "",
//             "id": 72872,
//             "join": "",
//             "name": "Rick Astley",
//             "resource_url": "https://api.discogs.com/artists/72872",
//             "role": "",
//             "tracks": ""
//         }
//     ],
//     "data_quality": "Correct",
//     "thumb": "https://api-img.discogs.com/kAXVhuZuh_uat5NNr50zMjN7lho=/fit-in/300x300/filters:strip_icc():format(jpeg):mode_rgb()/discogs-images/R-249504-1334592212.jpeg.jpg",
//     "community": {
//         "contributors": [
//             {
//                 "resource_url": "https://api.discogs.com/users/memory",
//                 "username": "memory"
//             },
//             {
//                 "resource_url": "https://api.discogs.com/users/_80_",
//                 "username": "_80_"
//             }
//         ],
//         "data_quality": "Correct",
//         "have": 252,
//         "rating": {
//             "average": 3.42,
//             "count": 45
//         },
//         "status": "Accepted",
//         "submitter": {
//             "resource_url": "https://api.discogs.com/users/memory",
//             "username": "memory"
//         },
//         "want": 42
//     },
//     "companies": [
//         {
//             "catno": "",
//             "entity_type": "13",
//             "entity_type_name": "Phonographic Copyright (p)",
//             "id": 82835,
//             "name": "BMG Records (UK) Ltd.",
//             "resource_url": "https://api.discogs.com/labels/82835"
//         },
//         {
//             "catno": "",
//             "entity_type": "29",
//             "entity_type_name": "Mastered At",
//             "id": 266218,
//             "name": "Utopia Studios",
//             "resource_url": "https://api.discogs.com/labels/266218"
//         }
//     ],
//     "country": "UK",
//     "date_added": "2004-04-30T08:10:05-07:00",
//     "date_changed": "2012-12-03T02:50:12-07:00",
//     "estimated_weight": 60,
//     "extraartists": [
//         {
//             "anv": "Me Co",
//             "id": 547352,
//             "join": "",
//             "name": "Me Company",
//             "resource_url": "https://api.discogs.com/artists/547352",
//             "role": "Design",
//             "tracks": ""
//         },
//         {
//             "anv": "Stock / Aitken / Waterman",
//             "id": 20942,
//             "join": "",
//             "name": "Stock, Aitken & Waterman",
//             "resource_url": "https://api.discogs.com/artists/20942",
//             "role": "Producer, Written-By",
//             "tracks": ""
//         }
//     ],
//     "format_quantity": 1,
//     "formats": [
//         {
//             "descriptions": [
//                 "7\"",
//                 "Single",
//                 "45 RPM"
//             ],
//             "name": "Vinyl",
//             "qty": "1"
//         }
//     ],
//     "genres": [
//         "Electronic",
//         "Pop"
//     ],
//     "identifiers": [
//         {
//             "type": "Barcode",
//             "value": "5012394144777"
//         },
//     ],
//     "images": [
//         {
//             "height": 600,
//             "resource_url": "https://api-img.discogs.com/z_u8yqxvDcwVnR4tX2HLNLaQO2Y=/fit-in/600x600/filters:strip_icc():format(jpeg):mode_rgb():quality(96)/discogs-images/R-249504-1334592212.jpeg.jpg",
//             "type": "primary",
//             "uri": "https://api-img.discogs.com/z_u8yqxvDcwVnR4tX2HLNLaQO2Y=/fit-in/600x600/filters:strip_icc():format(jpeg):mode_rgb():quality(96)/discogs-images/R-249504-1334592212.jpeg.jpg",
//             "uri150": "https://api-img.discogs.com/0ZYgPR4X2HdUKA_jkhPJF4SN5mM=/fit-in/150x150/filters:strip_icc():format(jpeg):mode_rgb()/discogs-images/R-249504-1334592212.jpeg.jpg",
//             "width": 600
//         },
//         {
//             "height": 600,
//             "resource_url": "https://api-img.discogs.com/EnQXaDOs5T6YI9zq-R5I_mT7hSk=/fit-in/600x600/filters:strip_icc():format(jpeg):mode_rgb():quality(96)/discogs-images/R-249504-1334592228.jpeg.jpg",
//             "type": "secondary",
//             "uri": "https://api-img.discogs.com/EnQXaDOs5T6YI9zq-R5I_mT7hSk=/fit-in/600x600/filters:strip_icc():format(jpeg):mode_rgb():quality(96)/discogs-images/R-249504-1334592228.jpeg.jpg",
//             "uri150": "https://api-img.discogs.com/abk0FWgWsRDjU4bkCDwk0gyMKBo=/fit-in/150x150/filters:strip_icc():format(jpeg):mode_rgb()/discogs-images/R-249504-1334592228.jpeg.jpg",
//             "width": 600
//         }
//     ],
//     "labels": [
//         {
//             "catno": "PB 41447",
//             "entity_type": "1",
//             "id": 895,
//             "name": "RCA",
//             "resource_url": "https://api.discogs.com/labels/895"
//         }
//     ],
//     "lowest_price": 0.63,
//     "master_id": 96559,
//     "master_url": "https://api.discogs.com/masters/96559",
//     "notes": "UK Release has a black label with the text \"Manufactured In England\" printed on it.\r\n\r\nSleeve:\r\n\u2117 1987 \u2022 BMG Records (UK) Ltd. \u00a9 1987 \u2022 BMG Records (UK) Ltd.\r\nDistributed in the UK by BMG Records \u2022  Distribu\u00e9 en Europe par BMG/Ariola \u2022 Vertrieb en Europa d\u00fcrch BMG/Ariola.\r\n\r\nCenter labels:\r\n\u2117 1987 Pete Waterman Ltd.\r\nOriginal Sound Recording made by PWL.\r\nBMG Records (UK) Ltd. are the exclusive licensees for the world.\r\n\r\nDurations do not appear on the release.\r\n",
//     "num_for_sale": 58,
//     "released": "1987",
//     "released_formatted": "1987",
//     "resource_url": "https://api.discogs.com/releases/249504",
//     "series": [],
//     "status": "Accepted",
//     "styles": [
//         "Synth-pop"
//     ],
//     "tracklist": [
//         {
//             "duration": "3:32",
//             "position": "A",
//             "title": "Never Gonna Give You Up",
//             "type_": "track"
//         },
//         {
//             "duration": "3:30",
//             "position": "B",
//             "title": "Never Gonna Give You Up (Instrumental)",
//             "type_": "track"
//         }
//     ],
//     "uri": "https://www.discogs.com/Rick-Astley-Never-Gonna-Give-You-Up/release/249504",
//     "videos": [
//         {
//             "description": "Rick Astley - Never Gonna Give You Up (Extended Version)",
//             "duration": 330,
//             "embed": true,
//             "title": "Rick Astley - Never Gonna Give You Up (Extended Version)",
//             "uri": "https://www.youtube.com/watch?v=te2jJncBVG4"
//         },
//     ],
//     "year": 1987
// }

func (r *Release) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID <= 0 {
		return gorm.ErrInvalidValue
	}
	if r.Title == "" {
		return gorm.ErrInvalidValue
	}
	if r.Format == "" {
		r.Format = FormatVinyl
	}

	return nil
}

func (r *Release) BeforeUpdate(tx *gorm.DB) (err error) {
	if r.Title == "" {
		return gorm.ErrInvalidValue
	}

	return nil
}

func (r *Release) GetDiscogsID() int64 {
	return r.ID
}
