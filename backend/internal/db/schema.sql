-- Idempotent schema: applied on every startup.

CREATE TABLE IF NOT EXISTS products (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    category    TEXT NOT NULL DEFAULT 'General',
    price       DOUBLE PRECISION NOT NULL DEFAULT 0,
    mrp         DOUBLE PRECISION NOT NULL DEFAULT 0,
    unit        TEXT NOT NULL DEFAULT 'pcs',
    aliases     JSONB NOT NULL DEFAULT '[]',
    photo_url   TEXT,
    emoji       TEXT NOT NULL DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_products_active ON products (active, sort_order, name);

CREATE TABLE IF NOT EXISTS sales (
    id          BIGSERIAL PRIMARY KEY,
    client_id   TEXT NOT NULL,
    product_id  BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    qty         INT NOT NULL CHECK (qty > 0),
    unit_price  DOUBLE PRECISION NOT NULL,
    total       DOUBLE PRECISION NOT NULL,
    source      TEXT NOT NULL DEFAULT 'tap',
    sold_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sales_sold_at ON sales (sold_at DESC);
CREATE INDEX IF NOT EXISTS idx_sales_product_time ON sales (product_id, sold_at DESC);
CREATE INDEX IF NOT EXISTS idx_sales_client ON sales (client_id);

CREATE TABLE IF NOT EXISTS reports (
    id          BIGSERIAL PRIMARY KEY,
    kind        TEXT NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,
    lang        TEXT NOT NULL DEFAULT 'en',
    content     TEXT NOT NULL,
    meta        JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_reports_kind_time ON reports (kind, created_at DESC);

CREATE TABLE IF NOT EXISTS job_runs (
    job_name   TEXT PRIMARY KEY,
    last_run   TIMESTAMPTZ,
    last_ok    BOOLEAN NOT NULL DEFAULT FALSE,
    last_error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
