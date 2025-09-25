-- Migration: add_calculation_fields_to_quotes
-- Description: Add calculation result fields to quotes table for storing calculated values
-- Generated: 2025-09-24

-- Add calculation result fields to quotes table
ALTER TABLE quotes ADD COLUMN calculation_material_cost DECIMAL(10,2);
ALTER TABLE quotes ADD COLUMN calculation_energy_cost DECIMAL(10,2);  
ALTER TABLE quotes ADD COLUMN calculation_wear_cost DECIMAL(10,2);
ALTER TABLE quotes ADD COLUMN calculation_labor_cost DECIMAL(10,2);
ALTER TABLE quotes ADD COLUMN calculation_direct_cost DECIMAL(10,2);
ALTER TABLE quotes ADD COLUMN calculation_final_price DECIMAL(10,2);
ALTER TABLE quotes ADD COLUMN calculation_print_time_hours DECIMAL(8,2);
ALTER TABLE quotes ADD COLUMN calculation_operator_minutes DECIMAL(8,2);
ALTER TABLE quotes ADD COLUMN calculation_modeler_minutes DECIMAL(8,2);
ALTER TABLE quotes ADD COLUMN calculation_service_type VARCHAR(50);
ALTER TABLE quotes ADD COLUMN calculation_applied_margin DECIMAL(5,2);
ALTER TABLE quotes ADD COLUMN calculation_calculated_at TIMESTAMP;

-- Create index for frequently queried calculated_at field
CREATE INDEX IF NOT EXISTS idx_quotes_calculation_calculated_at ON quotes(calculation_calculated_at);