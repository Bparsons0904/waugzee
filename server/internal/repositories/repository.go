package repositories

import (
	"waugzee/internal/database"
)

type Repository struct {
	User                     UserRepository
	UserConfiguration        UserConfigurationRepository
	Artist                   ArtistRepository
	Master                   MasterRepository
	Release                  ReleaseRepository
	Genre                    GenreRepository
	Label                    LabelRepository
	Folder                   FolderRepository
	UserRelease              UserReleaseRepository
	DiscogsDataProcessing    DiscogsDataProcessingRepository
}

func New(db database.DB) Repository {
	return Repository{
		User:                  NewUserRepository(db), // User repo needs cache for caching
		UserConfiguration:     NewUserConfigurationRepository(),
		Artist:               NewArtistRepository(),
		Master:               NewMasterRepository(),
		Release:              NewReleaseRepository(),
		Genre:                NewGenreRepository(),
		Label:                NewLabelRepository(),
		Folder:               NewFolderRepository(),
		UserRelease:          NewUserReleaseRepository(),
		DiscogsDataProcessing: NewDiscogsDataProcessingRepository(db),
	}
}
