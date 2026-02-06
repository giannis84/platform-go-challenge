package database

import (
	"context"
	"errors"

	"github.com/giannis84/platform-go-challenge/internal/models"
)

var (
	ErrNotFound      = errors.New("favourite not found")
	ErrAlreadyExists = errors.New("favourite already exists")
)

// FavouritesRepository defines the interface for managing user favourites storage.
type FavouritesRepository interface {
	GetUserFavouritesFromDB(userID string) ([]*models.FavouriteAsset, error)
	GetFavouriteFromDB(userID, assetID string) (*models.FavouriteAsset, error)
	AddFavouriteInDB(ctx context.Context, favourite *models.FavouriteAsset) error
	UpdateFavouriteInDB(favourite *models.FavouriteAsset) error
	DeleteFavouriteFromDB(userID, assetID string) error
}
