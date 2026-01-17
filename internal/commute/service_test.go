package commute_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/commute"
)

func TestService_Create(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	input := &models.CommuteCreateRequest{
		Label: "Home to Work",
		Origin: models.CommuteLocation{
			Point: models.Point{Lat: 52.370216, Lon: 4.895168},
		},
		Destination: models.CommuteLocation{
			Point: models.Point{Lat: 52.308056, Lon: 4.763889},
		},
		DaysOfWeek:                []int{1, 2, 3, 4, 5},
		PreferredArrivalTimeLocal: "09:00",
	}

	result, err := service.Create(ctx, "user123", input)
	if err != nil {
		t.Fatalf("failed to create commute: %v", err)
	}

	if result.ID == "" {
		t.Error("expected commute ID to be set")
	}
	if !strings.HasPrefix(result.ID, "cmt_") {
		t.Errorf("expected commute ID to start with 'cmt_', got %q", result.ID)
	}
	if result.Label != input.Label {
		t.Errorf("expected label %q, got %q", input.Label, result.Label)
	}
}

func TestService_Create_ValidationErrors(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	tests := []struct {
		name      string
		input     *models.CommuteCreateRequest
		wantField string
	}{
		{
			name: "empty label",
			input: &models.CommuteCreateRequest{
				Label: "",
				Origin: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				Destination: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				DaysOfWeek:                []int{1},
				PreferredArrivalTimeLocal: "09:00",
			},
			wantField: "label",
		},
		{
			name: "label too long",
			input: &models.CommuteCreateRequest{
				Label: strings.Repeat("a", 81),
				Origin: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				Destination: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				DaysOfWeek:                []int{1},
				PreferredArrivalTimeLocal: "09:00",
			},
			wantField: "label",
		},
		{
			name: "invalid latitude",
			input: &models.CommuteCreateRequest{
				Label: "Test",
				Origin: models.CommuteLocation{
					Point: models.Point{Lat: 91.0, Lon: 4.0},
				},
				Destination: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				DaysOfWeek:                []int{1},
				PreferredArrivalTimeLocal: "09:00",
			},
			wantField: "origin.point.lat",
		},
		{
			name: "invalid longitude",
			input: &models.CommuteCreateRequest{
				Label: "Test",
				Origin: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 181.0},
				},
				Destination: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				DaysOfWeek:                []int{1},
				PreferredArrivalTimeLocal: "09:00",
			},
			wantField: "origin.point.lon",
		},
		{
			name: "empty days of week",
			input: &models.CommuteCreateRequest{
				Label: "Test",
				Origin: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				Destination: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				DaysOfWeek:                []int{},
				PreferredArrivalTimeLocal: "09:00",
			},
			wantField: "daysOfWeek",
		},
		{
			name: "invalid day of week",
			input: &models.CommuteCreateRequest{
				Label: "Test",
				Origin: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				Destination: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				DaysOfWeek:                []int{8},
				PreferredArrivalTimeLocal: "09:00",
			},
			wantField: "daysOfWeek",
		},
		{
			name: "invalid time format",
			input: &models.CommuteCreateRequest{
				Label: "Test",
				Origin: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				Destination: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				DaysOfWeek:                []int{1},
				PreferredArrivalTimeLocal: "9:00 AM",
			},
			wantField: "preferredArrivalTimeLocal",
		},
		{
			name: "notes too long",
			input: &models.CommuteCreateRequest{
				Label: "Test",
				Origin: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				Destination: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				DaysOfWeek:                []int{1},
				PreferredArrivalTimeLocal: "09:00",
				Notes:                     strPtr(strings.Repeat("a", 501)),
			},
			wantField: "notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(ctx, "user123", tt.input)
			if err == nil {
				t.Fatal("expected validation error")
			}

			var validationErr *commute.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T", err)
			}

			found := false
			for _, fe := range validationErr.Errors {
				if fe.Field == tt.wantField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error for field %q, got errors: %v", tt.wantField, validationErr.Errors)
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	// Create a commute first
	input := &models.CommuteCreateRequest{
		Label: "Test Commute",
		Origin: models.CommuteLocation{
			Point: models.Point{Lat: 52.0, Lon: 4.0},
		},
		Destination: models.CommuteLocation{
			Point: models.Point{Lat: 52.1, Lon: 4.1},
		},
		DaysOfWeek:                []int{1, 2, 3},
		PreferredArrivalTimeLocal: "08:30",
	}

	created, err := service.Create(ctx, "user123", input)
	if err != nil {
		t.Fatalf("failed to create commute: %v", err)
	}

	// Get the commute
	result, err := service.Get(ctx, "user123", created.ID)
	if err != nil {
		t.Fatalf("failed to get commute: %v", err)
	}

	if result.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, result.ID)
	}
	if result.Label != input.Label {
		t.Errorf("expected label %q, got %q", input.Label, result.Label)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	_, err := service.Get(ctx, "user123", "nonexistent")
	if !errors.Is(err, commute.ErrCommuteNotFound) {
		t.Errorf("expected ErrCommuteNotFound, got %v", err)
	}
}

func TestService_Get_WrongUser(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	// Create a commute for user1
	input := &models.CommuteCreateRequest{
		Label: "Test",
		Origin: models.CommuteLocation{
			Point: models.Point{Lat: 52.0, Lon: 4.0},
		},
		Destination: models.CommuteLocation{
			Point: models.Point{Lat: 52.1, Lon: 4.1},
		},
		DaysOfWeek:                []int{1},
		PreferredArrivalTimeLocal: "09:00",
	}

	created, err := service.Create(ctx, "user1", input)
	if err != nil {
		t.Fatalf("failed to create commute: %v", err)
	}

	// Try to get it as user2
	_, err = service.Get(ctx, "user2", created.ID)
	if !errors.Is(err, commute.ErrCommuteNotFound) {
		t.Errorf("expected ErrCommuteNotFound for wrong user, got %v", err)
	}
}

func TestService_List(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	// Create multiple commutes
	for i := 0; i < 3; i++ {
		input := &models.CommuteCreateRequest{
			Label: "Test " + string(rune('A'+i)),
			Origin: models.CommuteLocation{
				Point: models.Point{Lat: 52.0, Lon: 4.0},
			},
			Destination: models.CommuteLocation{
				Point: models.Point{Lat: 52.1, Lon: 4.1},
			},
			DaysOfWeek:                []int{1},
			PreferredArrivalTimeLocal: "09:00",
		}
		_, err := service.Create(ctx, "user123", input)
		if err != nil {
			t.Fatalf("failed to create commute: %v", err)
		}
	}

	// List commutes
	result, err := service.List(ctx, "user123", 50)
	if err != nil {
		t.Fatalf("failed to list commutes: %v", err)
	}

	if len(result.Items) != 3 {
		t.Errorf("expected 3 commutes, got %d", len(result.Items))
	}
}

func TestService_List_OnlyOwnCommutes(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	// Create commutes for different users
	input := &models.CommuteCreateRequest{
		Label: "Test",
		Origin: models.CommuteLocation{
			Point: models.Point{Lat: 52.0, Lon: 4.0},
		},
		Destination: models.CommuteLocation{
			Point: models.Point{Lat: 52.1, Lon: 4.1},
		},
		DaysOfWeek:                []int{1},
		PreferredArrivalTimeLocal: "09:00",
	}

	_, _ = service.Create(ctx, "user1", input)
	_, _ = service.Create(ctx, "user1", input)
	_, _ = service.Create(ctx, "user2", input)

	// List for user1
	result, err := service.List(ctx, "user1", 50)
	if err != nil {
		t.Fatalf("failed to list commutes: %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("expected 2 commutes for user1, got %d", len(result.Items))
	}
}

func TestService_Update(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	// Create a commute
	input := &models.CommuteCreateRequest{
		Label: "Original",
		Origin: models.CommuteLocation{
			Point: models.Point{Lat: 52.0, Lon: 4.0},
		},
		Destination: models.CommuteLocation{
			Point: models.Point{Lat: 52.1, Lon: 4.1},
		},
		DaysOfWeek:                []int{1, 2, 3},
		PreferredArrivalTimeLocal: "09:00",
	}

	created, err := service.Create(ctx, "user123", input)
	if err != nil {
		t.Fatalf("failed to create commute: %v", err)
	}

	// Update it
	newLabel := "Updated"
	updateInput := &models.CommuteUpdateRequest{
		Label: &newLabel,
	}

	updated, err := service.Update(ctx, "user123", created.ID, updateInput)
	if err != nil {
		t.Fatalf("failed to update commute: %v", err)
	}

	if updated.Label != newLabel {
		t.Errorf("expected label %q, got %q", newLabel, updated.Label)
	}

	// Verify other fields unchanged
	if updated.PreferredArrivalTimeLocal != input.PreferredArrivalTimeLocal {
		t.Errorf("expected time %q unchanged, got %q", input.PreferredArrivalTimeLocal, updated.PreferredArrivalTimeLocal)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	newLabel := "Updated"
	updateInput := &models.CommuteUpdateRequest{
		Label: &newLabel,
	}

	_, err := service.Update(ctx, "user123", "nonexistent", updateInput)
	if !errors.Is(err, commute.ErrCommuteNotFound) {
		t.Errorf("expected ErrCommuteNotFound, got %v", err)
	}
}

func TestService_Delete(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	// Create a commute
	input := &models.CommuteCreateRequest{
		Label: "To Delete",
		Origin: models.CommuteLocation{
			Point: models.Point{Lat: 52.0, Lon: 4.0},
		},
		Destination: models.CommuteLocation{
			Point: models.Point{Lat: 52.1, Lon: 4.1},
		},
		DaysOfWeek:                []int{1},
		PreferredArrivalTimeLocal: "09:00",
	}

	created, err := service.Create(ctx, "user123", input)
	if err != nil {
		t.Fatalf("failed to create commute: %v", err)
	}

	// Delete it
	err = service.Delete(ctx, "user123", created.ID)
	if err != nil {
		t.Fatalf("failed to delete commute: %v", err)
	}

	// Verify it's gone
	_, err = service.Get(ctx, "user123", created.ID)
	if !errors.Is(err, commute.ErrCommuteNotFound) {
		t.Errorf("expected ErrCommuteNotFound after delete, got %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	err := service.Delete(ctx, "user123", "nonexistent")
	if !errors.Is(err, commute.ErrCommuteNotFound) {
		t.Errorf("expected ErrCommuteNotFound, got %v", err)
	}
}

func TestService_Delete_WrongUser(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	// Create a commute for user1
	input := &models.CommuteCreateRequest{
		Label: "Test",
		Origin: models.CommuteLocation{
			Point: models.Point{Lat: 52.0, Lon: 4.0},
		},
		Destination: models.CommuteLocation{
			Point: models.Point{Lat: 52.1, Lon: 4.1},
		},
		DaysOfWeek:                []int{1},
		PreferredArrivalTimeLocal: "09:00",
	}

	created, err := service.Create(ctx, "user1", input)
	if err != nil {
		t.Fatalf("failed to create commute: %v", err)
	}

	// Try to delete as user2
	err = service.Delete(ctx, "user2", created.ID)
	if !errors.Is(err, commute.ErrCommuteNotFound) {
		t.Errorf("expected ErrCommuteNotFound for wrong user, got %v", err)
	}
}

func TestService_ValidTimeFormats(t *testing.T) {
	repo := commute.NewInMemoryRepository()
	service := commute.NewService(repo)
	ctx := context.Background()

	validTimes := []string{
		"00:00",
		"09:00",
		"12:30",
		"23:59",
		"9:00", // Single digit hour should be valid
	}

	for _, time := range validTimes {
		t.Run(time, func(t *testing.T) {
			input := &models.CommuteCreateRequest{
				Label: "Test",
				Origin: models.CommuteLocation{
					Point: models.Point{Lat: 52.0, Lon: 4.0},
				},
				Destination: models.CommuteLocation{
					Point: models.Point{Lat: 52.1, Lon: 4.1},
				},
				DaysOfWeek:                []int{1},
				PreferredArrivalTimeLocal: time,
			}

			_, err := service.Create(ctx, "user123", input)
			if err != nil {
				t.Errorf("expected time %q to be valid, got error: %v", time, err)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
