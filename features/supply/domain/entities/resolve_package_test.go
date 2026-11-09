package entities

import "testing"

func ptrF(v float64) *float64 { return &v }
func ptrI(v int64) *int64     { return &v }

func TestResolvePackageUpdate(t *testing.T) {
	// Stored: lotion 400 ml for R$ 69,90 (derived unit_cost = 17).
	const storedSize, storedPrice = 400.0, int64(6990)
	cases := []struct {
		name      string
		req       UpsertSupplyRequestEntity
		wantSize  float64
		wantPrice int64
	}{
		{"nothing about price keeps the package", UpsertSupplyRequestEntity{}, 400, 6990},
		{"only size keeps the stored price", UpsertSupplyRequestEntity{PackageSize: ptrF(500)}, 500, 6990},
		{"only price keeps the stored size", UpsertSupplyRequestEntity{PackagePrice: ptrI(8000)}, 400, 8000},
		{"both package fields replace", UpsertSupplyRequestEntity{PackageSize: ptrF(200), PackagePrice: ptrI(3990)}, 200, 3990},
		{"legacy client echoing the derived unit_cost keeps the package", UpsertSupplyRequestEntity{UnitCost: ptrI(17)}, 400, 6990},
		{"legacy client changing unit_cost becomes a 1-unit package", UpsertSupplyRequestEntity{UnitCost: ptrI(25)}, 1, 25},
		{"package fields win over unit_cost", UpsertSupplyRequestEntity{PackagePrice: ptrI(7500), UnitCost: ptrI(1)}, 400, 7500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			size, price := tc.req.ResolvePackageUpdate(storedSize, storedPrice)
			if size != tc.wantSize || price != tc.wantPrice {
				t.Fatalf("got (%v, %d), want (%v, %d)", size, price, tc.wantSize, tc.wantPrice)
			}
		})
	}
}

func TestResolvePackageCreate(t *testing.T) {
	r := UpsertSupplyRequestEntity{UnitCost: ptrI(150)}
	if size, price := r.ResolvePackage(); size != 1 || price != 150 {
		t.Fatalf("legacy create: got (%v, %d)", size, price)
	}
	r = UpsertSupplyRequestEntity{PackageSize: ptrF(400), PackagePrice: ptrI(6990)}
	if size, price := r.ResolvePackage(); size != 400 || price != 6990 {
		t.Fatalf("package create: got (%v, %d)", size, price)
	}
	if size, price := (&UpsertSupplyRequestEntity{}).ResolvePackage(); size != 1 || price != 0 {
		t.Fatalf("empty create: got (%v, %d)", size, price)
	}
}
