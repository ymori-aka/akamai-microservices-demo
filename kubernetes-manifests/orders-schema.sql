-- Phase 2: Order persistence schema for Linode Managed PostgreSQL
-- Idempotent (safe to re-run).

CREATE TABLE IF NOT EXISTS orders (
  order_id              UUID PRIMARY KEY,
  session_id            TEXT,
  email                 TEXT,
  user_currency         TEXT NOT NULL,
  shipping_tracking_id  TEXT NOT NULL,
  shipping_cost_currency TEXT,
  shipping_cost_units   BIGINT,
  shipping_cost_nanos   INT,
  shipping_street       TEXT,
  shipping_city         TEXT,
  shipping_state        TEXT,
  shipping_country      TEXT,
  shipping_zip_code     INT,
  total_currency        TEXT,
  total_units           BIGINT,
  total_nanos           INT,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_items (
  id                    BIGSERIAL PRIMARY KEY,
  order_id              UUID NOT NULL REFERENCES orders(order_id) ON DELETE CASCADE,
  product_id            TEXT NOT NULL,
  product_name          TEXT,
  quantity              INT NOT NULL,
  unit_price_currency   TEXT,
  unit_price_units      BIGINT,
  unit_price_nanos      INT
);

CREATE INDEX IF NOT EXISTS idx_orders_session_id ON orders(session_id);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
