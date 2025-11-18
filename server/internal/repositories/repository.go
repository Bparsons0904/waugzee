package repositories

import (
	"waugzee/internal/database"
)

type Repository struct {
	User                  UserRepository
	UserConfiguration     UserConfigurationRepository
	Artist                ArtistRepository
	Master                MasterRepository
	Release               ReleaseRepository
	Genre                 GenreRepository
	Label                 LabelRepository
	Folder                FolderRepository
	UserRelease           UserReleaseRepository
	DiscogsDataProcessing DiscogsDataProcessingRepository
	Stylus                StylusRepository
	History               HistoryRepository
	DailyRecommendation   DailyRecommendationRepository
}

func New(db database.DB) Repository {
	return Repository{
		User:                  NewUserRepository(db),
		UserConfiguration:     NewUserConfigurationRepository(),
		Artist:                NewArtistRepository(),
		Master:                NewMasterRepository(),
		Release:               NewReleaseRepository(),
		Genre:                 NewGenreRepository(),
		Label:                 NewLabelRepository(),
		Folder:                NewFolderRepository(db),
		UserRelease:           NewUserReleaseRepository(db.Cache.User),
		DiscogsDataProcessing: NewDiscogsDataProcessingRepository(db.SQL),
		Stylus:                NewStylusRepository(db.Cache.User),
		History:               NewHistoryRepository(db.Cache.User),
		DailyRecommendation:   NewDailyRecommendationRepository(db.Cache.User),
	}
}
