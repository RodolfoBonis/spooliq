-- Rollback Migration: add_filament_metadata_fields
-- Description: Remove color_hex, price_per_meter and url fields from filaments table
-- Generated: 2025-09-23

-- Remove color_hex column from filaments table
ALTER TABLE filaments DROP COLUMN IF EXISTS color_hex;

-- Remove price_per_meter column
ALTER TABLE filaments DROP COLUMN IF EXISTS price_per_meter;

-- Remove URL column
ALTER TABLE filaments DROP COLUMN IF EXISTS url;