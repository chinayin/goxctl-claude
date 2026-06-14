---
inclusion: fileMatch
fileMatchPattern: "{migrations/**,**/dbmigrate/**,**/store.go,**/*migrate*,**/*migration*,**/*schema*}"
---

# Database Migration Standards (goose)

Use [pressly/goose v3](https://github.com/pressly/goose) exclusively for schema evolution.
Never use GORM AutoMigrate, raw DDL, or other migration tools.

## Rules When Generating Migration Files

1. Multi-table changes must be split into separate migration files, one concern per file
2. CREATE uses `IF NOT EXISTS`, DROP uses `IF EXISTS`
3. New columns must have a DEFAULT value or allow NULL
4. Never drop columns, change types, or rename directly -- use expand/contract pattern
5. Every file must contain `-- +goose Up` and `-- +goose Down`
6. Down must be the exact inverse of Up
7. After generation, verify against the checklist at the end of this document

## File Naming and Structure

- Format: `YYYYMMDDHHMMSS_<description>.sql`
- Generate with `goose create`, never number manually
- Description in snake_case: `add_product_tags`, `create_orders_table`

```sql
-- +goose Up
<up statements>

-- +goose Down
<down statements>
```

- Annotations must not have leading spaces, capitalize first letter: `Up`, `Down`

### Advanced Annotations

| annotation | purpose |
|---|---|
| `-- +goose StatementBegin` / `-- +goose StatementEnd` | Wrap complex statements (stored procedures, bulk INSERT) |
| `-- +goose NO TRANSACTION` | Statements that cannot run inside a transaction |
| `-- +goose ENVSUB ON` / `-- +goose ENVSUB OFF` | Environment variable substitution |

## expand/contract Pattern

Destructive changes require two separate migrations:

1. expand: add new column/table, code writes to both old and new
2. contract: after confirming old column has no references, drop in next release

```sql
-- First release: expand
-- +goose Up
ALTER TABLE users ADD COLUMN display_name TEXT DEFAULT '';

-- Second release: contract (after confirming code no longer reads old column)
-- +goose Up
-- SQLite requires table rebuild, MySQL can DROP COLUMN directly
```

## SQLite Limitations

ALTER TABLE has limited capabilities. These operations require table rebuild:

- Changing column type
- Changing column constraints (NOT NULL / NULL)
- Changing DEFAULT value

### Table Rebuild Pattern

```sql
-- +goose Up
CREATE TABLE users_new (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL,
    name  TEXT NOT NULL DEFAULT ''
);
INSERT INTO users_new (id, email, name) SELECT id, email, name FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- +goose Down
-- (reverse operation, also uses table rebuild pattern)
```

## MySQL 8 Limitations and Notes

### DDL Transaction Behavior
- DDL implicitly commits transactions, DDL rollback is not supported
- Each migration file should contain only one DDL statement when possible

### Large Table Migrations (millions of rows)
- Adding columns: prefer `ALGORITHM=INSTANT` (8.0.12+, lock-free)
- When INSTANT is not supported: use `ALGORITHM=INPLACE, LOCK=NONE`
- Adding indexes: use `ALGORITHM=INPLACE, LOCK=NONE`
- Very large tables: consider `gh-ost` or `pt-online-schema-change`

```sql
-- +goose Up
ALTER TABLE orders ADD COLUMN tags JSON DEFAULT NULL, ALGORITHM=INSTANT;
CREATE INDEX idx_orders_tags ON orders((CAST(tags->'$.type' AS CHAR(32) ARRAY)));
```

### Must Follow
- Table charset: `DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
- Never use `utf8` (only 3 bytes, incomplete)
- `CREATE INDEX IF NOT EXISTS` requires MySQL 8.0+
- Stored procedures require `-- +goose StatementBegin` / `-- +goose StatementEnd`

### Recommended
- Separate schema changes and data backfills into different migration files
- New table indexes go in the same migration as CREATE TABLE
- Indexes on existing large tables use `ALGORITHM=INPLACE, LOCK=NONE`

## goose CLI Commands

```bash
# Install
go install github.com/pressly/goose/v3/cmd/goose@latest

# Create migration
goose -dir migrations/main create <name> sql

# SQLite
goose -dir migrations/main sqlite3 <db_path> up
goose -dir migrations/main sqlite3 <db_path> down
goose -dir migrations/main sqlite3 <db_path> status

# MySQL
goose -dir migrations/main mysql "<user>:<password>@tcp(<host>:3306)/<dbname>?parseTime=true" up
goose -dir migrations/main mysql "<user>:<password>@tcp(<host>:3306)/<dbname>?parseTime=true" status

# Sharded databases (if applicable)
goose -dir migrations/<group> <driver> <dsn> up
```

## Project Structure

```
project/
├── migrations/
│   ├── main/                              # Main database migrations
│   │   ├── embed.go                       # embed.FS
│   │   └── YYYYMMDDHHMMSS_<name>.sql
│   └── history/                           # Sharded migrations (if applicable)
│       ├── embed.go
│       └── YYYYMMDDHHMMSS_<name>.sql
├── internal/
│   └── dbmigrate/
│       └── migrate.go                     # Thin wrapper
```

### embed.go Template

```go
// Package mainmig embeds main database migration files.
package mainmig

import "embed"

//go:embed *.sql
var FS embed.FS
```

### dbmigrate/migrate.go Template

```go
package dbmigrate

import (
    "context"
    "database/sql"
    "fmt"
    "io/fs"
    "log/slog"
    "time"

    "github.com/pressly/goose/v3"
)

// Run executes all pending migrations on the given database.
func Run(ctx context.Context, db *sql.DB, fsys fs.FS, dialect goose.Dialect, label string) error {
    provider, err := goose.NewProvider(dialect, db, fsys)
    if err != nil {
        return fmt.Errorf("dbmigrate: new provider [%s]: %w", label, err)
    }
    results, err := provider.Up(ctx)
    if err != nil {
        return fmt.Errorf("dbmigrate: up [%s]: %w", label, err)
    }
    for _, r := range results {
        if r.Error != nil {
            return fmt.Errorf("dbmigrate: [%s] version %d: %w", label, r.Source.Version, r.Error)
        }
        slog.Info("dbmigrate: applied",
            "target", label,
            "version", r.Source.Version,
            "duration", r.Duration.Round(time.Millisecond),
        )
    }
    return nil
}
```

### Integration Principles
- dbmigrate has zero business dependencies, accepts only `*sql.DB` + `fs.FS`
- Never import business packages in dbmigrate
- Never call goose API directly from business code, always use dbmigrate.Run

## Sharded Migrations (as needed)

- All shards share the same `migrations/<group>/` migration files
- New shards run all migrations on creation
- Existing shards apply pending migrations on next access

## Checklist

Verify after generating migration files:

- [ ] Generated with `goose create`, timestamp is correct
- [ ] Contains `-- +goose Up` and `-- +goose Down`
- [ ] CREATE/DROP uses `IF NOT EXISTS` / `IF EXISTS`
- [ ] New columns have DEFAULT or allow NULL
- [ ] No column drops, type changes, or renames (or expand/contract applied)
- [ ] Single responsibility, one concern per file
- [ ] Down is the exact inverse of Up
