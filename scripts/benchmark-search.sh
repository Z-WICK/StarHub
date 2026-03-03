#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/compose/docker-compose.yml"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/deploy/compose/.env.example}"
PG_CONTAINER="${PG_CONTAINER:-gsm-postgres}"
PG_USER="${PG_USER:-star_manager}"
PG_DB="${PG_DB:-star_manager}"
BENCH_USER_ID="${BENCH_USER_ID:-1}"
STAR_COUNT="${STAR_COUNT:-50000}"
KEYWORD="${KEYWORD:-awesome}"
PAGE_LIMIT="${PAGE_LIMIT:-20}"
PAGE_OFFSET="${PAGE_OFFSET:-0}"

OUT_DIR="$ROOT_DIR/.tmp"
OUT_FILE="$OUT_DIR/benchmark-search-$(date +%Y%m%d-%H%M%S).log"
mkdir -p "$OUT_DIR"

echo "[1/7] 启动 postgres 容器"
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d postgres >/dev/null

for _ in {1..60}; do
  if docker exec "$PG_CONTAINER" pg_isready -U "$PG_USER" -d "$PG_DB" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "[2/7] 重置 schema"
docker exec -i "$PG_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" >/dev/null

echo "[3/7] 应用迁移"
for migration in 0001_init.sql 0002_search_rules_ops.sql 0003_scheduler_rules.sql 0004_perf_indexes.sql 0005_search_index.sql 0006_search_backfill.sql; do
  docker cp "$ROOT_DIR/services/api/migrations/$migration" "$PG_CONTAINER:/tmp/$migration"
done
docker exec -i "$PG_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 \
  -f /tmp/0001_init.sql \
  -f /tmp/0002_search_rules_ops.sql \
  -f /tmp/0003_scheduler_rules.sql \
  -f /tmp/0004_perf_indexes.sql \
  -f /tmp/0005_search_index.sql >/dev/null

echo "[4/7] 生成基准数据 (STAR_COUNT=$STAR_COUNT)"
docker exec -i "$PG_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -c "
INSERT INTO users (id, display_name) VALUES (${BENCH_USER_ID}, 'bench-user');
SELECT setval(pg_get_serial_sequence('users','id'), (SELECT MAX(id) FROM users));

INSERT INTO repositories (id, github_repo_id, owner_login, name, full_name, private, html_url, description, language, stargazers_count, pushed_at, updated_at)
SELECT
  g,
  1000000 + g,
  'owner' || (g % 1000),
  'repo-' || g,
  'owner' || (g % 1000) || '/repo-' || g,
  false,
  'https://example.com/' || g,
  CASE WHEN g % 20 = 0 THEN 'awesome cache search benchmark term with note signal' ELSE 'regular repository description for benchmark' END,
  CASE WHEN g % 4 = 0 THEN 'Go' WHEN g % 4 = 1 THEN 'TypeScript' WHEN g % 4 = 2 THEN 'Python' ELSE 'Rust' END,
  (random()*50000)::int,
  NOW() - ((g % 365) || ' days')::interval,
  NOW()
FROM generate_series(1, ${STAR_COUNT}) AS g;
SELECT setval(pg_get_serial_sequence('repositories','id'), (SELECT MAX(id) FROM repositories));

INSERT INTO stars (user_id, repository_id, starred_at, last_seen_at)
SELECT ${BENCH_USER_ID}, g, NOW() - ((g % 700) || ' hours')::interval, NOW()
FROM generate_series(1, ${STAR_COUNT}) AS g;

INSERT INTO notes (user_id, repository_id, content, updated_at)
SELECT
  ${BENCH_USER_ID},
  g,
  CASE WHEN g % 15 = 0 THEN 'note includes awesome benchmark keyword for search' ELSE 'note text' END,
  NOW()
FROM generate_series(1, ${STAR_COUNT}) AS g
WHERE g % 3 = 0;
" >/dev/null

echo "[5/7] 回填 star_search_docs + ANALYZE"
docker exec -i "$PG_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -f /tmp/0006_search_backfill.sql >/dev/null
docker exec -i "$PG_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -c "ANALYZE;" >/dev/null

echo "[6/7] 执行 EXPLAIN ANALYZE"
docker cp "$ROOT_DIR/scripts/benchmark-search.sql" "$PG_CONTAINER:/tmp/benchmark-search.sql"
docker exec -i "$PG_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 \
  -v user_id="$BENCH_USER_ID" \
  -v keyword="$KEYWORD" \
  -v page_limit="$PAGE_LIMIT" \
  -v page_offset="$PAGE_OFFSET" \
  -f /tmp/benchmark-search.sql | tee "$OUT_FILE"

echo "[7/7] 完成"
echo "输出文件: $OUT_FILE"
echo "建议关注三段 Execution Time，并比较 old(count+list) vs new(single query)。"
