package services

import (
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestItemWasteGrams covers the single-source-of-truth waste helper the pricing engine
// and the stock aggregation both rely on.
func TestItemWasteGrams(t *testing.T) {
	t.Run("single filament has no color change and no waste", func(t *testing.T) {
		require.Zero(t, ItemWasteGrams(&CostPresetInput{WasteGramsPerColorChange: 20}, 1))
		require.Zero(t, ItemWasteGrams(nil, 1))
		require.Zero(t, ItemWasteGrams(nil, 0))
	})

	t.Run("uses the preset value for N-1 changes", func(t *testing.T) {
		// 3 filaments => 2 color changes => 20g * 2 = 40g.
		require.InDelta(t, 40.0, ItemWasteGrams(&CostPresetInput{WasteGramsPerColorChange: 20}, 3), 1e-9)
	})

	t.Run("falls back to the 15g default when preset is zero or absent", func(t *testing.T) {
		require.InDelta(t, 15.0, ItemWasteGrams(&CostPresetInput{WasteGramsPerColorChange: 0}, 2), 1e-9)
		require.InDelta(t, 15.0, ItemWasteGrams(nil, 2), 1e-9)
		// negative is treated as absent too.
		require.InDelta(t, 15.0, ItemWasteGrams(&CostPresetInput{WasteGramsPerColorChange: -5}, 2), 1e-9)
	})
}

// TestFilamentRequirements proves the pure aggregation: quantity plus the equal per-row
// share of each multi-filament item's purge waste, summed per filament across items.
func TestFilamentRequirements(t *testing.T) {
	t.Run("single-filament item has no waste", func(t *testing.T) {
		f := uuid.New()
		req := FilamentRequirements([]RequirementItem{
			{WasteGramsPerChange: 20, Filaments: []RequirementFilament{{FilamentID: f, Quantity: 100}}},
		})
		require.InDelta(t, 100.0, req[f], 1e-9)
		require.Zero(t, FilamentWasteGrams([]RequirementItem{
			{WasteGramsPerChange: 20, Filaments: []RequirementFilament{{FilamentID: f, Quantity: 100}}},
		})[f])
	})

	t.Run("3-filament item with preset 20g adds 40g waste split 13.33g per row", func(t *testing.T) {
		a, b, c := uuid.New(), uuid.New(), uuid.New()
		items := []RequirementItem{{
			WasteGramsPerChange: 20,
			Filaments: []RequirementFilament{
				{FilamentID: a, Quantity: 100},
				{FilamentID: b, Quantity: 200},
				{FilamentID: c, Quantity: 300},
			},
		}}
		req := FilamentRequirements(items)
		waste := FilamentWasteGrams(items)
		const share = 40.0 / 3.0 // 13.333...
		require.InDelta(t, share, waste[a], 1e-9)
		require.InDelta(t, 100+share, req[a], 1e-9)
		require.InDelta(t, 200+share, req[b], 1e-9)
		require.InDelta(t, 300+share, req[c], 1e-9)
		// The shares reconstitute the full 40g item waste.
		require.InDelta(t, 40.0, waste[a]+waste[b]+waste[c], 1e-9)
	})

	t.Run("preset <= 0 uses the 15g default", func(t *testing.T) {
		a, b := uuid.New(), uuid.New()
		items := []RequirementItem{{
			WasteGramsPerChange: 0,
			Filaments: []RequirementFilament{
				{FilamentID: a, Quantity: 100},
				{FilamentID: b, Quantity: 100},
			},
		}}
		req := FilamentRequirements(items)
		// 2 filaments => 1 change => 15g total => 7.5g per row.
		require.InDelta(t, 107.5, req[a], 1e-9)
		require.InDelta(t, 107.5, req[b], 1e-9)
	})

	t.Run("the same filament across items sums quantity and waste", func(t *testing.T) {
		shared := uuid.New()
		other := uuid.New()
		items := []RequirementItem{
			{
				WasteGramsPerChange: 20, // 2 filaments => 20g => 10g per row
				Filaments: []RequirementFilament{
					{FilamentID: shared, Quantity: 100},
					{FilamentID: other, Quantity: 50},
				},
			},
			{
				WasteGramsPerChange: 0, // default 15g, 2 filaments => 7.5g per row
				Filaments: []RequirementFilament{
					{FilamentID: shared, Quantity: 200},
					{FilamentID: other, Quantity: 50},
				},
			},
		}
		req := FilamentRequirements(items)
		// shared: 100 + 10 (item1) + 200 + 7.5 (item2) = 317.5
		require.InDelta(t, 317.5, req[shared], 1e-9)
		// other: 50 + 10 + 50 + 7.5 = 117.5
		require.InDelta(t, 117.5, req[other], 1e-9)
		// rounding once per filament matches the repository policy.
		require.Equal(t, int64(318), int64(math.Round(req[shared])))
	})
}
