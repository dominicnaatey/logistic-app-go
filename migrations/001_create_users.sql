-- Migration: 001_create_users
-- Description: Creates the users table for authentication and profile management
-- Phase: 1 - Authentication & User Management

-- Enable uuid-ossp extension for uuid_generate_v4() (belt-and-suspenders alongside Go-level UUIDs)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone       VARCHAR(20)  NOT NULL,
    role        VARCHAR(50)  NOT NULL,
    name        VARCHAR(255),
    language    VARCHAR(10)  NOT NULL DEFAULT 'en',
    kyc_status  VARCHAR(50)  NOT NULL DEFAULT 'pending',
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ  -- soft delete (NULL = not deleted)
);

-- Unique index on phone (only among non-deleted rows)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone
    ON users (phone)
    WHERE deleted_at IS NULL;

-- Index for fast lookup by role (admin dashboard queries)
CREATE INDEX IF NOT EXISTS idx_users_role
    ON users (role)
    WHERE deleted_at IS NULL;

-- Index for soft-delete queries (GORM always filters on deleted_at)
CREATE INDEX IF NOT EXISTS idx_users_deleted_at
    ON users (deleted_at);

-- Enforce valid roles at the database level
ALTER TABLE users
    ADD CONSTRAINT chk_users_role
    CHECK (role IN ('shipper', 'driver', 'fleet_admin', 'owner_operator', 'admin'));

-- Enforce valid KYC status values
ALTER TABLE users
    ADD CONSTRAINT chk_users_kyc_status
    CHECK (kyc_status IN ('pending', 'approved', 'rejected'));

-- Enforce valid language values
ALTER TABLE users
    ADD CONSTRAINT chk_users_language
    CHECK (language IN ('en', 'fr'));

-- Rollback (run this to undo this migration):
-- DROP TABLE IF EXISTS users CASCADE;
