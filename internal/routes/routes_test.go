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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/giannis84/platform-go-challenge/internal/database"
	"github.com/giannis84/platform-go-challenge/internal/logging"
	"github.com/giannis84/platform-go-challenge/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
)

var testCols = []string{"id", "user_id", "asset_type", "description", "data", "created_at", "updated_at"}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testToken(sub string) string {
	claims := jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	s, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	return s
}

func addAuthHeader(req *http.Request, userID string) {
	req.Header.Set("Authorization", "Bearer "+testToken(userID))
}

func setupTestHandler(t *testing.T) (*chi.Mux, sqlmock.Sqlmock) {
	t.Helper()
	logger := testLogger()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	database.DB = db

	router := chi.NewRouter()
	router.Use(logging.RequestLogger(logger))
	router.Group(RegisterFavouritesRoutes(""))

	return router, mock
}

func insightRequestBody() map[string]any {
	return map[string]any{
		"asset_type":  "insight",
		"description": "Social media usage insight",
		"asset_data": map[string]any{
			"id":   "insight1",
			"text": "40% of millennials spend more than 3 hours on social media daily",
		},
	}
}

func audienceRequestBody() map[string]any {
	return map[string]any{
		"asset_type":  "audience",
		"description": "Tech-savvy millennials",
		"asset_data": map[string]any{
			"id":                      "audience1",
			"gender":                  []string{"Male", "Female"},
			"birth_country":           []string{"US", "UK"},
			"age_groups":              []string{"25-34"},
			"social_media_hours_daily": "3-5",
			"purchases_last_month":    5,
		},
	}
}

func postFavourite(t *testing.T, router *chi.Mux, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/favourites", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(req, "user1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestFavouritesRoutes_AddFavourite(t *testing.T) {
	router, mock := setupTestHandler(t)

	mock.ExpectExec("INSERT INTO favourites").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rr := postFavourite(t, router, insightRequestBody())

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusCreated, status, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["message"] != "Favourite added successfully" {
		t.Errorf("unexpected response: %v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFavouritesRoutes_GetUserFavourites(t *testing.T) {
	router, mock := setupTestHandler(t)
	now := time.Now()

	// Add favourite
	mock.ExpectExec("INSERT INTO favourites").
		WillReturnResult(sqlmock.NewResult(0, 1))
	rr := postFavourite(t, router, insightRequestBody())
	if rr.Code != http.StatusCreated {
		t.Fatalf("setup failed: status %d, body: %s", rr.Code, rr.Body.String())
	}

	// Get favourites
	insightData, _ := json.Marshal(models.Insight{
		ID:   "insight1",
		Text: "40% of millennials spend more than 3 hours on social media daily",
	})
	mock.ExpectQuery("SELECT .+ FROM favourites WHERE user_id").
		WithArgs("user1").
		WillReturnRows(sqlmock.NewRows(testCols).
			AddRow("insight1", "user1", "insight", "Social media usage insight", insightData, now, now))

	req := httptest.NewRequest("GET", "/api/v1/favourites", nil)
	addAuthHeader(req, "user1")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, status, rr.Body.String())
	}

	var favourites []any
	json.Unmarshal(rr.Body.Bytes(), &favourites)
	if len(favourites) != 1 {
		t.Errorf("expected 1 favourite, got %d", len(favourites))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFavouritesRoutes_UpdateDescription(t *testing.T) {
	router, mock := setupTestHandler(t)
	now := time.Now()

	// Add favourite
	mock.ExpectExec("INSERT INTO favourites").
		WillReturnResult(sqlmock.NewResult(0, 1))
	rr := postFavourite(t, router, audienceRequestBody())
	if rr.Code != http.StatusCreated {
		t.Fatalf("setup failed: status %d, body: %s", rr.Code, rr.Body.String())
	}

	// Update description: handler calls GetFavouriteFromDB then UpdateFavouriteInDB
	audienceData, _ := json.Marshal(models.Audience{
		ID: "audience1", Gender: []string{"Male", "Female"}, BirthCountry: []string{"US", "UK"},
		AgeGroups: []string{"25-34"}, SocialMediaHoursDaily: "3-5", PurchasesLastMonth: 5,
	})
	mock.ExpectQuery("SELECT .+ FROM favourites WHERE user_id").
		WithArgs("user1", "audience1").
		WillReturnRows(sqlmock.NewRows(testCols).
			AddRow("audience1", "user1", "audience", "Tech-savvy millennials", audienceData, now, now))
	mock.ExpectExec("UPDATE favourites").
		WillReturnResult(sqlmock.NewResult(0, 1))

	updateBody, _ := json.Marshal(map[string]string{"description": "Updated description for audience"})
	req := httptest.NewRequest("PATCH", "/api/v1/favourites/audience1", bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(req, "user1")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, status, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["message"] != "Description updated successfully" {
		t.Errorf("unexpected response: %v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFavouritesRoutes_RemoveFavourite(t *testing.T) {
	router, mock := setupTestHandler(t)

	// Add favourite
	mock.ExpectExec("INSERT INTO favourites").
		WillReturnResult(sqlmock.NewResult(0, 1))
	rr := postFavourite(t, router, insightRequestBody())
	if rr.Code != http.StatusCreated {
		t.Fatalf("setup failed: status %d, body: %s", rr.Code, rr.Body.String())
	}

	// Remove favourite
	mock.ExpectExec("DELETE FROM favourites").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest("DELETE", "/api/v1/favourites/insight1", nil)
	addAuthHeader(req, "user1")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, status, rr.Body.String())
	}

	// Verify removed by getting empty list
	mock.ExpectQuery("SELECT .+ FROM favourites WHERE user_id").
		WithArgs("user1").
		WillReturnRows(sqlmock.NewRows(testCols))

	req = httptest.NewRequest("GET", "/api/v1/favourites", nil)
	addAuthHeader(req, "user1")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var favourites []any
	json.Unmarshal(rr.Body.Bytes(), &favourites)
	if len(favourites) != 0 {
		t.Errorf("expected 0 favourites after removal, got %d", len(favourites))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFavouritesRoutes_AddDuplicateFavourite(t *testing.T) {
	router, mock := setupTestHandler(t)

	// First add succeeds
	mock.ExpectExec("INSERT INTO favourites").
		WillReturnResult(sqlmock.NewResult(0, 1))
	rr := postFavourite(t, router, insightRequestBody())
	if rr.Code != http.StatusCreated {
		t.Fatalf("first add failed: status %d, body: %s", rr.Code, rr.Body.String())
	}

	// Second add returns unique violation
	mock.ExpectExec("INSERT INTO favourites").
		WillReturnError(&pq.Error{Code: "23505"})
	rr = postFavourite(t, router, insightRequestBody())

	if status := rr.Code; status != http.StatusConflict {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusConflict, status, rr.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "Favourite already exists" {
		t.Errorf("unexpected error message: %v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
