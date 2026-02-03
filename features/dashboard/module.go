package dashboard

import (
	"github.com/RodolfoBonis/spooliq/features/dashboard/data/repositories"
	domainRepos "github.com/RodolfoBonis/spooliq/features/dashboard/domain/repositories"
	"go.uber.org/fx"
)

// Module provides the dashboard feature dependencies.
var Module = fx.Module("dashboard",
	fx.Provide(
		fx.Annotate(
			repositories.NewDashboardRepository,
			fx.As(new(domainRepos.DashboardRepository)),
		),
	),
	fx.Provide(NewDashboardHandler),
)
