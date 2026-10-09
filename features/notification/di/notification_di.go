package di

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/notification/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/notification/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/notification/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides the notification repository and service.
var Module = fx.Module("notification", fx.Provide(
	func(db *gorm.DB) domainRepositories.NotificationRepository {
		return repositories.NewNotificationRepository(db)
	},
	func(repository domainRepositories.NotificationRepository, log logger.Logger) usecases.INotificationService {
		return usecases.NewNotificationService(repository, log)
	},
))
