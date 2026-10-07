package entities

import "github.com/google/uuid"

// LowStockFilament is a single row of the dashboard low-stock widget: a tracked
// filament currently at or below its alert threshold.
type LowStockFilament struct {
	ID                     uuid.UUID `json:"id"`
	Name                   string    `json:"name"`
	Color                  string    `json:"color"`
	ColorHex               *string   `json:"color_hex,omitempty"`
	BrandName              *string   `json:"brand_name,omitempty"`
	MaterialName           *string   `json:"material_name,omitempty"`
	StockGrams             int64     `json:"stock_grams"`
	LowStockThresholdGrams int       `json:"low_stock_threshold_grams"`
}

// LowStockResponse is the envelope for the low-stock widget.
type LowStockResponse struct {
	Data []LowStockFilament `json:"data"`
}
