package bootstrap

import (
	contractsfoundation "github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/foundation"

	"edukasi-finansial-goedukasi-finansial-go/config"
	"edukasi-finansial-goedukasi-finansial-go/routes"
)

func Boot() contractsfoundation.Application {
	return foundation.Setup().
		WithSeeders(Seeders).
		WithMigrations(Migrations).
		WithRouting(func() {
			routes.Web()
			routes.Api()
			routes.Grpc()
		}).
		WithProviders(Providers).
		WithConfig(config.Boot).
		Create()
}
