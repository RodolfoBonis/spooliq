package di

import (
	"go.uber.org/fx"

	"github.com/RodolfoBonis/spooliq/features/export/domain/services"
)

// Module provides the export feature module
var Module = fx.Module("export",
	fx.Provide(
		services.NewExportService,
	),
)
