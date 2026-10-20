package app

import "github.com/MatiXxD/beerer-bot/internal/user/controller"

func UserDomain(di *DI) {
	// layers
	ctrl := controller.New()

	ctrl.RegisterRoutes(di.router)
}
