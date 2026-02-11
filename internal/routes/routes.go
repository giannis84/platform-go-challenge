package routes

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/giannis84/platform-go-challenge/internal/auth"
	"github.com/giannis84/platform-go-challenge/internal/database"
	"github.com/giannis84/platform-go-challenge/internal/handlers"
	"github.com/giannis84/platform-go-challenge/internal/logging"
	"github.com/go-chi/chi/v5"
)

// RegisterFavouritesRoutes sets up the favourites API routes.
// HTTP concerns are handled here, while business logic is delegated to the handlers package.
func RegisterFavouritesRoutes(jwtSecret string) func(r chi.Router) {
	return func(r chi.Router) {
		r.Route("/api/v1", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(jwtSecret))
			r.Route("/favourites", func(r chi.Router) {
				r.Get("/", getUserFavouritesRoute())
				r.Post("/", addUserFavouriteRoute())
				r.Patch("/{assetID}", updateUserFavouriteRoute())
				r.Delete("/{assetID}", removeUserFavouriteRoute())
			})
		})
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func getUserFavouritesRoute() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := auth.UserIDFromContext(ctx)

		logging.Log(ctx).Layer("routes").Op("getUserFavourites").User(userID).
			Info("received get favourites request")

		favourites, err := handlers.GetUserFavourites(userID)
		if err != nil {
			logging.Log(ctx).Layer("routes").User(userID).Err(err).
				Error("failed to get user favourites")
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logging.Log(ctx).Layer("routes").Op("getUserFavourites").User(userID).
			Int("count", len(favourites)).Int("status_code", http.StatusOK).
			Info("favourites retrieved successfully")
		respondWithJSON(w, http.StatusOK, favourites)
	}
}

func addUserFavouriteRoute() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := auth.UserIDFromContext(ctx)

		var req handlers.AddFavouriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logging.Log(ctx).Layer("routes").Op("addUserFavourite").User(userID).Err(err).
				Error("failed to decode request body")
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		logging.Log(ctx).Layer("routes").Op("addUserFavourite").User(userID).
			AssetType(string(req.AssetType)).Str("asset_data", string(req.AssetData)).
			Info("received add favourite request")

		asset, err := handlers.ParseAddFavouriteRequest(&req)
		if err != nil {
			logging.Log(ctx).Layer("routes").User(userID).Err(err).
				Warn("invalid add favourite request")
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		err = handlers.AddFavourite(ctx, userID, asset, req.Description)
		if err != nil {
			var validationErr *handlers.ValidationError
			if errors.As(err, &validationErr) {
				respondWithError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err == database.ErrAlreadyExists {
				logging.Log(ctx).Layer("routes").User(userID).Asset(asset.GetID()).
					Warn("favourite already exists")
				respondWithError(w, http.StatusConflict, "Favourite already exists")
				return
			}
			logging.Log(ctx).Layer("routes").User(userID).Err(err).Error("failed to add favourite")
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logging.Log(ctx).Layer("routes").Op("addUserFavourite").User(userID).
			Asset(asset.GetID()).AssetType(string(req.AssetType)).Int("status_code", http.StatusCreated).
			Info("favourite added successfully")
		respondWithJSON(w, http.StatusCreated, map[string]string{"message": "Favourite added successfully"})
	}
}

func updateUserFavouriteRoute() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := auth.UserIDFromContext(ctx)
		assetID := chi.URLParam(r, "assetID")

		if err := handlers.ValidateAssetID(assetID); err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		var req handlers.UpdateDescriptionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logging.Log(ctx).Layer("routes").User(userID).Asset(assetID).Err(err).
				Error("failed to decode request body")
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		logging.Log(ctx).Layer("routes").Op("updateUserFavourite").User(userID).Asset(assetID).
			Str("description", req.Description).Info("received update favourite request")

		err := handlers.UpdateDescription(userID, assetID, req.Description)
		if err != nil {
			var validationErr *handlers.ValidationError
			if errors.As(err, &validationErr) {
				logging.Log(ctx).Layer("routes").User(userID).Asset(assetID).Err(err).
					Warn("validation error on update favourite")
				respondWithError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err == database.ErrNotFound {
				logging.Log(ctx).Layer("routes").User(userID).Asset(assetID).
					Warn("favourite not found")
				respondWithError(w, http.StatusNotFound, "Favourite not found")
				return
			}
			logging.Log(ctx).Layer("routes").User(userID).Asset(assetID).Err(err).
				Error("failed to update favourite")
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logging.Log(ctx).Layer("routes").Op("updateUserFavourite").User(userID).Asset(assetID).
			Int("status_code", http.StatusOK).Info("favourite updated successfully")
		respondWithJSON(w, http.StatusOK, map[string]string{"message": "Description updated successfully"})
	}
}

func removeUserFavouriteRoute() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := auth.UserIDFromContext(ctx)
		assetID := chi.URLParam(r, "assetID")

		if err := handlers.ValidateAssetID(assetID); err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		logging.Log(ctx).Layer("routes").Op("removeUserFavourite").User(userID).Asset(assetID).
			Info("received remove favourite request")

		err := handlers.RemoveFavourite(userID, assetID)
		if err != nil {
			if err == database.ErrNotFound {
				logging.Log(ctx).Layer("routes").User(userID).Asset(assetID).
					Warn("favourite not found")
				respondWithError(w, http.StatusNotFound, "Favourite not found")
				return
			}
			logging.Log(ctx).Layer("routes").User(userID).Asset(assetID).Err(err).
				Error("failed to remove favourite")
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		logging.Log(ctx).Layer("routes").Op("removeUserFavourite").User(userID).Asset(assetID).
			Int("status_code", http.StatusOK).Info("favourite removed successfully")
		respondWithJSON(w, http.StatusOK, map[string]string{"message": "Favourite removed successfully"})
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ErrorResponse{Error: message})
}
