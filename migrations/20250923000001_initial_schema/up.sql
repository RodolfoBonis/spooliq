-- Migration: initial_schema
-- Description: Create initial database schema
-- Generated: 2025-09-23

-- Create Filaments table
CREATE TABLE IF NOT EXISTS filaments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    brand_id INTEGER NOT NULL,
    material_id INTEGER NOT NULL,
    color VARCHAR(100) NOT NULL,
    color_hex VARCHAR(7),
    color_type VARCHAR(20) DEFAULT 'solid',
    color_data TEXT,
    color_preview TEXT,
    diameter DECIMAL(3,2) NOT NULL,
    weight DECIMAL(8,2),
    price_per_kg DECIMAL(10,2) NOT NULL,
    price_per_meter DECIMAL(10,4),
    url TEXT,
    owner_user_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create Presets table
CREATE TABLE IF NOT EXISTS presets (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL UNIQUE,
    data TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create Machine Profiles table
CREATE TABLE IF NOT EXISTS machine_profiles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    brand VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL,
    watt DECIMAL(10,2) NOT NULL,
    idle_factor DECIMAL(3,2) DEFAULT 0,
    description TEXT,
    url TEXT,
    owner_user_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create Energy Profiles table
CREATE TABLE IF NOT EXISTS energy_profiles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    base_tariff DECIMAL(10,4) NOT NULL,
    flag_surcharge DECIMAL(10,4) DEFAULT 0,
    location VARCHAR(255),
    year INTEGER,
    description TEXT,
    owner_user_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create Cost Profiles table
CREATE TABLE IF NOT EXISTS cost_profiles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    wear_percentage DECIMAL(5,2) DEFAULT 0,
    overhead_amount DECIMAL(10,2) DEFAULT 0,
    wear_cost_per_hour DECIMAL(10,4) DEFAULT 0,
    maintenance_cost_per_hour DECIMAL(10,4) DEFAULT 0,
    overhead_cost_per_hour DECIMAL(10,4) DEFAULT 0,
    description TEXT,
    owner_user_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create Margin Profiles table
CREATE TABLE IF NOT EXISTS margin_profiles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    printing_only_margin DECIMAL(5,2) DEFAULT 0,
    printing_plus_margin DECIMAL(5,2) DEFAULT 0,
    full_service_margin DECIMAL(5,2) DEFAULT 0,
    operator_rate_per_hour DECIMAL(10,2) DEFAULT 0,
    modeler_rate_per_hour DECIMAL(10,2) DEFAULT 0,
    labor_cost_per_hour DECIMAL(10,2) DEFAULT 0,
    profit_margin DECIMAL(5,2) DEFAULT 0,
    description TEXT,
    owner_user_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create Quotes table
CREATE TABLE IF NOT EXISTS quotes (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    notes TEXT,
    owner_user_id VARCHAR(255) NOT NULL,
    total_print_time INTEGER DEFAULT 0,
    total_filament_g DECIMAL(10,2) DEFAULT 0,
    total_cost DECIMAL(10,2) DEFAULT 0,
    machine_profile_id INTEGER,
    energy_profile_id INTEGER,
    cost_profile_id INTEGER,
    margin_profile_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create Quote Filament Lines table
CREATE TABLE IF NOT EXISTS quote_filament_lines (
    id SERIAL PRIMARY KEY,
    quote_id INTEGER NOT NULL,
    filament_id INTEGER NOT NULL,
    weight_grams DECIMAL(10,2) NOT NULL,
    length_meters DECIMAL(10,2),
    print_time_seconds INTEGER DEFAULT 0,
    cost DECIMAL(10,2) DEFAULT 0,
    notes TEXT,
    filament_snapshot_name VARCHAR(255),
    filament_snapshot_brand VARCHAR(255),
    filament_snapshot_material VARCHAR(100),
    filament_snapshot_color VARCHAR(100),
    filament_snapshot_color_hex VARCHAR(7),
    filament_snapshot_price_per_kg DECIMAL(10,2),
    filament_snapshot_price_per_meter DECIMAL(10,4),
    filament_snapshot_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_filaments_brand_id ON filaments(brand_id);
CREATE INDEX IF NOT EXISTS idx_filaments_material_id ON filaments(material_id);
CREATE INDEX IF NOT EXISTS idx_filaments_owner_user_id ON filaments(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_filaments_color_type ON filaments(color_type);
CREATE INDEX IF NOT EXISTS idx_machine_profiles_owner_user_id ON machine_profiles(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_energy_profiles_owner_user_id ON energy_profiles(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_cost_profiles_owner_user_id ON cost_profiles(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_margin_profiles_owner_user_id ON margin_profiles(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_quotes_owner_user_id ON quotes(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_quotes_machine_profile_id ON quotes(machine_profile_id);
CREATE INDEX IF NOT EXISTS idx_quotes_energy_profile_id ON quotes(energy_profile_id);
CREATE INDEX IF NOT EXISTS idx_quotes_cost_profile_id ON quotes(cost_profile_id);
CREATE INDEX IF NOT EXISTS idx_quotes_margin_profile_id ON quotes(margin_profile_id);
CREATE INDEX IF NOT EXISTS idx_quote_filament_lines_quote_id ON quote_filament_lines(quote_id);
CREATE INDEX IF NOT EXISTS idx_quote_filament_lines_filament_id ON quote_filament_lines(filament_id);

-- Create FilamentBrand table
CREATE TABLE IF NOT EXISTS filament_brands (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create FilamentMaterial table
CREATE TABLE IF NOT EXISTS filament_materials (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    properties TEXT,
    active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create brand and material indexes
CREATE INDEX IF NOT EXISTS idx_filament_brands_name ON filament_brands(name);
CREATE INDEX IF NOT EXISTS idx_filament_brands_active ON filament_brands(active);
CREATE INDEX IF NOT EXISTS idx_filament_materials_name ON filament_materials(name);
CREATE INDEX IF NOT EXISTS idx_filament_materials_active ON filament_materials(active);

-- Add foreign key constraints after all tables are created (idempotent)
-- PostgreSQL doesn't support IF NOT EXISTS for constraints directly, so we use a different approach

-- Add fk_filaments_brand constraint if it doesn't exist
ALTER TABLE filaments DROP CONSTRAINT IF EXISTS fk_filaments_brand;
ALTER TABLE filaments ADD CONSTRAINT fk_filaments_brand FOREIGN KEY (brand_id) REFERENCES filament_brands(id);

-- Add fk_filaments_material constraint if it doesn't exist
ALTER TABLE filaments DROP CONSTRAINT IF EXISTS fk_filaments_material;
ALTER TABLE filaments ADD CONSTRAINT fk_filaments_material FOREIGN KEY (material_id) REFERENCES filament_materials(id);

-- Add fk_quotes_machine_profile constraint if it doesn't exist
ALTER TABLE quotes DROP CONSTRAINT IF EXISTS fk_quotes_machine_profile;
ALTER TABLE quotes ADD CONSTRAINT fk_quotes_machine_profile FOREIGN KEY (machine_profile_id) REFERENCES machine_profiles(id);

-- Add fk_quotes_energy_profile constraint if it doesn't exist
ALTER TABLE quotes DROP CONSTRAINT IF EXISTS fk_quotes_energy_profile;
ALTER TABLE quotes ADD CONSTRAINT fk_quotes_energy_profile FOREIGN KEY (energy_profile_id) REFERENCES energy_profiles(id);

-- Add fk_quotes_cost_profile constraint if it doesn't exist
ALTER TABLE quotes DROP CONSTRAINT IF EXISTS fk_quotes_cost_profile;
ALTER TABLE quotes ADD CONSTRAINT fk_quotes_cost_profile FOREIGN KEY (cost_profile_id) REFERENCES cost_profiles(id);

-- Add fk_quotes_margin_profile constraint if it doesn't exist
ALTER TABLE quotes DROP CONSTRAINT IF EXISTS fk_quotes_margin_profile;
ALTER TABLE quotes ADD CONSTRAINT fk_quotes_margin_profile FOREIGN KEY (margin_profile_id) REFERENCES margin_profiles(id);

-- Add fk_quote_filament_lines_quote constraint if it doesn't exist
ALTER TABLE quote_filament_lines DROP CONSTRAINT IF EXISTS fk_quote_filament_lines_quote;
ALTER TABLE quote_filament_lines ADD CONSTRAINT fk_quote_filament_lines_quote FOREIGN KEY (quote_id) REFERENCES quotes(id) ON DELETE CASCADE;

-- Add fk_quote_filament_lines_filament constraint if it doesn't exist
ALTER TABLE quote_filament_lines DROP CONSTRAINT IF EXISTS fk_quote_filament_lines_filament;
ALTER TABLE quote_filament_lines ADD CONSTRAINT fk_quote_filament_lines_filament FOREIGN KEY (filament_id) REFERENCES filaments(id);