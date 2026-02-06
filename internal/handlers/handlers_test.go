package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/giannis84/platform-go-challenge/internal/database"
	"github.com/giannis84/platform-go-challenge/internal/logging"
	"github.com/giannis84/platform-go-challenge/internal/models"
)

// testContext returns a context with a discarding logger for tests.
func testContext() context.Context {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return logging.NewContextWithLogger(context.Background(), logger)
}

// --- Handler tests ---

func TestAddFavourite(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		asset      models.Asset
		addBefore  models.Asset // if set, add this asset first to seed the repo
		wantErr    bool
		wantValErr bool   // expect *ValidationError
		errSubstr  string // substring expected in error message
	}{
		{
			name:   "valid chart",
			userID: "user1",
			asset: &models.Chart{
				ID: "c1", Title: "Revenue", XAxisTitle: "Month", YAxisTitle: "USD",
			},
		},
		{
			name:   "valid insight",
			userID: "user1",
			asset: &models.Insight{
				ID: "i1", Text: "40% of millennials spend 3h on social media",
			},
		},
		{
			name:   "valid audience with all fields",
			userID: "user1",
			asset: &models.Audience{
				ID: "a1", Gender: []string{"Male"}, BirthCountry: []string{"Greece"},
				AgeGroups: []string{"25-34"}, SocialMediaHoursDaily: "3-5", PurchasesLastMonth: 5,
			},
		},
		{
			name:   "valid audience with only required fields",
			userID: "user1",
			asset:  &models.Audience{ID: "a2"},
		},
		{
			name:       "chart missing title returns validation error",
			userID:     "user1",
			asset:      &models.Chart{ID: "c1", XAxisTitle: "X", YAxisTitle: "Y"},
			wantErr:    true,
			wantValErr: true,
			errSubstr:  "title is required",
		},
		{
			name:       "insight missing text returns validation error",
			userID:     "user1",
			asset:      &models.Insight{ID: "i1"},
			wantErr:    true,
			wantValErr: true,
			errSubstr:  "text is required",
		},
		{
			name:       "audience with invalid gender returns validation error",
			userID:     "user1",
			asset:      &models.Audience{ID: "a1", Gender: []string{"Other"}},
			wantErr:    true,
			wantValErr: true,
			errSubstr:  "gender[0] has invalid value",
		},
		{
			name:       "audience with negative purchases returns validation error",
			userID:     "user1",
			asset:      &models.Audience{ID: "a1", PurchasesLastMonth: -1},
			wantErr:    true,
			wantValErr: true,
			errSubstr:  "purchases_last_month must not be negative",
		},
		{
			name:   "duplicate favourite returns already exists error",
			userID: "user1",
			asset: &models.Chart{
				ID: "dup1", Title: "T", XAxisTitle: "X", YAxisTitle: "Y",
			},
			addBefore: &models.Chart{
				ID: "dup1", Title: "T", XAxisTitle: "X", YAxisTitle: "Y",
			},
			wantErr:   true,
			errSubstr: "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := database.NewMockRepository()
			ctx := testContext()

			if tt.addBefore != nil {
				if err := AddFavourite(ctx, repo, tt.userID, tt.addBefore, ""); err != nil {
					t.Fatalf("seed setup failed: %v", err)
				}
			}

			err := AddFavourite(ctx, repo, tt.userID, tt.asset, "")

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tt.wantErr {
				if tt.wantValErr {
					var valErr *ValidationError
					if !errors.As(err, &valErr) {
						t.Fatalf("expected *ValidationError, got %T: %v", err, err)
					}
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error to contain %q, got: %v", tt.errSubstr, err)
				}
			}

			// verify asset was stored on success
			if !tt.wantErr {
				favs, _ := repo.GetUserFavouritesFromDB(tt.userID)
				found := false
				for _, f := range favs {
					if f.ID == tt.asset.GetID() {
						found = true
					}
				}
				if !found {
					t.Error("expected favourite to be stored in repository")
				}
			}
		})
	}
}

func TestUpdateDescription(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		assetID     string
		description string
		seedAsset   bool // whether to pre-add a favourite with this assetID
		wantErr     bool
		wantValErr  bool
		errSubstr   string
	}{
		{
			name:        "valid update",
			userID:      "user1",
			assetID:     "c1",
			description: "Updated description",
			seedAsset:   true,
		},
		{
			name:        "empty description returns validation error",
			userID:      "user1",
			assetID:     "c1",
			description: "",
			seedAsset:   true,
			wantErr:     true,
			wantValErr:  true,
			errSubstr:   "description is required",
		},
		{
			name:        "whitespace-only description returns validation error",
			userID:      "user1",
			assetID:     "c1",
			description: "   ",
			seedAsset:   true,
			wantErr:     true,
			wantValErr:  true,
			errSubstr:   "description is required",
		},
		{
			name:        "description too long returns validation error",
			userID:      "user1",
			assetID:     "c1",
			description: strings.Repeat("d", 256),
			seedAsset:   true,
			wantErr:     true,
			wantValErr:  true,
			errSubstr:   "description exceeds maximum length",
		},
		{
			name:        "non-existent favourite returns not found",
			userID:      "user1",
			assetID:     "nonexistent",
			description: "Some description",
			seedAsset:   false,
			wantErr:     true,
			errSubstr:   "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := database.NewMockRepository()
			ctx := testContext()

			if tt.seedAsset {
				chart := &models.Chart{ID: tt.assetID, Title: "T", XAxisTitle: "X", YAxisTitle: "Y"}
				if err := AddFavourite(ctx, repo, tt.userID, chart, ""); err != nil {
					t.Fatalf("seed setup failed: %v", err)
				}
			}

			err := UpdateDescription(repo, tt.userID, tt.assetID, tt.description)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tt.wantErr {
				if tt.wantValErr {
					var valErr *ValidationError
					if !errors.As(err, &valErr) {
						t.Fatalf("expected *ValidationError, got %T: %v", err, err)
					}
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error to contain %q, got: %v", tt.errSubstr, err)
				}
			}

			// verify description was updated on success
			if !tt.wantErr {
				fav, _ := repo.GetFavouriteFromDB(tt.userID, tt.assetID)
				if fav.Description != tt.description {
					t.Errorf("expected description %q, got %q", tt.description, fav.Description)
				}
			}
		})
	}
}

func TestRemoveFavourite(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		assetID   string
		seedAsset bool
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "remove existing favourite",
			userID:    "user1",
			assetID:   "c1",
			seedAsset: true,
		},
		{
			name:      "remove non-existent favourite returns not found",
			userID:    "user1",
			assetID:   "nonexistent",
			seedAsset: false,
			wantErr:   true,
			errSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := database.NewMockRepository()
			ctx := testContext()

			if tt.seedAsset {
				chart := &models.Chart{ID: tt.assetID, Title: "T", XAxisTitle: "X", YAxisTitle: "Y"}
				if err := AddFavourite(ctx, repo, tt.userID, chart, ""); err != nil {
					t.Fatalf("seed setup failed: %v", err)
				}
			}

			err := RemoveFavourite(repo, tt.userID, tt.assetID)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("expected error to contain %q, got: %v", tt.errSubstr, err)
			}

			// verify asset was removed on success
			if !tt.wantErr {
				favs, _ := repo.GetUserFavouritesFromDB(tt.userID)
				for _, f := range favs {
					if f.ID == tt.assetID {
						t.Error("expected favourite to be removed from repository")
					}
				}
			}
		})
	}
}

func TestGetUserFavourites(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		seedCount int // number of assets to pre-add
		wantCount int
	}{
		{
			name:      "returns favourites for user",
			userID:    "user1",
			seedCount: 2,
			wantCount: 2,
		},
		{
			name:      "returns empty list for unknown user",
			userID:    "unknown",
			seedCount: 0,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := database.NewMockRepository()
			ctx := testContext()

			for i := range tt.seedCount {
				chart := &models.Chart{
					ID: string(rune('a' + i)), Title: "T", XAxisTitle: "X", YAxisTitle: "Y",
				}
				if err := AddFavourite(ctx, repo, tt.userID, chart, ""); err != nil {
					t.Fatalf("seed setup failed: %v", err)
				}
			}

			favourites, err := GetUserFavourites(repo, tt.userID)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if len(favourites) != tt.wantCount {
				t.Errorf("expected %d favourites, got %d", tt.wantCount, len(favourites))
			}
		})
	}
}

// --- Validation tests ---

func TestValidateChart(t *testing.T) {
	tests := []struct {
		name      string
		chart     models.Chart
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid chart",
			chart:   models.Chart{ID: "c1", Title: "Revenue", XAxisTitle: "Month", YAxisTitle: "USD"},
			wantErr: false,
		},
		{
			name:      "missing id",
			chart:     models.Chart{Title: "Revenue", XAxisTitle: "Month", YAxisTitle: "USD"},
			wantErr:   true,
			errSubstr: "id is required",
		},
		{
			name:      "missing title",
			chart:     models.Chart{ID: "c1", XAxisTitle: "Month", YAxisTitle: "USD"},
			wantErr:   true,
			errSubstr: "title is required",
		},
		{
			name:      "missing x_axis_title",
			chart:     models.Chart{ID: "c1", Title: "Revenue", YAxisTitle: "USD"},
			wantErr:   true,
			errSubstr: "x_axis_title is required",
		},
		{
			name:      "missing y_axis_title",
			chart:     models.Chart{ID: "c1", Title: "Revenue", XAxisTitle: "Month"},
			wantErr:   true,
			errSubstr: "y_axis_title is required",
		},
		{
			name: "title too long",
			chart: models.Chart{
				ID: "c1", Title: strings.Repeat("a", 256), XAxisTitle: "Month", YAxisTitle: "USD",
			},
			wantErr:   true,
			errSubstr: "title exceeds maximum length",
		},
		{
			name:      "all required fields missing reports multiple errors",
			chart:     models.Chart{},
			wantErr:   true,
			errSubstr: "id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChart(&tt.chart)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("expected error to contain %q, got: %v", tt.errSubstr, err)
			}
		})
	}
}

func TestValidateInsight(t *testing.T) {
	tests := []struct {
		name      string
		insight   models.Insight
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid insight",
			insight: models.Insight{ID: "i1", Text: "40% of millennials spend 3h on social media"},
			wantErr: false,
		},
		{
			name:      "missing id",
			insight:   models.Insight{Text: "some text"},
			wantErr:   true,
			errSubstr: "id is required",
		},
		{
			name:      "missing text",
			insight:   models.Insight{ID: "i1"},
			wantErr:   true,
			errSubstr: "text is required",
		},
		{
			name:      "text too long",
			insight:   models.Insight{ID: "i1", Text: strings.Repeat("x", 256)},
			wantErr:   true,
			errSubstr: "text exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInsight(&tt.insight)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("expected error to contain %q, got: %v", tt.errSubstr, err)
			}
		})
	}
}

func TestValidateAudience(t *testing.T) {
	tests := []struct {
		name      string
		audience  models.Audience
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid audience with all fields",
			audience: models.Audience{
				ID: "a1", Gender: []string{"Male"}, BirthCountry: []string{"Greece"},
				AgeGroups: []string{"25-34"}, SocialMediaHoursDaily: "3-5", PurchasesLastMonth: 5,
			},
			wantErr: false,
		},
		{
			name:     "valid audience with only required fields",
			audience: models.Audience{ID: "a2"},
			wantErr:  false,
		},
		{
			name:     "valid audience with partial optional fields",
			audience: models.Audience{ID: "a3", AgeGroups: []string{"18-24", "25-34"}},
			wantErr:  false,
		},
		{
			name:      "missing id",
			audience:  models.Audience{Gender: []string{"Male"}},
			wantErr:   true,
			errSubstr: "id is required",
		},
		{
			name:      "invalid gender value",
			audience:  models.Audience{ID: "a1", Gender: []string{"Other"}},
			wantErr:   true,
			errSubstr: "gender[0] has invalid value",
		},
		{
			name:      "invalid age group value",
			audience:  models.Audience{ID: "a1", AgeGroups: []string{"10-17"}},
			wantErr:   true,
			errSubstr: "age_groups[0] has invalid value",
		},
		{
			name:      "invalid social media hours",
			audience:  models.Audience{ID: "a1", SocialMediaHoursDaily: "10+"},
			wantErr:   true,
			errSubstr: "social_media_hours_daily has invalid value",
		},
		{
			name:      "negative purchases",
			audience:  models.Audience{ID: "a1", PurchasesLastMonth: -1},
			wantErr:   true,
			errSubstr: "purchases_last_month must not be negative",
		},
		{
			name:      "empty birth country entry",
			audience:  models.Audience{ID: "a1", BirthCountry: []string{""}},
			wantErr:   true,
			errSubstr: "birth_country[0] is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAudience(&tt.audience)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("expected error to contain %q, got: %v", tt.errSubstr, err)
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "valid description",
			description: "My favourite chart",
			wantErr:     false,
		},
		{
			name:        "empty description",
			description: "",
			wantErr:     true,
			errSubstr:   "description is required",
		},
		{
			name:        "whitespace-only description",
			description: "   ",
			wantErr:     true,
			errSubstr:   "description is required",
		},
		{
			name:        "description too long",
			description: strings.Repeat("d", 256),
			wantErr:     true,
			errSubstr:   "description exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDescription(tt.description)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("expected error to contain %q, got: %v", tt.errSubstr, err)
			}
		})
	}
}

func TestValidationErrorCollectsAllFieldErrors(t *testing.T) {
	// Verify that validation returns all errors at once, not just the first one.
	err := validateChart(&models.Chart{})
	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	// An empty chart should fail on: id, title, x_axis_title, y_axis_title
	if len(valErr.Errors) != 4 {
		t.Errorf("expected 4 validation errors, got %d: %v", len(valErr.Errors), valErr.Errors)
	}
}
