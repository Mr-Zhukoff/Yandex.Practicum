CREATE TABLE IF NOT EXISTS products (
    product_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    category TEXT,
    brand TEXT,
    sku TEXT,
    store_id TEXT,
    price_amount NUMERIC(12, 2),
    price_currency TEXT,
    stock_available INTEGER,
    stock_reserved INTEGER,
    tags TEXT[],
    raw_payload JSONB NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    ingested_at TIMESTAMPTZ DEFAULT now(),
    search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('russian', coalesce(name, '') || ' ' || coalesce(description, '') || ' ' || coalesce(category, ''))
    ) STORED
);

CREATE INDEX IF NOT EXISTS idx_products_search_vector ON products USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
CREATE INDEX IF NOT EXISTS idx_products_tags ON products USING GIN(tags);

CREATE TABLE IF NOT EXISTS forbidden_products (
    product_id TEXT PRIMARY KEY,
    reason TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS recommendations (
    recommendation_key TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    user_id TEXT,
    payload JSONB NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT now()
);
