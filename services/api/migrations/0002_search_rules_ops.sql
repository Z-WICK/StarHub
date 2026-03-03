CREATE INDEX IF NOT EXISTS idx_repositories_updated_at ON repositories(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_repositories_pushed_at ON repositories(pushed_at DESC);
CREATE INDEX IF NOT EXISTS idx_repositories_stargazers_count ON repositories(stargazers_count DESC);
CREATE INDEX IF NOT EXISTS idx_notes_user_repo ON notes(user_id, repository_id);
CREATE INDEX IF NOT EXISTS idx_stars_repo_user ON stars(repository_id, user_id);

CREATE INDEX IF NOT EXISTS idx_repositories_search_fts ON repositories USING GIN (
  to_tsvector('simple', COALESCE(full_name, '') || ' ' || COALESCE(description, ''))
);

CREATE INDEX IF NOT EXISTS idx_notes_search_fts ON notes USING GIN (
  to_tsvector('simple', COALESCE(content, ''))
);
