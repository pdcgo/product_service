-- +goose Up
-- +goose StatementBegin
-- Soft-delete support for CategoryDelete (see docs/test.md). Additive & legacy-safe:
-- add the column only if it does not already exist so a shared/production DB is never
-- broken. GORM treats a NULL deleted_at as "live"; legacy code that ignores the column
-- keeps working.
ALTER TABLE categories ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_categories_deleted_at ON categories (deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- This column is owned by this service (unlike the shared `categories` table itself),
-- so a real reverse is safe here.
DROP INDEX IF EXISTS idx_categories_deleted_at;
ALTER TABLE categories DROP COLUMN IF EXISTS deleted_at;
-- +goose StatementEnd
