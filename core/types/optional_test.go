package types

import (
	"encoding/json"
	"testing"
	"time"
)

// wrapper mirrors how Optional is embedded in a request DTO: a present key invokes
// UnmarshalJSON, an absent key does not.
type wrapper struct {
	Value Optional[float64] `json:"value"`
}

// TestOptionalUnmarshal_Absent proves a missing key leaves the Optional unset, so the
// handler treats it as "leave stored value unchanged".
func TestOptionalUnmarshal_Absent(t *testing.T) {
	var w wrapper
	if err := json.Unmarshal([]byte(`{}`), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if w.Value.Set {
		t.Errorf("Set = true, want false for absent key")
	}
	if w.Value.IsClear() || w.Value.HasValue() {
		t.Errorf("absent key must be neither clear nor value-carrying: %+v", w.Value)
	}
}

// TestOptionalUnmarshal_Null proves an explicit JSON null marks the Optional as set and
// null, so the handler treats it as "clear".
func TestOptionalUnmarshal_Null(t *testing.T) {
	var w wrapper
	if err := json.Unmarshal([]byte(`{"value": null}`), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !w.Value.Set {
		t.Errorf("Set = false, want true for explicit null")
	}
	if !w.Value.Null {
		t.Errorf("Null = false, want true for explicit null")
	}
	if !w.Value.IsClear() {
		t.Errorf("IsClear() = false, want true for explicit null")
	}
	if w.Value.HasValue() {
		t.Errorf("HasValue() = true, want false for explicit null")
	}
}

// TestOptionalUnmarshal_Value proves a concrete value is decoded and flagged as present
// (set, not null).
func TestOptionalUnmarshal_Value(t *testing.T) {
	var w wrapper
	if err := json.Unmarshal([]byte(`{"value": 12.5}`), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !w.Value.Set || w.Value.Null {
		t.Errorf("want set && !null, got %+v", w.Value)
	}
	if !w.Value.HasValue() {
		t.Errorf("HasValue() = false, want true")
	}
	if w.Value.Value != 12.5 {
		t.Errorf("Value = %v, want 12.5", w.Value.Value)
	}
}

// TestOptionalUnmarshal_Time confirms the tri-state works for a struct inner type.
func TestOptionalUnmarshal_Time(t *testing.T) {
	type tw struct {
		At Optional[time.Time] `json:"at"`
	}
	var absent, null, val tw
	if err := json.Unmarshal([]byte(`{}`), &absent); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"at": null}`), &null); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"at": "2026-01-02T15:04:05Z"}`), &val); err != nil {
		t.Fatal(err)
	}
	if absent.At.Set {
		t.Error("absent: Set should be false")
	}
	if !null.At.IsClear() {
		t.Error("null: IsClear should be true")
	}
	if !val.At.HasValue() || val.At.Value.Year() != 2026 {
		t.Errorf("value: want 2026 time, got %+v", val.At)
	}
}

// TestOptionalMarshal confirms round-trip encoding: absent/null encode as null, a value
// encodes as the inner value.
func TestOptionalMarshal(t *testing.T) {
	cases := map[string]struct {
		in   Optional[float64]
		want string
	}{
		"unset": {Optional[float64]{}, "null"},
		"null":  {Optional[float64]{Set: true, Null: true}, "null"},
		"value": {Optional[float64]{Set: true, Value: 3.5}, "3.5"},
	}
	for name, tc := range cases {
		b, err := json.Marshal(tc.in)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		if string(b) != tc.want {
			t.Errorf("%s: got %s want %s", name, b, tc.want)
		}
	}
}
