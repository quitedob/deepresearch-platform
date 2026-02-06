-- AI Research Platform Database Initialization Script
-- This script runs automatically when the PostgreSQL container starts for the first time

-- Create database if it doesn't exist (handled by POSTGRES_DB env var)
-- CREATE DATABASE IF NOT EXISTS ai_research_platform;

-- Connect to the database
\c ai_research_platform;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For text search optimization

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE ai_research_platform TO postgres;

-- Create schema version table for migrations
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Log initialization
INSERT INTO schema_migrations (version) VALUES ('init') ON CONFLICT DO NOTHING;

-- Success message
SELECT 'Database initialized successfully' AS status;
