-- Rollback Migration: add_deleted_at_to_quote_filament_lines
-- Description: Remove deleted_at column from quote_filament_lines
-- Generated: 2025-09-23 11:28:20

-- Drop index first
DROP INDEX IF EXISTS idx_quote_filament_lines_deleted_at;

-- Remove deleted_at column from quote_filament_lines table
ALTER TABLE quote_filament_lines DROP COLUMN IF EXISTS deleted_at;