CREATE INDEX IF NOT EXISTS idx_sync_jobs_user_started_at ON sync_jobs(user_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_stars_user_starred_at ON stars(user_id, starred_at DESC);

CREATE INDEX IF NOT EXISTS idx_stars_user_repo_starred_at ON stars(user_id, repository_id, starred_at DESC);
