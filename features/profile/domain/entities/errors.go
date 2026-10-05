package entities

import "errors"

var (
	// ErrProfileNotFound is returned when a profile does not exist within the organization scope.
	ErrProfileNotFound = errors.New("profile not found")
	// ErrCannotDeleteDefaultProfile is returned when attempting to delete the default profile.
	ErrCannotDeleteDefaultProfile = errors.New("default profiles cannot be deleted")
	// ErrInvalidMachinePreset is returned when the referenced machine preset is missing or of the wrong type.
	ErrInvalidMachinePreset = errors.New("machine_preset_id must reference a machine preset in your organization")
	// ErrInvalidEnergyPreset is returned when the referenced energy preset is missing or of the wrong type.
	ErrInvalidEnergyPreset = errors.New("energy_preset_id must reference an energy preset in your organization")
	// ErrInvalidCostPreset is returned when the referenced cost preset is missing or of the wrong type.
	ErrInvalidCostPreset = errors.New("cost_preset_id must reference a cost preset in your organization")
	// ErrDefaultConflict is returned when a concurrent mutation would create a
	// second default profile for the same organization.
	ErrDefaultConflict = errors.New("já existe um perfil padrão para esta organização; tente novamente")
)

// Fallback names used when auto-generating a profile name with incomplete data.
const (
	// FallbackMachinePart is used when the machine preset name is unavailable.
	FallbackMachinePart = "Máquina"
	// FallbackEnergyPart is used when the energy preset name is unavailable.
	FallbackEnergyPart = "Energia"
	// FallbackProfileName is used when no referenced preset name is available.
	FallbackProfileName = "Perfil de impressão"
)

// GenerateProfileName builds a profile name from the referenced machine and
// energy preset names: "{machine} · {energy}". Missing parts fall back to
// generic labels; when both are empty it returns FallbackProfileName.
func GenerateProfileName(machineName, energyName string) string {
	machine := machineName
	energy := energyName
	if machine == "" && energy == "" {
		return FallbackProfileName
	}
	if machine == "" {
		machine = FallbackMachinePart
	}
	if energy == "" {
		energy = FallbackEnergyPart
	}
	return machine + " · " + energy
}
