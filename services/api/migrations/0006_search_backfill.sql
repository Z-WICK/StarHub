INSERT INTO star_search_docs (user_id, repository_id, doc, updated_at)
SELECT
  s.user_id,
  s.repository_id,
  to_tsvector('simple', COALESCE(r.full_name, '') || ' ' || COALESCE(r.description, '') || ' ' || COALESCE(n.content, '')),
  NOW()
FROM stars s
JOIN repositories r ON r.id = s.repository_id
LEFT JOIN notes n ON n.user_id = s.user_id AND n.repository_id = s.repository_id
ON CONFLICT (user_id, repository_id) DO UPDATE SET
  doc = EXCLUDED.doc,
  updated_at = NOW();
