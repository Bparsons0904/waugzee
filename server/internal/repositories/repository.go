package repositories

import (
	"waugzee/internal/database"
)

type Repository struct {
	User              UserRepository
	UserConfiguration UserConfigurationRepository
	Artist            ArtistRepository
	Master            MasterRepository
	Release           ReleaseRepository
	Genre             GenreRepository
	Label             LabelRepository
}

func New(db database.DB) Repository {
	return Repository{
		User:              NewUserRepository(db),
		UserConfiguration: NewUserConfigurationRepository(db),
		Artist:            NewArtistRepository(db),
		Master:            NewMasterRepository(db),
		Release:           NewReleaseRepository(db),
		Genre:             NewGenreRepository(db),
		Label:             NewLabelRepository(db),
	}
}
