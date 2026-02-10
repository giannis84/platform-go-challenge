package handlers

import (
	"context"
	"time"

	"github.com/giannis84/platform-go-challenge/internal/database"
	"github.com/giannis84/platform-go-challenge/internal/logging"
	"github.com/giannis84/platform-go-challenge/internal/models"
)

var Repo *database.PostgresRepository

func GetUserFavourites(userID string) ([]*models.FavouriteAsset, error) {
	return Repo.GetUserFavouritesFromDB(userID)
}

func AddFavourite(ctx context.Context, userID string, asset models.Asset, description string) error {
	if err := validateAsset(asset); err != nil {
		return err
	}

	favourite := &models.FavouriteAsset{
		ID:          asset.GetID(),
		UserID:      userID,
		AssetType:   asset.GetType(),
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Data:        asset,
	}

	logging.Log(ctx).Layer("handler").Op("AddFavourite").User(userID).
		Asset(favourite.ID).AssetType(string(favourite.AssetType)).
		Time("created_at", favourite.CreatedAt).
		Info("creating favourite asset")

	return Repo.AddFavouriteInDB(ctx, favourite)
}

func UpdateDescription(userID, assetID, description string) error {
	if err := validateDescription(description); err != nil {
		return err
	}

	favourite, err := Repo.GetFavouriteFromDB(userID, assetID)
	if err != nil {
		return err
	}

	favourite.Description = description
	favourite.UpdatedAt = time.Now()

	return Repo.UpdateFavouriteInDB(favourite)
}

func RemoveFavourite(userID, assetID string) error {
	return Repo.DeleteFavouriteFromDB(userID, assetID)
}

// validateAsset chooses the correct validation function based on asset type.
func validateAsset(asset models.Asset) error {
	switch a := asset.(type) {
	case *models.Chart:
		return validateChart(a)
	case *models.Insight:
		return validateInsight(a)
	case *models.Audience:
		return validateAudience(a)
	default:
		return nil
	}
}
