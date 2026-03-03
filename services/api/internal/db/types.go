package db

import "time"

type UserProfile struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"displayName"`
	GitHubLogin string `json:"githubLogin"`
	AvatarURL   string `json:"avatarUrl"`
}

type Tag struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type StarRecord struct {
	RepositoryID    int64      `json:"repositoryId"`
	GitHubRepoID    int64      `json:"githubRepoId"`
	OwnerLogin      string     `json:"ownerLogin"`
	Name            string     `json:"name"`
	FullName        string     `json:"fullName"`
	Private         bool       `json:"private"`
	HTMLURL         string     `json:"htmlUrl"`
	Description     string     `json:"description"`
	Language        string     `json:"language"`
	StargazersCount int        `json:"stargazersCount"`
	PushedAt        *time.Time `json:"pushedAt"`
	StarredAt       time.Time  `json:"starredAt"`
	LastSeenAt      time.Time  `json:"lastSeenAt"`
	Note            string     `json:"note"`
	Tags            []Tag      `json:"tags"`
}

type StarFilters struct {
	Query     string
	Language  string
	TagID     int64
	HasNote   *bool
	SortBy    string
	SortOrder string
	Page      int
	Limit     int
}

type RepositoryRef struct {
	RepositoryID int64  `json:"repositoryId"`
	OwnerLogin   string `json:"ownerLogin"`
	Name         string `json:"name"`
	FullName     string `json:"fullName"`
}

type SyncJob struct {
	ID           int64      `json:"id"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
	Cursor       string     `json:"cursor"`
	ErrorMessage string     `json:"errorMessage"`
}

type SyncSettings struct {
	Enabled       bool      `json:"enabled"`
	IntervalHours int       `json:"intervalHours"`
	RetryMax      int       `json:"retryMax"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type SmartRule struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	Enabled             bool      `json:"enabled"`
	LanguageEquals      string    `json:"languageEquals"`
	OwnerContains       string    `json:"ownerContains"`
	NameContains        string    `json:"nameContains"`
	DescriptionContains string    `json:"descriptionContains"`
	TagID               int64     `json:"tagId"`
	CreatedAt           time.Time `json:"createdAt"`
}

type GovernanceMetrics struct {
	TotalStars        int64   `json:"totalStars"`
	UntaggedStars     int64   `json:"untaggedStars"`
	UntaggedRatio     float64 `json:"untaggedRatio"`
	SyncJobs7d        int64   `json:"syncJobs7d"`
	SyncSuccess7d     int64   `json:"syncSuccess7d"`
	SyncSuccessRate7d float64 `json:"syncSuccessRate7d"`
	StaleStars        int64   `json:"staleStars"`
}

type ExportRule struct {
	Name                string `json:"name"`
	Enabled             bool   `json:"enabled"`
	LanguageEquals      string `json:"languageEquals"`
	OwnerContains       string `json:"ownerContains"`
	NameContains        string `json:"nameContains"`
	DescriptionContains string `json:"descriptionContains"`
	TagName             string `json:"tagName"`
}

type ExportNote struct {
	GitHubRepoID int64  `json:"githubRepoId"`
	Content      string `json:"content"`
}

type ExportTagBinding struct {
	GitHubRepoID int64  `json:"githubRepoId"`
	TagName      string `json:"tagName"`
}

type ExportPayload struct {
	Version      string             `json:"version"`
	ExportedAt   time.Time          `json:"exportedAt"`
	SyncSettings SyncSettings       `json:"syncSettings"`
	Tags         []Tag              `json:"tags"`
	SmartRules   []ExportRule       `json:"smartRules"`
	Notes        []ExportNote       `json:"notes"`
	TagBindings  []ExportTagBinding `json:"tagBindings"`
}

type ImportPayload struct {
	SyncSettings *SyncSettings      `json:"syncSettings"`
	Tags         []Tag              `json:"tags"`
	SmartRules   []ExportRule       `json:"smartRules"`
	Notes        []ExportNote       `json:"notes"`
	TagBindings  []ExportTagBinding `json:"tagBindings"`
}

type ImportResult struct {
	TagsUpserted      int64 `json:"tagsUpserted"`
	RulesUpserted     int64 `json:"rulesUpserted"`
	NotesUpserted     int64 `json:"notesUpserted"`
	TagBindingsLinked int64 `json:"tagBindingsLinked"`
}
