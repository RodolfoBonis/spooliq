-- Migration: add_filament_metadata_fields
-- Description: Add color_hex, price_per_meter and url fields to filaments table
-- Generated: 2025-09-23

-- Add color_hex column to filaments table
ALTER TABLE filaments ADD COLUMN IF NOT EXISTS color_hex VARCHAR(7);

-- Add price_per_meter column (if missing)
ALTER TABLE filaments ADD COLUMN IF NOT EXISTS price_per_meter DECIMAL(10,4);

-- Add URL column
ALTER TABLE filaments ADD COLUMN IF NOT EXISTS url TEXT;