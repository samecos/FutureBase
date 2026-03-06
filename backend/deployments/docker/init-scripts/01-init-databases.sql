-- ==========================================
-- Initialize Architecture Platform Databases
-- ==========================================

-- Enable PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create schemas for different services
CREATE SCHEMA IF NOT EXISTS user_service;
CREATE SCHEMA IF NOT EXISTS project_service;
CREATE SCHEMA IF NOT EXISTS property_service;
CREATE SCHEMA IF NOT EXISTS version_service;
CREATE SCHEMA IF NOT EXISTS search_service;
CREATE SCHEMA IF NOT EXISTS geometry_service;

-- Grant permissions
GRANT USAGE ON SCHEMA user_service TO postgres;
GRANT USAGE ON SCHEMA project_service TO postgres;
GRANT USAGE ON SCHEMA property_service TO postgres;
GRANT USAGE ON SCHEMA version_service TO postgres;
GRANT USAGE ON SCHEMA search_service TO postgres;
GRANT USAGE ON SCHEMA geometry_service TO postgres;

COMMENT ON SCHEMA user_service IS 'User management and authentication';
COMMENT ON SCHEMA project_service IS 'Project and member management';
COMMENT ON SCHEMA property_service IS 'Property templates and calculations';
COMMENT ON SCHEMA version_service IS 'Version control and branching';
COMMENT ON SCHEMA search_service IS 'Search indices configuration';
COMMENT ON SCHEMA geometry_service IS 'Geometry data and spatial queries';
