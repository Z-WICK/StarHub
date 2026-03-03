CREATE TABLE IF NOT EXISTS user_sync_settings (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  interval_hours INTEGER NOT NULL DEFAULT 12,
  retry_max INTEGER NOT NULL DEFAULT 2,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS smart_rules (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  language_equals TEXT NOT NULL DEFAULT '',
  owner_contains TEXT NOT NULL DEFAULT '',
  name_contains TEXT NOT NULL DEFAULT '',
  description_contains TEXT NOT NULL DEFAULT '',
  tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_sync_settings_enabled ON user_sync_settings(enabled);
CREATE INDEX IF NOT EXISTS idx_smart_rules_user_id ON smart_rules(user_id);
