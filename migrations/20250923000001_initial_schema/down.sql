-- Rollback Migration: initial_schema
-- Description: Drop initial database schema
-- Generated: 2025-09-23

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS quote_filament_lines CASCADE;
DROP TABLE IF EXISTS quotes CASCADE;
DROP TABLE IF EXISTS margin_profiles CASCADE;
DROP TABLE IF EXISTS cost_profiles CASCADE;
DROP TABLE IF EXISTS energy_profiles CASCADE;
DROP TABLE IF EXISTS machine_profiles CASCADE;
DROP TABLE IF EXISTS presets CASCADE;
DROP TABLE IF EXISTS filaments CASCADE;