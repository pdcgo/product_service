-- +goose Up
-- +goose StatementBegin
-- Legacy-compat: the `categories` table predates this service (see docs/database-schema.md).
-- Create it ONLY if it does not already exist so a shared/production DB is never altered.
CREATE TABLE IF NOT EXISTS categories (
    id        BIGSERIAL   PRIMARY KEY,
    parent_id BIGINT,
    level     INTEGER     NOT NULL DEFAULT 0,
    name      TEXT        NOT NULL,
    is_last   BOOLEAN     NOT NULL DEFAULT false,
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id) REFERENCES categories (id)
);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories (parent_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- No-op: `categories` is a shared legacy table; never drop it on a down migration.
SELECT 1;
-- +goose StatementEnd
