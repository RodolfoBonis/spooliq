-- Migration: add_deleted_at_to_quote_filament_lines
-- Description: Add deleted_at column to quote_filament_lines for soft delete support
-- Generated: 2025-09-23 11:28:20

-- Add deleted_at column to quote_filament_lines table
ALTER TABLE quote_filament_lines ADD COLUMN deleted_at TIMESTAMP;

-- Create index on deleted_at for better query performance
CREATE INDEX IF NOT EXISTS idx_quote_filament_lines_deleted_at ON quote_filament_lines(deleted_at);