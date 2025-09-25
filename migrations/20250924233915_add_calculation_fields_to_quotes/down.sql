-- Migration: add_calculation_fields_to_quotes (DOWN)
-- Description: Remove calculation result fields from quotes table
-- Generated: 2025-09-24

-- Remove index
DROP INDEX IF EXISTS idx_quotes_calculation_calculated_at;

-- Remove calculation result fields from quotes table
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_material_cost;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_energy_cost;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_wear_cost;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_labor_cost;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_direct_cost;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_final_price;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_print_time_hours;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_operator_minutes;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_modeler_minutes;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_service_type;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_applied_margin;
ALTER TABLE quotes DROP COLUMN IF EXISTS calculation_calculated_at;