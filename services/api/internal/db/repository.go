package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type Repository struct {
	store *PostgresStore
}

func NewRepository(store *PostgresStore) *Repository {
	return &Repository{store: store}
}

func (r *Repository) EnsureUserByGitHubID(ctx context.Context, githubUserID int64, displayName string) (int64, error) {
	selectQuery := `SELECT user_id FROM github_accounts WHERE github_user_id = $1`
	var existingUserID int64
	if err := r.store.Pool.QueryRow(ctx, selectQuery, githubUserID).Scan(&existingUserID); err == nil {
		return existingUserID, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("find user by github id: %w", err)
	}

	insertQuery := `
		INSERT INTO users (display_name)
		VALUES ($1)
		RETURNING id`
	var newUserID int64
	if err := r.store.Pool.QueryRow(ctx, insertQuery, displayName).Scan(&newUserID); err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}
	return newUserID, nil
}

func (r *Repository) SaveGitHubAccount(ctx context.Context, userID int64, githubUserID int64, login string, avatarURL string, tokenEncrypted string, scopes string) error {
	query := `
		INSERT INTO github_accounts (user_id, github_user_id, login, avatar_url, token_encrypted, token_scopes)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			github_user_id = EXCLUDED.github_user_id,
			login = EXCLUDED.login,
			avatar_url = EXCLUDED.avatar_url,
			token_encrypted = EXCLUDED.token_encrypted,
			token_scopes = EXCLUDED.token_scopes,
			updated_at = NOW()`
	_, err := r.store.Pool.Exec(ctx, query, userID, githubUserID, login, avatarURL, tokenEncrypted, scopes)
	if err != nil {
		return fmt.Errorf("save github account: %w", err)
	}
	return nil
}

func (r *Repository) GetGitHubAccountByUserID(ctx context.Context, userID int64) (int64, string, string, string, error) {
	query := `
		SELECT github_user_id, login, avatar_url, token_encrypted
		FROM github_accounts
		WHERE user_id = $1`
	var githubUserID int64
	var login, avatarURL, tokenEncrypted string
	if err := r.store.Pool.QueryRow(ctx, query, userID).Scan(&githubUserID, &login, &avatarURL, &tokenEncrypted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", "", "", ErrNotFound
		}
		return 0, "", "", "", fmt.Errorf("get github account: %w", err)
	}
	return githubUserID, login, avatarURL, tokenEncrypted, nil
}

func (r *Repository) UpsertRepositoryAndStar(ctx context.Context, userID int64, star StarRecord) error {
	tx, err := r.store.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := r.upsertRepositoryAndStarTx(ctx, tx, userID, star); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *Repository) UpsertRepositoriesAndStars(ctx context.Context, userID int64, stars []StarRecord) ([]int64, error) {
	if len(stars) == 0 {
		return nil, nil
	}

	tx, err := r.store.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin batch upsert tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	repositoryIDs := make([]int64, 0, len(stars))
	repositoryIDSet := make(map[int64]struct{}, len(stars))
	for _, star := range stars {
		repositoryID, err := r.upsertRepositoryAndStarTx(ctx, tx, userID, star)
		if err != nil {
			return nil, err
		}
		if _, exists := repositoryIDSet[repositoryID]; exists {
			continue
		}
		repositoryIDSet[repositoryID] = struct{}{}
		repositoryIDs = append(repositoryIDs, repositoryID)
	}

	if err := r.upsertStarSearchDocsForRepositoryIDs(ctx, tx, userID, repositoryIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit batch upsert tx: %w", err)
	}
	return repositoryIDs, nil
}

func (r *Repository) upsertRepositoryAndStarTx(ctx context.Context, tx pgx.Tx, userID int64, star StarRecord) (int64, error) {
	repoQuery := `
		INSERT INTO repositories (github_repo_id, owner_login, name, full_name, private, html_url, description, language, stargazers_count, pushed_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
		ON CONFLICT (github_repo_id) DO UPDATE SET
			owner_login = EXCLUDED.owner_login,
			name = EXCLUDED.name,
			full_name = EXCLUDED.full_name,
			private = EXCLUDED.private,
			html_url = EXCLUDED.html_url,
			description = EXCLUDED.description,
			language = EXCLUDED.language,
			stargazers_count = EXCLUDED.stargazers_count,
			pushed_at = EXCLUDED.pushed_at,
			updated_at = NOW()
		RETURNING id`

	var repositoryID int64
	if err := tx.QueryRow(
		ctx,
		repoQuery,
		star.GitHubRepoID,
		star.OwnerLogin,
		star.Name,
		star.FullName,
		star.Private,
		star.HTMLURL,
		star.Description,
		star.Language,
		star.StargazersCount,
		star.PushedAt,
	).Scan(&repositoryID); err != nil {
		return 0, fmt.Errorf("upsert repository: %w", err)
	}

	starQuery := `
		INSERT INTO stars (user_id, repository_id, starred_at, last_seen_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, repository_id) DO UPDATE SET
			starred_at = EXCLUDED.starred_at,
			last_seen_at = NOW()`
	if _, err := tx.Exec(ctx, starQuery, userID, repositoryID, star.StarredAt); err != nil {
		return 0, fmt.Errorf("upsert star: %w", err)
	}

	return repositoryID, nil
}

func (r *Repository) upsertStarSearchDocTx(ctx context.Context, tx pgx.Tx, userID int64, repositoryID int64) error {
	query := `
		INSERT INTO star_search_docs (user_id, repository_id, doc, updated_at)
		SELECT
			$1,
			$2,
			to_tsvector('simple', COALESCE(repo.full_name, '') || ' ' || COALESCE(repo.description, '') || ' ' || COALESCE(note.content, '')),
			NOW()
		FROM repositories repo
		LEFT JOIN notes note ON note.user_id = $1 AND note.repository_id = $2
		WHERE repo.id = $2
		ON CONFLICT (user_id, repository_id) DO UPDATE SET
			doc = EXCLUDED.doc,
			updated_at = NOW()`
	if _, err := tx.Exec(ctx, query, userID, repositoryID); err != nil {
		return fmt.Errorf("upsert star search doc: %w", err)
	}
	return nil
}

func (r *Repository) upsertStarSearchDocsForRepositoryIDs(ctx context.Context, tx pgx.Tx, userID int64, repositoryIDs []int64) error {
	if len(repositoryIDs) == 0 {
		return nil
	}
	query := `
		INSERT INTO star_search_docs (user_id, repository_id, doc, updated_at)
		SELECT
			$1,
			s.repository_id,
			to_tsvector('simple', COALESCE(repo.full_name, '') || ' ' || COALESCE(repo.description, '') || ' ' || COALESCE(note.content, '')),
			NOW()
		FROM stars s
		JOIN repositories repo ON repo.id = s.repository_id
		LEFT JOIN notes note ON note.user_id = s.user_id AND note.repository_id = s.repository_id
		WHERE s.user_id = $1
		  AND s.repository_id = ANY($2::bigint[])
		ON CONFLICT (user_id, repository_id) DO UPDATE SET
			doc = EXCLUDED.doc,
			updated_at = NOW()`
	if _, err := tx.Exec(ctx, query, userID, repositoryIDs); err != nil {
		return fmt.Errorf("upsert star search docs batch: %w", err)
	}
	return nil
}

func (r *Repository) ListStars(ctx context.Context, userID int64, filters StarFilters) ([]StarRecord, int, error) {
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.Limit <= 0 || filters.Limit > 100 {
		filters.Limit = 20
	}

	sortColumn := "s.starred_at"
	switch strings.ToLower(filters.SortBy) {
	case "starred_at":
		sortColumn = "s.starred_at"
	case "pushed_at":
		sortColumn = "r.pushed_at"
	case "stargazers_count":
		sortColumn = "r.stargazers_count"
	case "updated_at":
		sortColumn = "r.updated_at"
	}
	sortOrder := "DESC"
	if strings.EqualFold(filters.SortOrder, "asc") {
		sortOrder = "ASC"
	}

	joinSearchClause := ""
	whereParts := []string{"s.user_id = $1"}
	args := []any{userID}
	argIndex := 2

	if filters.Query != "" {
		joinSearchClause = fmt.Sprintf(" JOIN star_search_docs sd ON sd.user_id = s.user_id AND sd.repository_id = s.repository_id AND sd.doc @@ plainto_tsquery('simple', $%d)", argIndex)
		args = append(args, filters.Query)
		argIndex++
	}
	if filters.Language != "" {
		whereParts = append(whereParts, fmt.Sprintf("r.language = $%d", argIndex))
		args = append(args, filters.Language)
		argIndex++
	}
	if filters.TagID > 0 {
		whereParts = append(whereParts, fmt.Sprintf("EXISTS (SELECT 1 FROM repository_tags rt WHERE rt.user_id = s.user_id AND rt.repository_id = s.repository_id AND rt.tag_id = $%d)", argIndex))
		args = append(args, filters.TagID)
		argIndex++
	}
	if filters.HasNote != nil {
		if *filters.HasNote {
			whereParts = append(whereParts, "COALESCE(n.content, '') <> ''")
		} else {
			whereParts = append(whereParts, "COALESCE(n.content, '') = ''")
		}
	}

	whereClause := strings.Join(whereParts, " AND ")
	offset := (filters.Page - 1) * filters.Limit
	listQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) OVER() AS total_count,
			r.id,
			r.github_repo_id,
			r.owner_login,
			r.name,
			r.full_name,
			r.private,
			r.html_url,
			r.description,
			r.language,
			r.stargazers_count,
			r.pushed_at,
			s.starred_at,
			s.last_seen_at,
			COALESCE(n.content, '')
		FROM stars s
		JOIN repositories r ON r.id = s.repository_id
		LEFT JOIN notes n ON n.user_id = s.user_id AND n.repository_id = s.repository_id
		%s
		WHERE %s
		ORDER BY %s %s NULLS LAST, s.repository_id DESC
		LIMIT $%d OFFSET $%d`, joinSearchClause, whereClause, sortColumn, sortOrder, argIndex, argIndex+1)

	args = append(args, filters.Limit, offset)

	rows, err := r.store.Pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stars: %w", err)
	}
	defer rows.Close()

	total := 0
	items := make([]StarRecord, 0)
	repositoryIDs := make([]int64, 0)
	for rows.Next() {
		var item StarRecord
		var totalCount int
		if err := rows.Scan(
			&totalCount,
			&item.RepositoryID,
			&item.GitHubRepoID,
			&item.OwnerLogin,
			&item.Name,
			&item.FullName,
			&item.Private,
			&item.HTMLURL,
			&item.Description,
			&item.Language,
			&item.StargazersCount,
			&item.PushedAt,
			&item.StarredAt,
			&item.LastSeenAt,
			&item.Note,
		); err != nil {
			return nil, 0, fmt.Errorf("scan star: %w", err)
		}
		total = totalCount
		items = append(items, item)
		repositoryIDs = append(repositoryIDs, item.RepositoryID)
	}

	tagMap, err := r.GetTagsByRepositoryIDs(ctx, userID, repositoryIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("load tags: %w", err)
	}

	for i := range items {
		items[i].Tags = tagMap[items[i].RepositoryID]
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate stars: %w", err)
	}

	return items, total, nil
}

func (r *Repository) GetRepositoryRef(ctx context.Context, userID int64, repositoryID int64) (RepositoryRef, error) {
	query := `
		SELECT r.id, r.owner_login, r.name, r.full_name
		FROM stars s
		JOIN repositories r ON r.id = s.repository_id
		WHERE s.user_id = $1 AND r.id = $2
		LIMIT 1`

	var item RepositoryRef
	if err := r.store.Pool.QueryRow(ctx, query, userID, repositoryID).Scan(
		&item.RepositoryID,
		&item.OwnerLogin,
		&item.Name,
		&item.FullName,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RepositoryRef{}, ErrNotFound
		}
		return RepositoryRef{}, fmt.Errorf("get repository ref: %w", err)
	}

	return item, nil
}

func (r *Repository) CreateTag(ctx context.Context, userID int64, name string, color string) (Tag, error) {
	query := `
		INSERT INTO tags (user_id, name, color)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, name) DO UPDATE SET color = EXCLUDED.color
		RETURNING id, name, color`
	var tag Tag
	if err := r.store.Pool.QueryRow(ctx, query, userID, name, color).Scan(&tag.ID, &tag.Name, &tag.Color); err != nil {
		return Tag{}, fmt.Errorf("create tag: %w", err)
	}
	return tag, nil
}

func (r *Repository) ListTags(ctx context.Context, userID int64) ([]Tag, error) {
	query := `SELECT id, name, color FROM tags WHERE user_id = $1 ORDER BY name ASC`
	rows, err := r.store.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	items := make([]Tag, 0)
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		items = append(items, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return items, nil
}

func (r *Repository) AssignTag(ctx context.Context, userID int64, repositoryID int64, tagID int64) error {
	query := `
		INSERT INTO repository_tags (user_id, repository_id, tag_id)
		SELECT $1, $2, t.id
		FROM tags t
		WHERE t.id = $3 AND t.user_id = $1
		ON CONFLICT (user_id, repository_id, tag_id) DO NOTHING`
	result, err := r.store.Pool.Exec(ctx, query, userID, repositoryID, tagID)
	if err != nil {
		return fmt.Errorf("assign tag: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) UnassignTag(ctx context.Context, userID int64, repositoryID int64, tagID int64) error {
	query := `DELETE FROM repository_tags WHERE user_id = $1 AND repository_id = $2 AND tag_id = $3`
	_, err := r.store.Pool.Exec(ctx, query, userID, repositoryID, tagID)
	if err != nil {
		return fmt.Errorf("unassign tag: %w", err)
	}
	return nil
}

func (r *Repository) BatchAssignTag(ctx context.Context, userID int64, repositoryIDs []int64, tagID int64) error {
	tx, err := r.store.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin batch assign tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	validateQuery := `SELECT 1 FROM tags WHERE id = $1 AND user_id = $2 LIMIT 1`
	var exists int
	if err := tx.QueryRow(ctx, validateQuery, tagID, userID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("validate tag ownership: %w", err)
	}

	query := `
		INSERT INTO repository_tags (user_id, repository_id, tag_id)
		SELECT $1, unnest($2::bigint[]), $3
		ON CONFLICT (user_id, repository_id, tag_id) DO NOTHING`
	if _, err := tx.Exec(ctx, query, userID, repositoryIDs, tagID); err != nil {
		return fmt.Errorf("batch assign tag: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit batch assign tx: %w", err)
	}
	return nil
}

func (r *Repository) BatchUnassignTag(ctx context.Context, userID int64, repositoryIDs []int64, tagID int64) error {
	tx, err := r.store.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin batch unassign tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	query := `
		DELETE FROM repository_tags
		WHERE user_id = $1
		  AND tag_id = $3
		  AND repository_id = ANY($2::bigint[])`
	if _, err := tx.Exec(ctx, query, userID, repositoryIDs, tagID); err != nil {
		return fmt.Errorf("batch unassign tag: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit batch unassign tx: %w", err)
	}
	return nil
}

func (r *Repository) GetTagsByRepositoryIDs(ctx context.Context, userID int64, repositoryIDs []int64) (map[int64][]Tag, error) {
	if len(repositoryIDs) == 0 {
		return map[int64][]Tag{}, nil
	}

	query := `
		SELECT rt.repository_id, t.id, t.name, t.color
		FROM repository_tags rt
		JOIN tags t ON t.id = rt.tag_id
		WHERE rt.user_id = $1 AND t.user_id = $1 AND rt.repository_id = ANY($2)
		ORDER BY rt.repository_id ASC, t.name ASC`

	rows, err := r.store.Pool.Query(ctx, query, userID, repositoryIDs)
	if err != nil {
		return nil, fmt.Errorf("get tags by repositories: %w", err)
	}
	defer rows.Close()

	tagMap := make(map[int64][]Tag)
	for rows.Next() {
		var repositoryID int64
		var tag Tag
		if err := rows.Scan(&repositoryID, &tag.ID, &tag.Name, &tag.Color); err != nil {
			return nil, fmt.Errorf("scan repository tag: %w", err)
		}
		tagMap[repositoryID] = append(tagMap[repositoryID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate repository tags: %w", err)
	}
	return tagMap, nil
}

func (r *Repository) UpsertNote(ctx context.Context, userID int64, repositoryID int64, content string) error {
	tx, err := r.store.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin note tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	query := `
		INSERT INTO notes (user_id, repository_id, content, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, repository_id) DO UPDATE SET
			content = EXCLUDED.content,
			updated_at = NOW()`
	if _, err := tx.Exec(ctx, query, userID, repositoryID, content); err != nil {
		return fmt.Errorf("upsert note: %w", err)
	}

	if err := r.upsertStarSearchDocTx(ctx, tx, userID, repositoryID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit note tx: %w", err)
	}
	return nil
}

func (r *Repository) CreateSyncJob(ctx context.Context, userID int64) (int64, error) {
	tx, err := r.store.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin create sync job tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	lockQuery := `SELECT pg_try_advisory_xact_lock($1)`
	var locked bool
	if err := tx.QueryRow(ctx, lockQuery, userID).Scan(&locked); err != nil {
		return 0, fmt.Errorf("acquire sync lock: %w", err)
	}
	if !locked {
		return 0, ErrConflict
	}

	checkQuery := `
		SELECT id
		FROM sync_jobs
		WHERE user_id = $1 AND status = 'running'
		ORDER BY started_at DESC
		LIMIT 1`
	var runningID int64
	if err := tx.QueryRow(ctx, checkQuery, userID).Scan(&runningID); err == nil {
		return 0, ErrConflict
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("check running sync job: %w", err)
	}

	createQuery := `INSERT INTO sync_jobs (user_id, status) VALUES ($1, 'running') RETURNING id`
	var id int64
	if err := tx.QueryRow(ctx, createQuery, userID).Scan(&id); err != nil {
		return 0, fmt.Errorf("create sync job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit create sync job tx: %w", err)
	}
	return id, nil
}

func (r *Repository) FinishSyncJob(ctx context.Context, jobID int64, status string, cursor string, errMessage string) error {
	query := `
		UPDATE sync_jobs
		SET status = $2, finished_at = NOW(), cursor = $3, error_message = $4
		WHERE id = $1`
	_, err := r.store.Pool.Exec(ctx, query, jobID, status, cursor, errMessage)
	if err != nil {
		return fmt.Errorf("finish sync job: %w", err)
	}
	return nil
}

func (r *Repository) LatestSyncStatus(ctx context.Context, userID int64) (SyncJob, error) {
	query := `
		SELECT id, status, started_at, finished_at, cursor, error_message
		FROM sync_jobs
		WHERE user_id = $1
		ORDER BY started_at DESC
		LIMIT 1`
	var item SyncJob
	if err := r.store.Pool.QueryRow(ctx, query, userID).Scan(&item.ID, &item.Status, &item.StartedAt, &item.FinishedAt, &item.Cursor, &item.ErrorMessage); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SyncJob{}, ErrNotFound
		}
		return SyncJob{}, fmt.Errorf("latest sync status: %w", err)
	}
	return item, nil
}

func (r *Repository) GetSyncSettings(ctx context.Context, userID int64) (SyncSettings, error) {
	insertQuery := `
		INSERT INTO user_sync_settings (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING`
	if _, err := r.store.Pool.Exec(ctx, insertQuery, userID); err != nil {
		return SyncSettings{}, fmt.Errorf("ensure sync settings: %w", err)
	}

	query := `
		SELECT enabled, interval_hours, retry_max, updated_at
		FROM user_sync_settings
		WHERE user_id = $1`
	var item SyncSettings
	if err := r.store.Pool.QueryRow(ctx, query, userID).Scan(&item.Enabled, &item.IntervalHours, &item.RetryMax, &item.UpdatedAt); err != nil {
		return SyncSettings{}, fmt.Errorf("get sync settings: %w", err)
	}
	return item, nil
}

func (r *Repository) UpsertSyncSettings(ctx context.Context, userID int64, enabled bool, intervalHours int, retryMax int) (SyncSettings, error) {
	query := `
		INSERT INTO user_sync_settings (user_id, enabled, interval_hours, retry_max, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			interval_hours = EXCLUDED.interval_hours,
			retry_max = EXCLUDED.retry_max,
			updated_at = NOW()
		RETURNING enabled, interval_hours, retry_max, updated_at`
	var item SyncSettings
	if err := r.store.Pool.QueryRow(ctx, query, userID, enabled, intervalHours, retryMax).Scan(&item.Enabled, &item.IntervalHours, &item.RetryMax, &item.UpdatedAt); err != nil {
		return SyncSettings{}, fmt.Errorf("upsert sync settings: %w", err)
	}
	return item, nil
}

func (r *Repository) ListUsersDueForSync(ctx context.Context) ([]int64, error) {
	query := `
		SELECT s.user_id
		FROM user_sync_settings s
		JOIN github_accounts ga ON ga.user_id = s.user_id
		LEFT JOIN LATERAL (
			SELECT sj.started_at
			FROM sync_jobs sj
			WHERE sj.user_id = s.user_id
			ORDER BY sj.started_at DESC
			LIMIT 1
		) last_job ON true
		WHERE s.enabled = true
		  AND (
			last_job.started_at IS NULL
			OR last_job.started_at <= NOW() - (s.interval_hours || ' hours')::interval
		  )`
	rows, err := r.store.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users due for sync: %w", err)
	}
	defer rows.Close()

	items := make([]int64, 0)
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan due user: %w", err)
		}
		items = append(items, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due users: %w", err)
	}
	return items, nil
}

func (r *Repository) ListSmartRules(ctx context.Context, userID int64) ([]SmartRule, error) {
	query := `
		SELECT id, name, enabled, language_equals, owner_contains, name_contains, description_contains, tag_id, created_at
		FROM smart_rules
		WHERE user_id = $1
		ORDER BY id ASC`
	rows, err := r.store.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list smart rules: %w", err)
	}
	defer rows.Close()

	items := make([]SmartRule, 0)
	for rows.Next() {
		var item SmartRule
		if err := rows.Scan(&item.ID, &item.Name, &item.Enabled, &item.LanguageEquals, &item.OwnerContains, &item.NameContains, &item.DescriptionContains, &item.TagID, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan smart rule: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate smart rules: %w", err)
	}
	return items, nil
}

func (r *Repository) CreateSmartRule(ctx context.Context, userID int64, item SmartRule) (SmartRule, error) {
	query := `
		INSERT INTO smart_rules (user_id, name, enabled, language_equals, owner_contains, name_contains, description_contains, tag_id)
		SELECT $1, $2, $3, $4, $5, $6, $7, t.id
		FROM tags t
		WHERE t.id = $8 AND t.user_id = $1
		RETURNING id, name, enabled, language_equals, owner_contains, name_contains, description_contains, tag_id, created_at`
	var created SmartRule
	if err := r.store.Pool.QueryRow(
		ctx,
		query,
		userID,
		item.Name,
		item.Enabled,
		item.LanguageEquals,
		item.OwnerContains,
		item.NameContains,
		item.DescriptionContains,
		item.TagID,
	).Scan(
		&created.ID,
		&created.Name,
		&created.Enabled,
		&created.LanguageEquals,
		&created.OwnerContains,
		&created.NameContains,
		&created.DescriptionContains,
		&created.TagID,
		&created.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SmartRule{}, ErrNotFound
		}
		return SmartRule{}, fmt.Errorf("create smart rule: %w", err)
	}
	return created, nil
}

func (r *Repository) DeleteSmartRule(ctx context.Context, userID int64, ruleID int64) error {
	query := `DELETE FROM smart_rules WHERE user_id = $1 AND id = $2`
	result, err := r.store.Pool.Exec(ctx, query, userID, ruleID)
	if err != nil {
		return fmt.Errorf("delete smart rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ApplySmartRule(ctx context.Context, userID int64, rule SmartRule) (int64, error) {
	query := `
		INSERT INTO repository_tags (user_id, repository_id, tag_id)
		SELECT $1, s.repository_id, $2
		FROM stars s
		JOIN repositories r ON r.id = s.repository_id
		WHERE s.user_id = $1
		  AND ($3 = '' OR r.language = $3)
		  AND ($4 = '' OR r.owner_login ILIKE ('%' || $4 || '%'))
		  AND ($5 = '' OR r.name ILIKE ('%' || $5 || '%'))
		  AND ($6 = '' OR r.description ILIKE ('%' || $6 || '%'))
		ON CONFLICT (user_id, repository_id, tag_id) DO NOTHING`
	result, err := r.store.Pool.Exec(
		ctx,
		query,
		userID,
		rule.TagID,
		rule.LanguageEquals,
		rule.OwnerContains,
		rule.NameContains,
		rule.DescriptionContains,
	)
	if err != nil {
		return 0, fmt.Errorf("apply smart rule: %w", err)
	}
	return result.RowsAffected(), nil
}

func (r *Repository) applyEnabledSmartRulesWithScope(ctx context.Context, userID int64, repositoryIDs []int64) (int64, error) {
	whereRepositoryScope := ""
	args := []any{userID}
	if len(repositoryIDs) > 0 {
		whereRepositoryScope = " AND s.repository_id = ANY($2::bigint[])"
		args = append(args, repositoryIDs)
	}

	query := fmt.Sprintf(`
		INSERT INTO repository_tags (user_id, repository_id, tag_id)
		SELECT DISTINCT $1, s.repository_id, sr.tag_id
		FROM smart_rules sr
		JOIN stars s ON s.user_id = sr.user_id
		JOIN repositories r ON r.id = s.repository_id
		WHERE sr.user_id = $1
		  AND sr.enabled = true
		  AND (sr.language_equals = '' OR r.language = sr.language_equals)
		  AND (sr.owner_contains = '' OR r.owner_login ILIKE ('%%' || sr.owner_contains || '%%'))
		  AND (sr.name_contains = '' OR r.name ILIKE ('%%' || sr.name_contains || '%%'))
		  AND (sr.description_contains = '' OR r.description ILIKE ('%%' || sr.description_contains || '%%'))
		  %s
		ON CONFLICT (user_id, repository_id, tag_id) DO NOTHING`, whereRepositoryScope)
	result, err := r.store.Pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("apply enabled smart rules: %w", err)
	}
	return result.RowsAffected(), nil
}

func (r *Repository) ApplyEnabledSmartRules(ctx context.Context, userID int64) (int64, error) {
	return r.applyEnabledSmartRulesWithScope(ctx, userID, nil)
}

func (r *Repository) ApplyEnabledSmartRulesForRepositories(ctx context.Context, userID int64, repositoryIDs []int64) (int64, error) {
	if len(repositoryIDs) == 0 {
		return 0, nil
	}
	return r.applyEnabledSmartRulesWithScope(ctx, userID, repositoryIDs)
}

func (r *Repository) GetGovernanceMetrics(ctx context.Context, userID int64) (GovernanceMetrics, error) {
	query := `
		WITH total_stars AS (
			SELECT COUNT(*)::bigint AS cnt
			FROM stars s
			WHERE s.user_id = $1
		),
		untagged_stars AS (
			SELECT COUNT(*)::bigint AS cnt
			FROM stars s
			WHERE s.user_id = $1
			  AND NOT EXISTS (
				SELECT 1
				FROM repository_tags rt
				WHERE rt.user_id = s.user_id AND rt.repository_id = s.repository_id
			  )
		),
		sync_7d AS (
			SELECT
				COUNT(*)::bigint AS jobs,
				COUNT(*) FILTER (WHERE status = 'success')::bigint AS success
			FROM sync_jobs
			WHERE user_id = $1
			  AND started_at >= NOW() - INTERVAL '7 days'
		),
		stale_stars AS (
			SELECT COUNT(*)::bigint AS cnt
			FROM stars s
			JOIN repositories r ON r.id = s.repository_id
			WHERE s.user_id = $1
			  AND (r.pushed_at IS NULL OR r.pushed_at < NOW() - INTERVAL '180 days')
		)
		SELECT
			ts.cnt,
			us.cnt,
			s7.jobs,
			s7.success,
			ss.cnt
		FROM total_stars ts, untagged_stars us, sync_7d s7, stale_stars ss`

	var metrics GovernanceMetrics
	if err := r.store.Pool.QueryRow(ctx, query, userID).Scan(
		&metrics.TotalStars,
		&metrics.UntaggedStars,
		&metrics.SyncJobs7d,
		&metrics.SyncSuccess7d,
		&metrics.StaleStars,
	); err != nil {
		return GovernanceMetrics{}, fmt.Errorf("get governance metrics: %w", err)
	}
	if metrics.TotalStars > 0 {
		metrics.UntaggedRatio = float64(metrics.UntaggedStars) / float64(metrics.TotalStars)
	}
	if metrics.SyncJobs7d > 0 {
		metrics.SyncSuccessRate7d = float64(metrics.SyncSuccess7d) / float64(metrics.SyncJobs7d)
	}
	return metrics, nil
}

func (r *Repository) BuildExportPayload(ctx context.Context, userID int64) (ExportPayload, error) {
	syncSettings, err := r.GetSyncSettings(ctx, userID)
	if err != nil {
		return ExportPayload{}, fmt.Errorf("load sync settings: %w", err)
	}
	tags, err := r.ListTags(ctx, userID)
	if err != nil {
		return ExportPayload{}, fmt.Errorf("load tags: %w", err)
	}

	rulesQuery := `
		SELECT sr.name, sr.enabled, sr.language_equals, sr.owner_contains, sr.name_contains, sr.description_contains, t.name
		FROM smart_rules sr
		JOIN tags t ON t.id = sr.tag_id AND t.user_id = sr.user_id
		WHERE sr.user_id = $1
		ORDER BY sr.id ASC`
	ruleRows, err := r.store.Pool.Query(ctx, rulesQuery, userID)
	if err != nil {
		return ExportPayload{}, fmt.Errorf("load smart rules: %w", err)
	}
	defer ruleRows.Close()
	rules := make([]ExportRule, 0)
	for ruleRows.Next() {
		var item ExportRule
		if err := ruleRows.Scan(
			&item.Name,
			&item.Enabled,
			&item.LanguageEquals,
			&item.OwnerContains,
			&item.NameContains,
			&item.DescriptionContains,
			&item.TagName,
		); err != nil {
			return ExportPayload{}, fmt.Errorf("scan smart rule: %w", err)
		}
		rules = append(rules, item)
	}
	if err := ruleRows.Err(); err != nil {
		return ExportPayload{}, fmt.Errorf("iterate smart rules: %w", err)
	}

	notesQuery := `
		SELECT rep.github_repo_id, n.content
		FROM notes n
		JOIN repositories rep ON rep.id = n.repository_id
		WHERE n.user_id = $1
		ORDER BY rep.github_repo_id ASC`
	noteRows, err := r.store.Pool.Query(ctx, notesQuery, userID)
	if err != nil {
		return ExportPayload{}, fmt.Errorf("load notes: %w", err)
	}
	defer noteRows.Close()
	notes := make([]ExportNote, 0)
	for noteRows.Next() {
		var item ExportNote
		if err := noteRows.Scan(&item.GitHubRepoID, &item.Content); err != nil {
			return ExportPayload{}, fmt.Errorf("scan note: %w", err)
		}
		notes = append(notes, item)
	}
	if err := noteRows.Err(); err != nil {
		return ExportPayload{}, fmt.Errorf("iterate notes: %w", err)
	}

	bindingsQuery := `
		SELECT rep.github_repo_id, t.name
		FROM repository_tags rt
		JOIN repositories rep ON rep.id = rt.repository_id
		JOIN tags t ON t.id = rt.tag_id
		WHERE rt.user_id = $1
		  AND t.user_id = $1
		ORDER BY rep.github_repo_id ASC, t.name ASC`
	bindingRows, err := r.store.Pool.Query(ctx, bindingsQuery, userID)
	if err != nil {
		return ExportPayload{}, fmt.Errorf("load tag bindings: %w", err)
	}
	defer bindingRows.Close()
	bindings := make([]ExportTagBinding, 0)
	for bindingRows.Next() {
		var item ExportTagBinding
		if err := bindingRows.Scan(&item.GitHubRepoID, &item.TagName); err != nil {
			return ExportPayload{}, fmt.Errorf("scan tag binding: %w", err)
		}
		bindings = append(bindings, item)
	}
	if err := bindingRows.Err(); err != nil {
		return ExportPayload{}, fmt.Errorf("iterate tag bindings: %w", err)
	}

	payload := ExportPayload{
		Version:      "v1",
		ExportedAt:   time.Now().UTC(),
		SyncSettings: syncSettings,
		Tags:         tags,
		SmartRules:   rules,
		Notes:        notes,
		TagBindings:  bindings,
	}
	return payload, nil
}

func (r *Repository) ApplyImportPayload(ctx context.Context, userID int64, payload ImportPayload) (ImportResult, error) {
	tx, err := r.store.Pool.Begin(ctx)
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin import tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	result := ImportResult{}

	if payload.SyncSettings != nil {
		upsertSettings := `
			INSERT INTO user_sync_settings (user_id, enabled, interval_hours, retry_max, updated_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (user_id) DO UPDATE SET
				enabled = EXCLUDED.enabled,
				interval_hours = EXCLUDED.interval_hours,
				retry_max = EXCLUDED.retry_max,
				updated_at = NOW()`
		if _, err := tx.Exec(ctx, upsertSettings, userID, payload.SyncSettings.Enabled, payload.SyncSettings.IntervalHours, payload.SyncSettings.RetryMax); err != nil {
			return ImportResult{}, fmt.Errorf("import sync settings: %w", err)
		}
	}

	for _, tag := range payload.Tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		if len(name) > 50 {
			return ImportResult{}, fmt.Errorf("tag name too long")
		}
		if len(tag.Color) != 7 || tag.Color[0] != '#' {
			return ImportResult{}, fmt.Errorf("invalid tag color")
		}
		query := `
			INSERT INTO tags (user_id, name, color)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, name) DO UPDATE SET color = EXCLUDED.color`
		if _, err := tx.Exec(ctx, query, userID, name, tag.Color); err != nil {
			return ImportResult{}, fmt.Errorf("import tag %q: %w", name, err)
		}
		result.TagsUpserted++
	}

	noteGitHubRepoIDs := make([]int64, 0)
	noteGitHubRepoIDSet := make(map[int64]struct{})
	for _, note := range payload.Notes {
		if note.GitHubRepoID <= 0 {
			continue
		}
		if len(note.Content) > 5000 {
			return ImportResult{}, fmt.Errorf("note too long")
		}
		query := `
			INSERT INTO notes (user_id, repository_id, content, updated_at)
			SELECT $1, s.repository_id, $3, NOW()
			FROM stars s
			JOIN repositories r ON r.id = s.repository_id
			WHERE s.user_id = $1 AND r.github_repo_id = $2
			ON CONFLICT (user_id, repository_id) DO UPDATE SET
				content = EXCLUDED.content,
				updated_at = NOW()`
		execResult, err := tx.Exec(ctx, query, userID, note.GitHubRepoID, note.Content)
		if err != nil {
			return ImportResult{}, fmt.Errorf("import note repo=%d: %w", note.GitHubRepoID, err)
		}
		if execResult.RowsAffected() > 0 {
			result.NotesUpserted++
		}
		if _, exists := noteGitHubRepoIDSet[note.GitHubRepoID]; exists {
			continue
		}
		noteGitHubRepoIDSet[note.GitHubRepoID] = struct{}{}
		noteGitHubRepoIDs = append(noteGitHubRepoIDs, note.GitHubRepoID)
	}

	if len(noteGitHubRepoIDs) > 0 {
		repositoryRows, err := tx.Query(ctx, `
			SELECT s.repository_id
			FROM stars s
			JOIN repositories r ON r.id = s.repository_id
			WHERE s.user_id = $1 AND r.github_repo_id = ANY($2::bigint[])
		`, userID, noteGitHubRepoIDs)
		if err != nil {
			return ImportResult{}, fmt.Errorf("load note repository ids: %w", err)
		}
		repositoryIDs := make([]int64, 0)
		for repositoryRows.Next() {
			var repositoryID int64
			if err := repositoryRows.Scan(&repositoryID); err != nil {
				repositoryRows.Close()
				return ImportResult{}, fmt.Errorf("scan note repository id: %w", err)
			}
			repositoryIDs = append(repositoryIDs, repositoryID)
		}
		if err := repositoryRows.Err(); err != nil {
			repositoryRows.Close()
			return ImportResult{}, fmt.Errorf("iterate note repository ids: %w", err)
		}
		repositoryRows.Close()
		if err := r.upsertStarSearchDocsForRepositoryIDs(ctx, tx, userID, repositoryIDs); err != nil {
			return ImportResult{}, err
		}
	}

	for _, rule := range payload.SmartRules {
		tagName := strings.TrimSpace(rule.TagName)
		name := strings.TrimSpace(rule.Name)
		if tagName == "" || name == "" {
			continue
		}
		languageEquals := strings.TrimSpace(rule.LanguageEquals)
		ownerContains := strings.TrimSpace(rule.OwnerContains)
		nameContains := strings.TrimSpace(rule.NameContains)
		descriptionContains := strings.TrimSpace(rule.DescriptionContains)
		if len(name) > 80 {
			return ImportResult{}, fmt.Errorf("rule name too long")
		}
		if len(languageEquals) > 32 || len(ownerContains) > 100 || len(nameContains) > 120 || len(descriptionContains) > 200 {
			return ImportResult{}, fmt.Errorf("rule condition too long")
		}
		query := `
			INSERT INTO smart_rules (user_id, name, enabled, language_equals, owner_contains, name_contains, description_contains, tag_id)
			SELECT $1, $2, $3, $4, $5, $6, $7, t.id
			FROM tags t
			WHERE t.user_id = $1 AND t.name = $8`
		execResult, err := tx.Exec(ctx, query,
			userID,
			name,
			rule.Enabled,
			languageEquals,
			ownerContains,
			nameContains,
			descriptionContains,
			tagName,
		)
		if err != nil {
			return ImportResult{}, fmt.Errorf("import smart rule %q: %w", name, err)
		}
		if execResult.RowsAffected() > 0 {
			result.RulesUpserted++
		}
	}

	for _, binding := range payload.TagBindings {
		tagName := strings.TrimSpace(binding.TagName)
		if binding.GitHubRepoID <= 0 || tagName == "" {
			continue
		}
		query := `
			INSERT INTO repository_tags (user_id, repository_id, tag_id)
			SELECT $1, s.repository_id, t.id
			FROM stars s
			JOIN repositories r ON r.id = s.repository_id
			JOIN tags t ON t.user_id = $1 AND t.name = $3
			WHERE s.user_id = $1 AND r.github_repo_id = $2
			ON CONFLICT (user_id, repository_id, tag_id) DO NOTHING`
		execResult, err := tx.Exec(ctx, query, userID, binding.GitHubRepoID, tagName)
		if err != nil {
			return ImportResult{}, fmt.Errorf("import tag binding repo=%d tag=%q: %w", binding.GitHubRepoID, tagName, err)
		}
		if execResult.RowsAffected() > 0 {
			result.TagBindingsLinked++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ImportResult{}, fmt.Errorf("commit import tx: %w", err)
	}
	return result, nil
}

func (r *Repository) CreateSession(ctx context.Context, sessionID string, userID int64, tokenHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO sessions (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)`
	_, err := r.store.Pool.Exec(ctx, query, sessionID, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *Repository) ValidateSession(ctx context.Context, tokenHash string) (int64, error) {
	query := `
		SELECT user_id
		FROM sessions
		WHERE token_hash = $1 AND expires_at > NOW()
		LIMIT 1`
	var userID int64
	if err := r.store.Pool.QueryRow(ctx, query, tokenHash).Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("validate session: %w", err)
	}
	return userID, nil
}
