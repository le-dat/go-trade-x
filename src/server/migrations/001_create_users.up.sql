-- Migration: Create users and balances tables
-- Up migration

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS balances (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    asset TEXT NOT NULL,
    available NUMERIC(20, 8) NOT NULL DEFAULT 0,
    locked NUMERIC(20, 8) NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, asset)
);

CREATE INDEX IF NOT EXISTS idx_balances_user_id ON balances(user_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);