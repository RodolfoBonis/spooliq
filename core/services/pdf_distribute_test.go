package services

import "testing"

// TestDistributeMarkup verifies that the per-item final costs always sum EXACTLY
// to the budget total, that the rounding remainder lands on the last item, and
// that the degenerate cases (no items, zero direct cost) are handled.
func TestDistributeMarkup(t *testing.T) {
	tests := []struct {
		name      string
		itemCosts []int64
		totalCost int64
		want      []int64
	}{
		{
			name:      "no items",
			itemCosts: nil,
			totalCost: 0,
			want:      []int64{},
		},
		{
			name:      "single item absorbs all markup",
			itemCosts: []int64{1000},
			totalCost: 1500,
			want:      []int64{1500},
		},
		{
			name:      "no markup leaves costs untouched",
			itemCosts: []int64{1000, 2000, 3000},
			totalCost: 6000,
			want:      []int64{1000, 2000, 3000},
		},
		{
			name:      "even split",
			itemCosts: []int64{1000, 1000},
			totalCost: 3000, // markup 1000 split 500/500
			want:      []int64{1500, 1500},
		},
		{
			name:      "remainder lands on last item",
			itemCosts: []int64{1, 1, 1},
			totalCost: 13, // markup 10 over 3 equal items: round(3.33)=3,3 -> last gets 4
			want:      []int64{4, 4, 5},
		},
		{
			name:      "proportional to direct cost",
			itemCosts: []int64{100, 300},
			totalCost: 800, // markup 400: 25%/75% -> 100,300
			want:      []int64{200, 600},
		},
		{
			name:      "zero direct cost puts markup on last item",
			itemCosts: []int64{0, 0},
			totalCost: 500,
			want:      []int64{0, 500},
		},
		{
			name:      "negative markup (total below subtotal) still reconciles",
			itemCosts: []int64{1000, 1000},
			totalCost: 1500, // markup -500 split -250/-250
			want:      []int64{750, 750},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := distributeMarkup(tt.itemCosts, tt.totalCost)

			if len(got) != len(tt.want) {
				t.Fatalf("length mismatch: got %v want %v", got, tt.want)
			}
			var sum int64
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %d want %d (full: %v)", i, got[i], tt.want[i], got)
				}
				sum += got[i]
			}
			// The core invariant: the distribution must always reconcile exactly
			// with the budget total (when there is at least one item).
			if len(got) > 0 && sum != tt.totalCost {
				t.Errorf("sum %d != totalCost %d", sum, tt.totalCost)
			}
		})
	}
}

// TestDistributeMarkupAlwaysSumsToTotal is a property-style check over a range of
// uneven item costs ensuring the exact-sum invariant holds regardless of rounding.
func TestDistributeMarkupAlwaysSumsToTotal(t *testing.T) {
	cases := [][]int64{
		{7, 11, 13, 17},
		{1, 2, 3, 4, 5, 6, 7},
		{999, 1, 1},
		{333, 333, 334},
	}
	markups := []int64{0, 1, 7, 100, 9999, -50}

	for _, costs := range cases {
		var subtotal int64
		for _, c := range costs {
			subtotal += c
		}
		for _, m := range markups {
			total := subtotal + m
			got := distributeMarkup(costs, total)
			var sum int64
			for _, v := range got {
				sum += v
			}
			if sum != total {
				t.Errorf("costs=%v total=%d: sum=%d (got %v)", costs, total, sum, got)
			}
		}
	}
}
