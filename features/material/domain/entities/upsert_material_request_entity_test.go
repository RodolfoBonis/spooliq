package entities

import (
	"encoding/json"
	"testing"
)

// TestUpsertMaterialRequestUnmarshal verifies the request accepts both the
// canonical snake_case keys and the legacy camelCase keys, that snake_case wins
// when both are present, and that a legitimate 0 is preserved.
func TestUpsertMaterialRequestUnmarshal(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		wantTable        float32
		wantExtruder     float32
		wantName         string
		wantUnmarshalErr bool
	}{
		{
			name:         "snake_case keys",
			body:         `{"name":"PLA","temp_table":60,"temp_extruder":210}`,
			wantName:     "PLA",
			wantTable:    60,
			wantExtruder: 210,
		},
		{
			name:         "legacy camelCase keys",
			body:         `{"name":"PETG","tempTable":80,"tempExtruder":240}`,
			wantName:     "PETG",
			wantTable:    80,
			wantExtruder: 240,
		},
		{
			name:         "snake_case wins over camelCase",
			body:         `{"name":"ABS","temp_table":100,"tempTable":55,"temp_extruder":250,"tempExtruder":230}`,
			wantName:     "ABS",
			wantTable:    100,
			wantExtruder: 250,
		},
		{
			name:         "zero is preserved, not dropped",
			body:         `{"name":"Flex","temp_table":0,"temp_extruder":0}`,
			wantName:     "Flex",
			wantTable:    0,
			wantExtruder: 0,
		},
		{
			name:         "missing temps default to zero",
			body:         `{"name":"Nylon"}`,
			wantName:     "Nylon",
			wantTable:    0,
			wantExtruder: 0,
		},
		{
			name:             "invalid json",
			body:             `{"name":`,
			wantUnmarshalErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req UpsertMaterialRequestEntity
			err := json.Unmarshal([]byte(tt.body), &req)
			if tt.wantUnmarshalErr {
				if err == nil {
					t.Fatalf("expected unmarshal error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if req.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", req.Name, tt.wantName)
			}
			if req.TempTable != tt.wantTable {
				t.Errorf("TempTable = %v, want %v", req.TempTable, tt.wantTable)
			}
			if req.TempExtruder != tt.wantExtruder {
				t.Errorf("TempExtruder = %v, want %v", req.TempExtruder, tt.wantExtruder)
			}
		})
	}
}

// TestMaterialEntityTempFieldsNotOmitted verifies the entity serializes the
// temperature fields even when zero (no omitempty) under the snake_case keys.
func TestMaterialEntityTempFieldsNotOmitted(t *testing.T) {
	data, err := json.Marshal(MaterialEntity{Name: "PLA"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := out["temp_table"]; !ok {
		t.Error("temp_table should be present even when zero")
	}
	if _, ok := out["temp_extruder"]; !ok {
		t.Error("temp_extruder should be present even when zero")
	}
}
