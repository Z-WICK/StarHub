package stars

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/wick/github-star-manager/services/api/internal/auth"
	"github.com/wick/github-star-manager/services/api/internal/db"
	"github.com/wick/github-star-manager/services/api/internal/github"
)

type Service struct {
	repo     *db.Repository
	authSvc  *auth.Service
	ghClient *github.Client
}

const maxBatchRepositoryCount = 100

type ListResult struct {
	Items []db.StarRecord `json:"items"`
	Total int             `json:"total"`
}

type ReadmeResult struct {
	Repository db.RepositoryRef `json:"repository"`
	Content    string           `json:"content"`
}

func NewService(repo *db.Repository, authSvc *auth.Service, ghClient *github.Client) *Service {
	return &Service{repo: repo, authSvc: authSvc, ghClient: ghClient}
}

func (s *Service) List(ctx context.Context, userID int64, filters db.StarFilters) (ListResult, error) {
	items, total, err := s.repo.ListStars(ctx, userID, filters)
	if err != nil {
		return ListResult{}, fmt.Errorf("list stars: %w", err)
	}
	return ListResult{Items: items, Total: total}, nil
}

func (s *Service) CreateTag(ctx context.Context, userID int64, name string, color string) (db.Tag, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return db.Tag{}, fmt.Errorf("tag name is required")
	}
	if len(trimmedName) > 50 {
		return db.Tag{}, fmt.Errorf("tag name too long")
	}
	if color == "" {
		color = "#4f46e5"
	}
	if len(color) != 7 || color[0] != '#' {
		return db.Tag{}, fmt.Errorf("invalid tag color")
	}

	tag, err := s.repo.CreateTag(ctx, userID, trimmedName, color)
	if err != nil {
		return db.Tag{}, fmt.Errorf("create tag: %w", err)
	}
	return tag, nil
}

func (s *Service) ListTags(ctx context.Context, userID int64) ([]db.Tag, error) {
	items, err := s.repo.ListTags(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return items, nil
}

func (s *Service) AssignTag(ctx context.Context, userID int64, repositoryID int64, tagID int64) error {
	if repositoryID <= 0 || tagID <= 0 {
		return fmt.Errorf("invalid repository or tag id")
	}
	if err := s.repo.AssignTag(ctx, userID, repositoryID, tagID); err != nil {
		return fmt.Errorf("assign tag: %w", err)
	}
	return nil
}

func (s *Service) UnassignTag(ctx context.Context, userID int64, repositoryID int64, tagID int64) error {
	if repositoryID <= 0 || tagID <= 0 {
		return fmt.Errorf("invalid repository or tag id")
	}
	if err := s.repo.UnassignTag(ctx, userID, repositoryID, tagID); err != nil {
		return fmt.Errorf("unassign tag: %w", err)
	}
	return nil
}

func (s *Service) UpsertNote(ctx context.Context, userID int64, repositoryID int64, content string) error {
	if repositoryID <= 0 {
		return fmt.Errorf("invalid repository id")
	}
	if len(content) > 5000 {
		return fmt.Errorf("note too long")
	}
	if err := s.repo.UpsertNote(ctx, userID, repositoryID, content); err != nil {
		return fmt.Errorf("save note: %w", err)
	}
	return nil
}

func (s *Service) BatchAssignTag(ctx context.Context, userID int64, repositoryIDs []int64, tagID int64) error {
	if tagID <= 0 {
		return fmt.Errorf("invalid tag id")
	}
	if len(repositoryIDs) == 0 || len(repositoryIDs) > maxBatchRepositoryCount {
		return fmt.Errorf("invalid repository ids")
	}
	for _, repositoryID := range repositoryIDs {
		if repositoryID <= 0 {
			return fmt.Errorf("invalid repository ids")
		}
	}
	if err := s.repo.BatchAssignTag(ctx, userID, repositoryIDs, tagID); err != nil {
		return fmt.Errorf("batch assign tag: %w", err)
	}
	return nil
}

func (s *Service) BatchUnassignTag(ctx context.Context, userID int64, repositoryIDs []int64, tagID int64) error {
	if tagID <= 0 {
		return fmt.Errorf("invalid tag id")
	}
	if len(repositoryIDs) == 0 || len(repositoryIDs) > maxBatchRepositoryCount {
		return fmt.Errorf("invalid repository ids")
	}
	for _, repositoryID := range repositoryIDs {
		if repositoryID <= 0 {
			return fmt.Errorf("invalid repository ids")
		}
	}
	if err := s.repo.BatchUnassignTag(ctx, userID, repositoryIDs, tagID); err != nil {
		return fmt.Errorf("batch unassign tag: %w", err)
	}
	return nil
}

func (s *Service) GetReadme(ctx context.Context, userID int64, repositoryID int64) (ReadmeResult, error) {
	if repositoryID <= 0 {
		return ReadmeResult{}, fmt.Errorf("invalid repository id")
	}

	repository, err := s.repo.GetRepositoryRef(ctx, userID, repositoryID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return ReadmeResult{}, db.ErrNotFound
		}
		return ReadmeResult{}, fmt.Errorf("get repository: %w", err)
	}

	githubToken, err := s.authSvc.GetGitHubAccessToken(ctx, userID)
	if err != nil {
		return ReadmeResult{}, fmt.Errorf("resolve github token: %w", err)
	}

	readme, err := s.ghClient.GetReadme(ctx, githubToken, repository.OwnerLogin, repository.Name)
	if err != nil {
		if errors.Is(err, github.ErrNotFound) {
			return ReadmeResult{}, github.ErrNotFound
		}
		return ReadmeResult{}, fmt.Errorf("fetch github readme: %w", err)
	}

	return ReadmeResult{Repository: repository, Content: readme.Content}, nil
}
