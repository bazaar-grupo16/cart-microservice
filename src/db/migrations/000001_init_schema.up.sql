-- 000001_init_schema.up.sql

CREATE TABLE products (
    id              SERIAL PRIMARY KEY,
    seller_id       INTEGER NOT NULL CHECK (seller_id > 0),
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    price           NUMERIC(15, 2) NOT NULL CHECK (price > 0),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE cart_items (
    id              SERIAL PRIMARY KEY,
    user_id         INTEGER NOT NULL,
    product_id      INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    title           VARCHAR(255) NOT NULL,
    unit_price      NUMERIC(15, 2) NOT NULL CHECK (unit_price > 0),
    added_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);