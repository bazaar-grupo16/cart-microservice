-- 000001_create_cart_items.up.sql
-- Creates the cart_items table.
-- The product_id is intentionally NOT a foreign key to the products table —
-- this microservice owns only cart data. Product info (title, unit_price) is
-- stored as an immutable snapshot captured at the time the item was added.

CREATE TABLE IF NOT EXISTS cart_items (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER NOT NULL,
    product_id  INTEGER NOT NULL,
    title       VARCHAR(255) NOT NULL,
    unit_price  NUMERIC(15, 2) NOT NULL CHECK (unit_price > 0),
    added_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cart_items_user_id ON cart_items (user_id);