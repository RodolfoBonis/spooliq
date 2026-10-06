package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

// ptr returns a pointer to the given UUID (test ergonomics).
func ptr(id uuid.UUID) *uuid.UUID { return &id }

// eqPtr reports whether two *uuid.UUID are equal (both nil, or both the same id).
func eqPtr(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// recordingValidator returns a validator that records every (id,type) it is asked
// to validate and, optionally, rejects a specific id to prove resolved IDs are
// validated against the organization.
func recordingValidator(reject *uuid.UUID, recorded *[]presetValidation) presetValidator {
	return func(_ context.Context, id uuid.UUID, presetType, _ string) error {
		if recorded != nil {
			*recorded = append(*recorded, presetValidation{id: id, presetType: presetType})
		}
		if reject != nil && *reject == id {
			return errors.New("preset not in organization")
		}
		return nil
	}
}

func TestPresetResolver_Precedence(t *testing.T) {
	explicitM, explicitE, explicitC := uuid.New(), uuid.New(), uuid.New()
	profM, profE, profC := uuid.New(), uuid.New(), uuid.New()
	defProfM, defProfE, defProfC := uuid.New(), uuid.New(), uuid.New()
	orgM, orgE, orgC := uuid.New(), uuid.New(), uuid.New()
	profileID := uuid.New()

	profile := &ProfilePresets{Name: "Prof", MachinePresetID: ptr(profM), EnergyPresetID: ptr(profE), CostPresetID: ptr(profC)}
	defProfile := &ProfilePresets{Name: "Default", MachinePresetID: ptr(defProfM), EnergyPresetID: ptr(defProfE), CostPresetID: ptr(defProfC)}
	orgDefaults := map[string]*uuid.UUID{"machine": ptr(orgM), "energy": ptr(orgE), "cost": ptr(orgC)}

	tests := []struct {
		name        string
		in          PresetResolutionInput
		profiles    *fakeProfileProvider
		presets     *fakeDefaultPresetProvider
		wantMachine *uuid.UUID
		wantEnergy  *uuid.UUID
		wantCost    *uuid.UUID
		wantProfile *uuid.UUID
		wantErr     error
	}{
		{
			name: "explicit request IDs win over everything",
			in:   PresetResolutionInput{ProfileID: ptr(profileID), MachinePresetID: ptr(explicitM), EnergyPresetID: ptr(explicitE), CostPresetID: ptr(explicitC)},
			profiles: &fakeProfileProvider{
				byID: map[uuid.UUID]*ProfilePresets{profileID: profile},
				def:  defProfile,
			},
			presets:     &fakeDefaultPresetProvider{byType: orgDefaults},
			wantMachine: ptr(explicitM), wantEnergy: ptr(explicitE), wantCost: ptr(explicitC), wantProfile: ptr(profileID),
		},
		{
			name: "requested profile presets win over defaults",
			in:   PresetResolutionInput{ProfileID: ptr(profileID)},
			profiles: &fakeProfileProvider{
				byID: map[uuid.UUID]*ProfilePresets{profileID: profile},
				def:  defProfile,
			},
			presets:     &fakeDefaultPresetProvider{byType: orgDefaults},
			wantMachine: ptr(profM), wantEnergy: ptr(profE), wantCost: ptr(profC), wantProfile: ptr(profileID),
		},
		{
			name:        "default profile presets win over org default presets",
			in:          PresetResolutionInput{},
			profiles:    &fakeProfileProvider{def: defProfile},
			presets:     &fakeDefaultPresetProvider{byType: orgDefaults},
			wantMachine: ptr(defProfM), wantEnergy: ptr(defProfE), wantCost: ptr(defProfC),
		},
		{
			name:        "org default presets are the last resort",
			in:          PresetResolutionInput{},
			profiles:    &fakeProfileProvider{},
			presets:     &fakeDefaultPresetProvider{byType: orgDefaults},
			wantMachine: ptr(orgM), wantEnergy: ptr(orgE), wantCost: ptr(orgC),
		},
		{
			name: "mixed precedence across slots",
			in:   PresetResolutionInput{ProfileID: ptr(profileID), MachinePresetID: ptr(explicitM)},
			profiles: &fakeProfileProvider{
				// requested profile only provides energy; cost comes from default profile.
				byID: map[uuid.UUID]*ProfilePresets{profileID: {Name: "P", EnergyPresetID: ptr(profE)}},
				def:  &ProfilePresets{Name: "D", CostPresetID: ptr(defProfC)},
			},
			presets:     &fakeDefaultPresetProvider{byType: orgDefaults},
			wantMachine: ptr(explicitM), wantEnergy: ptr(profE), wantCost: ptr(defProfC), wantProfile: ptr(profileID),
		},
		{
			name:        "no profile and no defaults resolves to nil (not an error)",
			in:          PresetResolutionInput{},
			profiles:    &fakeProfileProvider{},
			presets:     &fakeDefaultPresetProvider{},
			wantMachine: nil, wantEnergy: nil, wantCost: nil,
		},
		{
			name:     "profile from another organization yields ErrProfileNotFound",
			in:       PresetResolutionInput{ProfileID: ptr(profileID)},
			profiles: &fakeProfileProvider{missing: map[uuid.UUID]bool{profileID: true}},
			presets:  &fakeDefaultPresetProvider{},
			wantErr:  entities.ErrProfileNotFound,
		},
		{
			name:     "deleted profile yields ErrProfileNotFound",
			in:       PresetResolutionInput{ProfileID: ptr(profileID)},
			profiles: &fakeProfileProvider{missing: map[uuid.UUID]bool{profileID: true}},
			presets:  &fakeDefaultPresetProvider{},
			wantErr:  entities.ErrProfileNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newPresetResolver(tt.profiles, tt.presets, recordingValidator(nil, nil))
			got, err := r.Resolve(context.Background(), "org-a", tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !eqPtr(got.MachinePresetID, tt.wantMachine) {
				t.Errorf("machine = %v, want %v", got.MachinePresetID, tt.wantMachine)
			}
			if !eqPtr(got.EnergyPresetID, tt.wantEnergy) {
				t.Errorf("energy = %v, want %v", got.EnergyPresetID, tt.wantEnergy)
			}
			if !eqPtr(got.CostPresetID, tt.wantCost) {
				t.Errorf("cost = %v, want %v", got.CostPresetID, tt.wantCost)
			}
			if !eqPtr(got.ProfileID, tt.wantProfile) {
				t.Errorf("profile = %v, want %v", got.ProfileID, tt.wantProfile)
			}
		})
	}
}

// TestPresetResolver_ValidatesResolvedIDs proves every resolved (non-nil) preset is
// validated against the organization+type, and a rejected one fails resolution.
func TestPresetResolver_ValidatesResolvedIDs(t *testing.T) {
	m, e, cst := uuid.New(), uuid.New(), uuid.New()
	profiles := &fakeProfileProvider{}
	presets := &fakeDefaultPresetProvider{byType: map[string]*uuid.UUID{"machine": &m, "energy": &e, "cost": &cst}}

	var recorded []presetValidation
	r := newPresetResolver(profiles, presets, recordingValidator(nil, &recorded))
	if _, err := r.Resolve(context.Background(), "org-a", PresetResolutionInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[uuid.UUID]string{m: "machine", e: "energy", cst: "cost"}
	if len(recorded) != len(want) {
		t.Fatalf("expected %d validations, got %d: %+v", len(want), len(recorded), recorded)
	}
	for _, v := range recorded {
		if want[v.id] != v.presetType {
			t.Errorf("validated %s as %q, want %q", v.id, v.presetType, want[v.id])
		}
	}

	// A resolved preset that fails validation fails the whole resolution.
	r2 := newPresetResolver(profiles, presets, recordingValidator(&e, nil))
	if _, err := r2.Resolve(context.Background(), "org-a", PresetResolutionInput{}); err == nil {
		t.Fatal("expected validation error for a resolved preset outside the org, got nil")
	}
}
