package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/giannis84/platform-go-challenge/internal/database"
	"github.com/giannis84/platform-go-challenge/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

// testLogger returns a logger that discards all output for tests
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// testToken returns an unsigned JWT with the given subject.
func testToken(sub string) string {
	claims := jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	s, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	return s
}

// addAuthHeader adds a Bearer token for the given user to the request.
func addAuthHeader(req *http.Request, userID string) {
	req.Header.Set("Authorization", "Bearer "+testToken(userID))
}

func setupTestHandler() *chi.Mux {
	logger := testLogger()
	repo := database.NewMockRepository()

	router := chi.NewRouter()
	router.Use(logging.RequestLogger(logger)) // Add test logger middleware
	router.Group(RegisterFavouritesRoutes(repo, "")) // empty secret = unsigned tokens

	return router
}

func TestFavouritesRoutes_AddFavourite(t *testing.T) {
	router := setupTestHandler()

	requestBody := map[string]any{
		"asset_type":  "chart",
		"description": "Monthly sales data",
		"asset_data": map[string]any{
			"id":           "chart1",
			"title":        "Sales Chart",
			"x_axis_title": "Month",
			"y_axis_title": "Sales",
			"data": map[string]any{
				"Jan": 1000,
				"Feb": 1500,
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/favourites", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(req, "user1")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, status)
	}

	var response map[string]string
	json.Unmarshal(rr.Body.Bytes(), &response)

	if response["message"] != "Favourite added successfully" {
		t.Errorf("Expected success message, got %s", response["message"])
	}
}

func TestFavouritesRoutes_GetUserFavourites(t *testing.T) {
	router := setupTestHandler()

	// Add a favourite first
	requestBody := map[string]any{
		"asset_type":  "insight",
		"description": "Social media usage insight",
		"asset_data": map[string]any{
			"id":   "insight1",
			"text": "40% of millennials spend more than 3 hours on social media daily",
		},
	}

	body, _ := json.Marshal(requestBody)
	addReq := httptest.NewRequest("POST", "/api/v1/favourites", bytes.NewBuffer(body))
	addReq.Header.Set("Content-Type", "application/json")
	addAuthHeader(addReq, "user1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, addReq)

	// Now get the favourites
	req := httptest.NewRequest("GET", "/api/v1/favourites", nil)
	addAuthHeader(req, "user1")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}

	var favourites []any
	json.Unmarshal(rr.Body.Bytes(), &favourites)

	if len(favourites) != 1 {
		t.Errorf("Expected 1 favourite, got %d", len(favourites))
	}
}

func TestFavouritesRoutes_UpdateDescription(t *testing.T) {
	router := setupTestHandler()

	// Add a favourite first
	requestBody := map[string]any{
		"asset_type":  "audience",
		"description": "Tech-savvy millennials",
		"asset_data": map[string]any{
			"id":                       "audience1",
			"gender":                   []string{"Male", "Female"},
			"birth_country":            []string{"US", "UK"},
			"age_groups":               []string{"25-34"},
			"social_media_hours_daily": "3-5",
			"purchases_last_month":     5,
		},
	}

	body, _ := json.Marshal(requestBody)
	addReq := httptest.NewRequest("POST", "/api/v1/favourites", bytes.NewBuffer(body))
	addReq.Header.Set("Content-Type", "application/json")
	addAuthHeader(addReq, "user1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, addReq)

	// Update description
	updateBody := map[string]string{
		"description": "Updated audience description",
	}
	updateBodyBytes, _ := json.Marshal(updateBody)
	req := httptest.NewRequest("PATCH", "/api/v1/favourites/audience1", bytes.NewBuffer(updateBodyBytes))
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(req, "user1")

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}
}

func TestFavouritesRoutes_RemoveFavourite(t *testing.T) {
	router := setupTestHandler()

	// Add a favourite first
	requestBody := map[string]any{
		"asset_type": "chart",
		"asset_data": map[string]any{
			"id":           "chart1",
			"title":        "Test Chart",
			"x_axis_title": "X",
			"y_axis_title": "Y",
		},
	}

	body, _ := json.Marshal(requestBody)
	addReq := httptest.NewRequest("POST", "/api/v1/favourites", bytes.NewBuffer(body))
	addReq.Header.Set("Content-Type", "application/json")
	addAuthHeader(addReq, "user1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, addReq)

	// Remove the favourite
	req := httptest.NewRequest("DELETE", "/api/v1/favourites/chart1", nil)
	addAuthHeader(req, "user1")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}

	// Verify it's removed
	getReq := httptest.NewRequest("GET", "/api/v1/favourites", nil)
	addAuthHeader(getReq, "user1")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, getReq)

	var favourites []any
	json.Unmarshal(rr.Body.Bytes(), &favourites)

	if len(favourites) != 0 {
		t.Errorf("Expected 0 favourites after deletion, got %d", len(favourites))
	}
}

func TestFavouritesRoutes_AddDuplicateFavourite(t *testing.T) {
	router := setupTestHandler()

	requestBody := map[string]any{
		"asset_type": "chart",
		"asset_data": map[string]any{
			"id":           "chart1",
			"title":        "Test Chart",
			"x_axis_title": "X",
			"y_axis_title": "Y",
		},
	}

	body, _ := json.Marshal(requestBody)

	// Add first time
	req1 := httptest.NewRequest("POST", "/api/v1/favourites", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	addAuthHeader(req1, "user1")
	rr1 := httptest.NewRecorder()
	router.ServeHTTP(rr1, req1)

	// Add second time (duplicate)
	req2 := httptest.NewRequest("POST", "/api/v1/favourites", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	addAuthHeader(req2, "user1")
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	if status := rr2.Code; status != http.StatusConflict {
		t.Errorf("Expected status %d for duplicate, got %d", http.StatusConflict, status)
	}
}
