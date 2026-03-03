package sync

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/wick/github-star-manager/services/api/internal/auth"
	"github.com/wick/github-star-manager/services/api/internal/db"
	"github.com/wick/github-star-manager/services/api/internal/github"
)

type Service struct {
	repo                *db.Repository
	authSvc             *auth.Service
	ghClient            *github.Client
	schedulerTick       time.Duration
	schedulerMaxWorkers int
}

type SyncResult struct {
	Processed int    `json:"processed"`
	Cursor    string `json:"cursor"`
}

func NewService(repo *db.Repository, authSvc *auth.Service, ghClient *github.Client, schedulerTick time.Duration, schedulerMaxWorkers int) *Service {
	if schedulerTick <= 0 {
		schedulerTick = time.Minute
	}
	if schedulerMaxWorkers <= 0 {
		schedulerMaxWorkers = 3
	}
	return &Service{
		repo:                repo,
		authSvc:             authSvc,
		ghClient:            ghClient,
		schedulerTick:       schedulerTick,
		schedulerMaxWorkers: schedulerMaxWorkers,
	}
}

func (s *Service) SyncStars(ctx context.Context, userID int64) (SyncResult, error) {
	settings, err := s.repo.GetSyncSettings(ctx, userID)
	if err != nil {
		return SyncResult{}, fmt.Errorf("load sync settings: %w", err)
	}
	return s.syncStarsWithRetry(ctx, userID, settings.RetryMax)
}

func (s *Service) syncStarsWithRetry(ctx context.Context, userID int64, retryMax int) (SyncResult, error) {
	if retryMax < 0 {
		retryMax = 0
	}

	githubToken, err := s.authSvc.GetGitHubAccessToken(ctx, userID)
	if err != nil {
		return SyncResult{}, fmt.Errorf("resolve github token: %w", err)
	}

	jobID, err := s.repo.CreateSyncJob(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			return SyncResult{}, db.ErrConflict
		}
		return SyncResult{}, fmt.Errorf("create sync job: %w", err)
	}

	processed := 0
	cursor := ""
	page := 1
	etag := ""
	changedRepositoryIDs := make([]int64, 0)
	changedRepositoryIDSet := make(map[int64]struct{})

	for {
		var (
			items       []github.StarItem
			paging      github.Pagination
			rateLimited bool
			retryAfter  int
		)

		requestErr := s.retry(ctx, retryMax, func() error {
			resultItems, resultPaging, limited, after, listErr := s.ghClient.ListStarred(ctx, githubToken, page, etag)
			if listErr != nil {
				return listErr
			}
			items = resultItems
			paging = resultPaging
			rateLimited = limited
			retryAfter = after
			return nil
		})
		if requestErr != nil {
			_ = s.repo.FinishSyncJob(ctx, jobID, "failed", cursor, requestErr.Error())
			return SyncResult{}, fmt.Errorf("list github starred: %w", requestErr)
		}

		if rateLimited {
			wait := time.Duration(retryAfter) * time.Second
			if wait <= 0 {
				wait = time.Minute
			}
			select {
			case <-ctx.Done():
				_ = s.repo.FinishSyncJob(ctx, jobID, "failed", cursor, "context cancelled")
				return SyncResult{}, ctx.Err()
			case <-time.After(wait):
				continue
			}
		}

		if len(items) == 0 && paging.ETag != "" {
			cursor = paging.ETag
			break
		}

		records := make([]db.StarRecord, 0, len(items))
		for _, item := range items {
			records = append(records, db.StarRecord{
				GitHubRepoID:    item.Repository.ID,
				OwnerLogin:      item.Repository.Owner.Login,
				Name:            item.Repository.Name,
				FullName:        item.Repository.FullName,
				Private:         item.Repository.Private,
				HTMLURL:         item.Repository.HTMLURL,
				Description:     item.Repository.Description,
				Language:        item.Repository.Language,
				StargazersCount: item.Repository.StargazersCount,
				PushedAt:        item.Repository.PushedAt,
				StarredAt:       item.StarredAt,
				LastSeenAt:      time.Now(),
			})
		}
		repositoryIDs, err := s.repo.UpsertRepositoriesAndStars(ctx, userID, records)
		if err != nil {
			_ = s.repo.FinishSyncJob(ctx, jobID, "failed", cursor, err.Error())
			return SyncResult{}, fmt.Errorf("save star records: %w", err)
		}
		for _, repositoryID := range repositoryIDs {
			if _, exists := changedRepositoryIDSet[repositoryID]; exists {
				continue
			}
			changedRepositoryIDSet[repositoryID] = struct{}{}
			changedRepositoryIDs = append(changedRepositoryIDs, repositoryID)
		}
		processed += len(records)

		etag = paging.ETag
		cursor = paging.ETag
		if paging.NextPage == 0 {
			break
		}
		page = paging.NextPage
	}

	if _, err := s.repo.ApplyEnabledSmartRulesForRepositories(ctx, userID, changedRepositoryIDs); err != nil {
		if _, fallbackErr := s.repo.ApplyEnabledSmartRules(ctx, userID); fallbackErr != nil {
			_ = s.repo.FinishSyncJob(ctx, jobID, "failed", cursor, fallbackErr.Error())
			return SyncResult{}, fmt.Errorf("apply smart rules fallback: %w", fallbackErr)
		}
	}

	if err := s.repo.FinishSyncJob(ctx, jobID, "success", cursor, ""); err != nil {
		return SyncResult{}, fmt.Errorf("finish sync job: %w", err)
	}

	return SyncResult{Processed: processed, Cursor: cursor}, nil
}

func (s *Service) retry(ctx context.Context, retryMax int, fn func() error) error {
	attempt := 0
	for {
		err := fn()
		if err == nil {
			return nil
		}
		if attempt >= retryMax {
			return err
		}
		attempt++
		wait := time.Duration(1<<uint(attempt-1)) * time.Second
		if wait > 10*time.Second {
			wait = 10 * time.Second
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

func (s *Service) LatestSyncStatus(ctx context.Context, userID int64) (db.SyncJob, error) {
	item, err := s.repo.LatestSyncStatus(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return db.SyncJob{}, db.ErrNotFound
		}
		return db.SyncJob{}, fmt.Errorf("latest sync status: %w", err)
	}
	return item, nil
}

func (s *Service) GetSyncSettings(ctx context.Context, userID int64) (db.SyncSettings, error) {
	item, err := s.repo.GetSyncSettings(ctx, userID)
	if err != nil {
		return db.SyncSettings{}, fmt.Errorf("get sync settings: %w", err)
	}
	return item, nil
}

func (s *Service) UpdateSyncSettings(ctx context.Context, userID int64, enabled bool, intervalHours int, retryMax int) (db.SyncSettings, error) {
	if intervalHours != 6 && intervalHours != 12 && intervalHours != 24 {
		return db.SyncSettings{}, fmt.Errorf("intervalHours must be one of 6/12/24")
	}
	if retryMax < 0 || retryMax > 5 {
		return db.SyncSettings{}, fmt.Errorf("retryMax must be between 0 and 5")
	}
	item, err := s.repo.UpsertSyncSettings(ctx, userID, enabled, intervalHours, retryMax)
	if err != nil {
		return db.SyncSettings{}, fmt.Errorf("update sync settings: %w", err)
	}
	return item, nil
}

func (s *Service) ListSmartRules(ctx context.Context, userID int64) ([]db.SmartRule, error) {
	items, err := s.repo.ListSmartRules(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list smart rules: %w", err)
	}
	return items, nil
}

func (s *Service) CreateSmartRule(ctx context.Context, userID int64, item db.SmartRule) (db.SmartRule, error) {
	item.Name = strings.TrimSpace(item.Name)
	item.LanguageEquals = strings.TrimSpace(item.LanguageEquals)
	item.OwnerContains = strings.TrimSpace(item.OwnerContains)
	item.NameContains = strings.TrimSpace(item.NameContains)
	item.DescriptionContains = strings.TrimSpace(item.DescriptionContains)
	if item.Name == "" {
		return db.SmartRule{}, fmt.Errorf("name is required")
	}
	if len(item.Name) > 80 {
		return db.SmartRule{}, fmt.Errorf("name too long")
	}
	if len(item.LanguageEquals) > 32 || len(item.OwnerContains) > 100 || len(item.NameContains) > 120 || len(item.DescriptionContains) > 200 {
		return db.SmartRule{}, fmt.Errorf("rule condition too long")
	}
	if item.TagID <= 0 {
		return db.SmartRule{}, fmt.Errorf("tagId is required")
	}
	if item.LanguageEquals == "" && item.OwnerContains == "" && item.NameContains == "" && item.DescriptionContains == "" {
		return db.SmartRule{}, fmt.Errorf("at least one match condition is required")
	}
	created, err := s.repo.CreateSmartRule(ctx, userID, item)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return db.SmartRule{}, db.ErrNotFound
		}
		return db.SmartRule{}, fmt.Errorf("create smart rule: %w", err)
	}
	return created, nil
}

func (s *Service) DeleteSmartRule(ctx context.Context, userID int64, ruleID int64) error {
	if ruleID <= 0 {
		return fmt.Errorf("invalid rule id")
	}
	if err := s.repo.DeleteSmartRule(ctx, userID, ruleID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return db.ErrNotFound
		}
		return fmt.Errorf("delete smart rule: %w", err)
	}
	return nil
}

func (s *Service) ApplyRulesNow(ctx context.Context, userID int64) (int64, error) {
	affected, err := s.repo.ApplyEnabledSmartRules(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("apply rules: %w", err)
	}
	return affected, nil
}

func (s *Service) GetGovernanceMetrics(ctx context.Context, userID int64) (db.GovernanceMetrics, error) {
	item, err := s.repo.GetGovernanceMetrics(ctx, userID)
	if err != nil {
		return db.GovernanceMetrics{}, fmt.Errorf("get governance metrics: %w", err)
	}
	return item, nil
}

func (s *Service) ExportData(ctx context.Context, userID int64) (db.ExportPayload, error) {
	item, err := s.repo.BuildExportPayload(ctx, userID)
	if err != nil {
		return db.ExportPayload{}, fmt.Errorf("export data: %w", err)
	}
	return item, nil
}

func (s *Service) ImportData(ctx context.Context, userID int64, payload db.ImportPayload) (db.ImportResult, error) {
	const (
		maxImportTags        = 200
		maxImportRules       = 200
		maxImportNotes       = 1000
		maxImportTagBindings = 5000
	)
	if payload.SyncSettings != nil {
		if payload.SyncSettings.IntervalHours != 6 && payload.SyncSettings.IntervalHours != 12 && payload.SyncSettings.IntervalHours != 24 {
			return db.ImportResult{}, fmt.Errorf("syncSettings.intervalHours must be one of 6/12/24")
		}
		if payload.SyncSettings.RetryMax < 0 || payload.SyncSettings.RetryMax > 5 {
			return db.ImportResult{}, fmt.Errorf("syncSettings.retryMax must be between 0 and 5")
		}
	}
	if len(payload.Tags) > maxImportTags {
		return db.ImportResult{}, fmt.Errorf("too many tags")
	}
	if len(payload.SmartRules) > maxImportRules {
		return db.ImportResult{}, fmt.Errorf("too many smartRules")
	}
	if len(payload.Notes) > maxImportNotes {
		return db.ImportResult{}, fmt.Errorf("too many notes")
	}
	if len(payload.TagBindings) > maxImportTagBindings {
		return db.ImportResult{}, fmt.Errorf("too many tagBindings")
	}
	result, err := s.repo.ApplyImportPayload(ctx, userID, payload)
	if err != nil {
		return db.ImportResult{}, fmt.Errorf("import data: %w", err)
	}
	return result, nil
}

func (s *Service) StartScheduler(ctx context.Context) {
	ticker := time.NewTicker(s.schedulerTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runSchedulerTick(ctx)
		}
	}
}

func (s *Service) runSchedulerTick(ctx context.Context) {
	users, err := s.repo.ListUsersDueForSync(ctx)
	if err != nil {
		log.Printf("sync scheduler list due users failed: %v", err)
		return
	}
	if len(users) == 0 {
		return
	}

	workerCount := s.schedulerMaxWorkers
	if workerCount > len(users) {
		workerCount = len(users)
	}

	jobs := make(chan int64)
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for userID := range jobs {
				settings, err := s.repo.GetSyncSettings(ctx, userID)
				if err != nil {
					log.Printf("sync scheduler load settings failed user=%d err=%v", userID, err)
					continue
				}
				if !settings.Enabled {
					continue
				}
				if _, err := s.syncStarsWithRetry(ctx, userID, settings.RetryMax); err != nil {
					log.Printf("sync scheduler sync failed user=%d err=%v", userID, err)
				}
			}
		}()
	}

	for _, userID := range users {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case jobs <- userID:
		}
	}
	close(jobs)
	wg.Wait()
}
