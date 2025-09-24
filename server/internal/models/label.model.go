package models

import (
	"gorm.io/gorm"
)

type Label struct {
	BaseDiscogModel
	Profile     *string   `gorm:"type:text"                                json:"profile,omitempty"`
	ResourceURL string    `gorm:"type:text"                                json:"resourceUrl,omitempty"`
	URI         string    `gorm:"type:text"                                json:"uri,omitempty"`
	Name        string    `gorm:"type:text;not null;index:idx_labels_name" json:"name"`
	Releases    []Release `gorm:"foreignKey:LabelID"                       json:"releases,omitempty"`
}

// For reference only, from Discogs api docs
// {
//   "profile": "Classic Techno label from Detroit, USA.\r\n[b]Label owner:[/b] [a=Carl Craig].\r\n",
//   "releases_url": "https://api.discogs.com/labels/1/releases",
//   "name": "Planet E",
//   "contact_info": "Planet E Communications\r\nP.O. Box 27218\r\nDetroit, 48227, USA\r\n\r\np: 313.874.8729 \r\nf: 313.874.8732\r\n\r\nemail: info AT Planet-e DOT net\r\n",
//   "uri": "https://www.discogs.com/label/1-Planet-E",
//   "sublabels": [
//     {
//       "resource_url": "https://api.discogs.com/labels/86537",
//       "id": 86537,
//       "name": "Antidote (4)"
//     },
//     {
//       "resource_url": "https://api.discogs.com/labels/41841",
//       "id": 41841,
//       "name": "Community Projects"
//     }
//   ],
//   "urls": [
//     "http://www.planet-e.net",
//     "http://planetecommunications.bandcamp.com",
//     "http://twitter.com/planetedetroit"
//   ],
//   "images": [
//     {
//       "height": 24,
//       "resource_url": "https://api-img.discogs.com/85-gKw4oEXfDp9iHtqtCF5Y_ZgI=/fit-in/132x24/filters:strip_icc():format(jpeg):mode_rgb():quality(96)/discogs-images/L-1-1111053865.png.jpg",
//       "type": "primary",
//       "uri": "https://api-img.discogs.com/85-gKw4oEXfDp9iHtqtCF5Y_ZgI=/fit-in/132x24/filters:strip_icc():format(jpeg):mode_rgb():quality(96)/discogs-images/L-1-1111053865.png.jpg",
//       "uri150": "https://api-img.discogs.com/cYmCut4Yh99FaLFHyoqkFo-Md1E=/fit-in/150x150/filters:strip_icc():format(jpeg):mode_rgb()/discogs-images/L-1-1111053865.png.jpg",
//       "width": 132
//     }
//   ],
//   "resource_url": "https://api.discogs.com/labels/1",
//   "id": 1,
//   "data_quality": "Needs Vote"
// }

func (l *Label) BeforeCreate(tx *gorm.DB) (err error) {
	// if l.ID <= 0 {
	// 	return gorm.ErrInvalidValue
	// }
	// if l.Name == "" {
	// 	return gorm.ErrInvalidValue
	// }
	return nil
}

func (l *Label) BeforeUpdate(tx *gorm.DB) (err error) {
	// if l.Name == "" {
	// 	return gorm.ErrInvalidValue
	// }
	return nil
}
