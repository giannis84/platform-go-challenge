package database

import (
	"context"
	"sync"

	"github.com/giannis84/platform-go-challenge/internal/models"
)

// MockRepository is a simple in-memory FavouritesRepository intended for unit tests only.
type MockRepository struct {
	mu         sync.RWMutex
	favourites map[string]map[string]*models.FavouriteAsset
}

// NewMockRepository returns a MockRepository for testing.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		favourites: make(map[string]map[string]*models.FavouriteAsset),
	}
}

func (r *MockRepository) GetUserFavouritesFromDB(userID string) ([]*models.FavouriteAsset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userFavourites, exists := r.favourites[userID]
	if !exists {
		return []*models.FavouriteAsset{}, nil
	}

	result := make([]*models.FavouriteAsset, 0, len(userFavourites))
	for _, fav := range userFavourites {
		result = append(result, fav)
	}
	return result, nil
}

func (r *MockRepository) GetFavouriteFromDB(userID, assetID string) (*models.FavouriteAsset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userFavourites, exists := r.favourites[userID]
	if !exists {
		return nil, ErrNotFound
	}

	favourite, exists := userFavourites[assetID]
	if !exists {
		return nil, ErrNotFound
	}

	return favourite, nil
}

func (r *MockRepository) AddFavouriteInDB(_ context.Context, favourite *models.FavouriteAsset) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.favourites[favourite.UserID]; !exists {
		r.favourites[favourite.UserID] = make(map[string]*models.FavouriteAsset)
	}

	if _, exists := r.favourites[favourite.UserID][favourite.ID]; exists {
		return ErrAlreadyExists
	}

	r.favourites[favourite.UserID][favourite.ID] = favourite
	return nil
}

func (r *MockRepository) UpdateFavouriteInDB(favourite *models.FavouriteAsset) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	userFavourites, exists := r.favourites[favourite.UserID]
	if !exists {
		return ErrNotFound
	}

	if _, exists := userFavourites[favourite.ID]; !exists {
		return ErrNotFound
	}

	r.favourites[favourite.UserID][favourite.ID] = favourite
	return nil
}

func (r *MockRepository) DeleteFavouriteFromDB(userID, assetID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	userFavourites, exists := r.favourites[userID]
	if !exists {
		return ErrNotFound
	}

	if _, exists := userFavourites[assetID]; !exists {
		return ErrNotFound
	}

	delete(r.favourites[userID], assetID)
	return nil
}
