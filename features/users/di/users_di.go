package di

import (
	"github.com/RodolfoBonis/spooliq/features/users"
	"github.com/RodolfoBonis/spooliq/features/users/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/users/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// UsersModule provides the fx module for users dependencies.
var UsersModule = fx.Module("users",
	fx.Provide(
		// Repository
		func(db *gorm.DB) domainRepositories.UserRepository {
			return repositories.NewUserRepository(db)
		},

		// Use Cases
		usecases.NewCreateUserUseCase,
		usecases.NewListUsersUseCase,
		usecases.NewFindUserUseCase,
		usecases.NewUpdateUserUseCase,
		usecases.NewDeleteUserUseCase,

		// Handler
		func(
			createUserUC *usecases.CreateUserUseCase,
			listUsersUC *usecases.ListUsersUseCase,
			findUserUC *usecases.FindUserUseCase,
			updateUserUC *usecases.UpdateUserUseCase,
			deleteUserUC *usecases.DeleteUserUseCase,
		) *users.Handler {
			return users.NewUserHandler(createUserUC, listUsersUC, findUserUC, updateUserUC, deleteUserUC)
		},
	),
)
