\timing on
\echo '=== OLD COUNT QUERY (runtime to_tsvector) ==='
EXPLAIN (ANALYZE, BUFFERS)
SELECT COUNT(*)
FROM stars s
JOIN repositories r ON r.id = s.repository_id
LEFT JOIN notes n ON n.user_id = s.user_id AND n.repository_id = s.repository_id
WHERE s.user_id = :user_id
  AND to_tsvector('simple', COALESCE(r.full_name,'') || ' ' || COALESCE(r.description,'') || ' ' || COALESCE(n.content,''))
      @@ plainto_tsquery('simple', :'keyword');

\echo '=== OLD LIST QUERY (runtime to_tsvector) ==='
EXPLAIN (ANALYZE, BUFFERS)
SELECT
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
WHERE s.user_id = :user_id
  AND to_tsvector('simple', COALESCE(r.full_name,'') || ' ' || COALESCE(r.description,'') || ' ' || COALESCE(n.content,''))
      @@ plainto_tsquery('simple', :'keyword')
ORDER BY s.starred_at DESC NULLS LAST, s.repository_id DESC
LIMIT :page_limit OFFSET :page_offset;

\echo '=== NEW LIST+TOTAL QUERY (star_search_docs + COUNT OVER) ==='
EXPLAIN (ANALYZE, BUFFERS)
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
JOIN star_search_docs sd
  ON sd.user_id = s.user_id
 AND sd.repository_id = s.repository_id
 AND sd.doc @@ plainto_tsquery('simple', :'keyword')
WHERE s.user_id = :user_id
ORDER BY s.starred_at DESC NULLS LAST, s.repository_id DESC
LIMIT :page_limit OFFSET :page_offset;
