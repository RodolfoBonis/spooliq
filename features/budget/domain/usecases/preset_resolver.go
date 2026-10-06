package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

// ProfilePresets holds the three preset IDs a print profile references. The
// machine and energy slots are always present on a profile; the cost slot is
// optional. This is a budget-local view of a profile, deliberately decoupled from
// the profile feature's own entities so the budget package does not import it.
type ProfilePresets struct {
	// Name is the profile's display name (used to build the response ref).
	Name            string
	MachinePresetID *uuid.UUID
	EnergyPresetID  *uuid.UUID
	CostPresetID    *uuid.UUID
}

// ProfilePresetProvider is the narrow view of the profile repository that the
// budget preset resolver needs. It is defined HERE (in the budget package) and
// implemented by an adapter wired in FX, so the dependency points budget -> profile
// without the budget domain importing the profile package (no import cycle).
type ProfilePresetProvider interface {
	// ProfileByID returns the preset IDs referenced by the profile, scoped to the
	// organization. It returns (nil, nil) when the profile does not exist within the
	// organization (another tenant's profile or a soft-deleted one), so the resolver
	// can surface that as a 400. A non-nil error signals an infrastructure failure.
	ProfileByID(ctx context.Context, id uuid.UUID, organizationID string) (*ProfilePresets, error)
	// DefaultProfile returns the preset IDs of the organization's default print
	// profile, or (nil, nil) when the organization has no default profile.
	DefaultProfile(ctx context.Context, organizationID string) (*ProfilePresets, error)
}

// DefaultPresetProvider is the narrow view of the preset repository that the
// resolver needs: the organization's default preset ID for a given type, or nil
// when the organization has no default preset of that type.
type DefaultPresetProvider interface {
	DefaultPresetID(ctx context.Context, organizationID string, presetType string) (*uuid.UUID, error)
}

// presetValidator validates that a preset belongs to the organization and has the
// expected type. It matches budgetRepo.BudgetRepository.ValidatePresetInOrg.
type presetValidator func(ctx context.Context, presetID uuid.UUID, presetType, organizationID string) error

// PresetResolutionInput carries the explicit references provided on a budget
// request. Any nil field triggers the fallback chain for that slot.
type PresetResolutionInput struct {
	ProfileID       *uuid.UUID
	MachinePresetID *uuid.UUID
	EnergyPresetID  *uuid.UUID
	CostPresetID    *uuid.UUID
}

// ResolvedPresets is the outcome of resolution: the profile that was used (if any)
// plus the resolved machine/energy/cost preset IDs. Any of them may be nil when no
// explicit value, profile value or org default exists for that slot.
type ResolvedPresets struct {
	ProfileID       *uuid.UUID
	MachinePresetID *uuid.UUID
	EnergyPresetID  *uuid.UUID
	CostPresetID    *uuid.UUID
}

// presetResolver resolves each preset slot of a budget following a fixed
// precedence, validating every resolved ID against the organization. It is a small
// pure-ish unit (its only collaborators are the two narrow providers and the
// validator) so it is directly table-testable.
type presetResolver struct {
	profiles ProfilePresetProvider
	presets  DefaultPresetProvider
	validate presetValidator
}

// newPresetResolver builds a resolver from its collaborators.
func newPresetResolver(profiles ProfilePresetProvider, presets DefaultPresetProvider, validate presetValidator) *presetResolver {
	return &presetResolver{profiles: profiles, presets: presets, validate: validate}
}

// Resolve resolves the machine, energy and cost preset for a budget. For each slot
// the first available source wins, in order:
//
//  1. the explicit ID provided in the request;
//  2. the preset referenced by the requested profile (profile_id);
//  3. the preset referenced by the organization's default profile;
//  4. the organization's default preset of that type.
//
// The requested profile is validated first (same org, not deleted); an unknown
// profile yields entities.ErrProfileNotFound. Every resolved (non-nil) ID is then
// validated with the org+type validator, so a reference pointing outside the org or
// of the wrong type is rejected. Missing defaults are fine — a slot simply resolves
// to nil. On success the resolved ProfileID echoes the requested one (nil when no
// profile was requested).
func (r *presetResolver) Resolve(ctx context.Context, organizationID string, in PresetResolutionInput) (ResolvedPresets, error) {
	var out ResolvedPresets
	out.ProfileID = in.ProfileID

	var requested *ProfilePresets
	if in.ProfileID != nil {
		p, err := r.profiles.ProfileByID(ctx, *in.ProfileID, organizationID)
		if err != nil {
			return ResolvedPresets{}, err
		}
		if p == nil {
			return ResolvedPresets{}, entities.ErrProfileNotFound
		}
		requested = p
	}

	def, err := r.profiles.DefaultProfile(ctx, organizationID)
	if err != nil {
		return ResolvedPresets{}, err
	}

	machine, err := r.resolveSlot(ctx, organizationID, "machine", in.MachinePresetID, requested, def, func(p *ProfilePresets) *uuid.UUID { return p.MachinePresetID })
	if err != nil {
		return ResolvedPresets{}, err
	}
	energy, err := r.resolveSlot(ctx, organizationID, "energy", in.EnergyPresetID, requested, def, func(p *ProfilePresets) *uuid.UUID { return p.EnergyPresetID })
	if err != nil {
		return ResolvedPresets{}, err
	}
	cost, err := r.resolveSlot(ctx, organizationID, "cost", in.CostPresetID, requested, def, func(p *ProfilePresets) *uuid.UUID { return p.CostPresetID })
	if err != nil {
		return ResolvedPresets{}, err
	}

	out.MachinePresetID = machine
	out.EnergyPresetID = energy
	out.CostPresetID = cost
	return out, nil
}

// resolveSlot applies the precedence chain to a single slot and validates the
// resolved ID (when any) against the organization and expected type.
func (r *presetResolver) resolveSlot(
	ctx context.Context,
	organizationID string,
	presetType string,
	explicit *uuid.UUID,
	requested *ProfilePresets,
	def *ProfilePresets,
	pick func(*ProfilePresets) *uuid.UUID,
) (*uuid.UUID, error) {
	var id *uuid.UUID
	switch {
	case explicit != nil:
		id = explicit
	case requested != nil && pick(requested) != nil:
		id = pick(requested)
	case def != nil && pick(def) != nil:
		id = pick(def)
	default:
		d, err := r.presets.DefaultPresetID(ctx, organizationID, presetType)
		if err != nil {
			return nil, err
		}
		id = d
	}

	if id != nil {
		if err := r.validate(ctx, *id, presetType, organizationID); err != nil {
			return nil, err
		}
	}
	return id, nil
}
