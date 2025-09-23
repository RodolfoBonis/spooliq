-- Migration: test_idempotent_constraints
-- Description: Test idempotent constraint creation to simulate production scenario
-- Generated: 2025-09-23 08:17:34

-- This migration tests if we can re-apply the same foreign key constraints
-- that already exist, simulating the production scenario

-- Re-apply the same constraints using the idempotent approach
ALTER TABLE filaments DROP CONSTRAINT IF EXISTS fk_filaments_brand;
ALTER TABLE filaments ADD CONSTRAINT fk_filaments_brand FOREIGN KEY (brand_id) REFERENCES filament_brands(id);

ALTER TABLE filaments DROP CONSTRAINT IF EXISTS fk_filaments_material;
ALTER TABLE filaments ADD CONSTRAINT fk_filaments_material FOREIGN KEY (material_id) REFERENCES filament_materials(id);

-- Test adding a new constraint that doesn't exist yet
ALTER TABLE filaments DROP CONSTRAINT IF EXISTS fk_filaments_test;
-- (We won't actually add this constraint in this test, just show the pattern)

