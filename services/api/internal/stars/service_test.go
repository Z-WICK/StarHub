package stars

import (
	"context"
	"testing"

	"github.com/wick/github-star-manager/services/api/internal/db"
)

type fakeRepo struct{}

func TestCreateTagValidation(t *testing.T) {
	svc := &Service{repo: &db.Repository{}}

	_, err := svc.CreateTag(context.Background(), 1, "", "#123456")
	if err == nil {
		t.Fatalf("expected error for empty tag name")
	}

	_, err = svc.CreateTag(context.Background(), 1, "tag", "bad")
	if err == nil {
		t.Fatalf("expected error for invalid color")
	}
}
