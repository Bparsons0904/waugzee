package models

type Artist struct {
	BaseDiscogModel
	Name        string `gorm:"type:text" json:"name"`
	Profile     string `gorm:"type:text" json:"profile"`
	Uri         string `gorm:"type:text" json:"uri"`
	ReleasesURL string `gorm:"type:text" json:"releasesUrl,omitempty"`
	ResourceURL string `gorm:"type:text" json:"resourceUrl,omitempty"`

	// Many-to-many relationship for band members
	Members []*Artist `gorm:"many2many:artist_members;joinForeignKey:artist_id;joinReferences:member_id" json:"members,omitempty"`

	// Relationships
	Releases []Release `gorm:"many2many:release_artists;" json:"releases,omitempty"`
}

// Discogs API response for reference - https://www.discogs.com/developers?gad_source=1&gad_campaignid=823995355&gbraid=0AAAAADmy1_qz72zU5htXZz3lK6Y3ullFL&gclid=CjwKCAjwobnGBhBNEiwAu2mpFHcf8cHDn0K2FJBEyUUOfD427IsCWbYaAxZJC0XueuPQU7VwnLvGtBoCGIsQAvD_BwE#page:database,header:database-master-release
// NOTE: Name doesn't appear in the docs but does appear in the response, added for reference
// {
//   "name": "Nickelback",
//   "namevariations": [
//     "Nickleback"
//   ],
//   "profile": "Nickelback is a Canadian rock band from Hanna, Alberta formed in 1995. Nickelback's music is classed as hard rock and alternative metal. Nickelback is one of the most commercially successful Canadian groups, having sold almost 50 million albums worldwide, ranking as the 11th best selling music act of the 2000s, and is the 2nd best selling foreign act in the U.S. behind The Beatles for the 2000's.",
//   "releases_url": "https://api.discogs.com/artists/108713/releases",
//   "resource_url": "https://api.discogs.com/artists/108713",
//   "uri": "https://www.discogs.com/artist/108713-Nickelback",
//   "urls": [
//     "http://www.nickelback.com/",
//     "http://en.wikipedia.org/wiki/Nickelback"
//   ],
//   "data_quality": "Needs Vote",
//   "id": 108713,
//   "images": [
//     {
//       "height": 260,
//       "resource_url": "https://api-img.discogs.com/9xJ5T7IBn23DDMpg1USsDJ7IGm4=/330x260/smart/filters:strip_icc():format(jpeg):mode_rgb():quality(96)/discogs-images/A-108713-1110576087.jpg.jpg",
//       "type": "primary",
//       "uri": "https://api-img.discogs.com/9xJ5T7IBn23DDMpg1USsDJ7IGm4=/330x260/smart/filters:strip_icc():format(jpeg):mode_rgb():quality(96)/discogs-images/A-108713-1110576087.jpg.jpg",
//       "uri150": "https://api-img.discogs.com/--xqi8cBtaBZz3qOjVcvzGvNRmU=/150x150/smart/filters:strip_icc():format(jpeg):mode_rgb()/discogs-images/A-108713-1110576087.jpg.jpg",
//       "width": 330
//     },
//     {
//       "height": 500,
//       "resource_url": "https://api-img.discogs.com/r1jRG8b9-nlqTHPlJ-t8JR5ugoA=/493x500/smart/filters:strip_icc():format(jpeg):mode_rgb():quality(96)/discogs-images/A-108713-1264273865.jpeg.jpg",
//       "type": "secondary",
//       "uri": "https://api-img.discogs.com/r1jRG8b9-nlqTHPlJ-t8JR5ugoA=/493x500/smart/filters:strip_icc():format(jpeg):mode_rgb():quality(96)/discogs-images/A-108713-1264273865.jpeg.jpg",
//       "uri150": "https://api-img.discogs.com/6K-cI5xDgsurmc-2OX6uCygzDgw=/150x150/smart/filters:strip_icc():format(jpeg):mode_rgb()/discogs-images/A-108713-1264273865.jpeg.jpg",
//       "width": 493
//     }
//   ],
//   "members": [
//     {
//       "active": true,
//       "id": 270222,
//       "name": "Chad Kroeger",
//       "resource_url": "https://api.discogs.com/artists/270222"
//     },
//     {
//       "active": true,
//       "id": 685755,
//       "name": "Daniel Adair",
//       "resource_url": "https://api.discogs.com/artists/685755"
//     },
//     {
//       "active": true,
//       "id": 685754,
//       "name": "Mike Kroeger",
//       "resource_url": "https://api.discogs.com/artists/685754"
//     },
//     {
//       "active": true,
//       "id": 685756,
//       "name": "Ryan \"Vik\" Vikedal",
//       "resource_url": "https://api.discogs.com/artists/685756"
//     },
//     {
//       "active": true,
//       "id": 685757,
//       "name": "Ryan Peake",
//       "resource_url": "https://api.discogs.com/artists/685757"
//     }
//   ],
// }
