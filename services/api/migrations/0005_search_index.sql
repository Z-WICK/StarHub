CREATE TABLE IF NOT EXISTS star_search_docs (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
  doc tsvector NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, repository_id)
);

CREATE INDEX IF NOT EXISTS idx_star_search_docs_doc ON star_search_docs USING GIN(doc);
CREATE INDEX IF NOT EXISTS idx_star_search_docs_user_repo ON star_search_docs(user_id, repository_id);
