-- +goose Up
-- +goose StatementBegin
-- Legacy-compat: the `products` table predates this service (see docs/database-schema.md).
-- Create it ONLY if it does not already exist so a shared/production DB is never altered.
CREATE TABLE IF NOT EXISTS products (
    id             BIGSERIAL        PRIMARY KEY,
    team_id        BIGINT           NOT NULL,
    deleted        BOOLEAN          NOT NULL DEFAULT false,
    user_id        BIGINT,
    ref_id         TEXT,
    name           TEXT             NOT NULL,
    image          JSONB            NOT NULL DEFAULT '[]',
    "desc"         TEXT,
    markup_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
    priority       BOOLEAN          NOT NULL DEFAULT false,
    cross_locked   BOOLEAN          NOT NULL DEFAULT false,
    stock_reserved BIGINT           NOT NULL DEFAULT 0,
    created        TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_products_team FOREIGN KEY (team_id) REFERENCES teams (id),
    CONSTRAINT fk_products_user FOREIGN KEY (user_id) REFERENCES users (id)
);
CREATE INDEX IF NOT EXISTS idx_products_team_id ON products (team_id);
CREATE INDEX IF NOT EXISTS idx_products_deleted ON products (deleted);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- No-op: `products` is a shared legacy table; never drop it on a down migration.
SELECT 1;
-- +goose StatementEnd
