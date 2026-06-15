#!/usr/bin/env sh
set -eu

MIGRATIONS_DIR="${MIGRATIONS_DIR:-/database/migrations}"

psql -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL

for file in "$MIGRATIONS_DIR"/*.sql; do
  [ -e "$file" ] || continue
  version="$(basename "$file")"
  applied="$(psql -v ON_ERROR_STOP=1 -Atqc "SELECT 1 FROM schema_migrations WHERE version = '$version'")"

  if [ "$applied" = "1" ]; then
    echo "Skipping $version"
    continue
  fi

  echo "Applying $version"
  {
    printf 'BEGIN;\n'
    cat "$file"
    printf "\nINSERT INTO schema_migrations (version) VALUES ('%s');\n" "$version"
    printf 'COMMIT;\n'
  } | psql -v ON_ERROR_STOP=1
done
