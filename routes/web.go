package routes

import (
	"github.com/goravel/framework/contracts/route"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/http/controllers"
	"edukasi-finansial-goedukasi-finansial-go/app/http/middleware"
)

func Web() {
	dashboardController := controllers.NewDashboardController()
	facades.Route().Get("/", dashboardController.Landing)
	facades.Route().Get("/dasbor", dashboardController.Dashboard)
	facades.Route().Get("/landing", dashboardController.Landing)
	facades.Route().Get("/platform", dashboardController.Platform)
	facades.Route().Get("/saham", dashboardController.Saham)
	facades.Route().Get("/sbsn", dashboardController.Sbsn)
	facades.Route().Get("/logam-mulia", dashboardController.LogamMulia)

	facades.Route().Middleware(middleware.NewAuthMiddleware()).Group(func(router route.Router) {
		router.Get("/pembelajaran", dashboardController.Pembelajaran)
		router.Get("/belajar", dashboardController.Belajar)
		router.Get("/kuis", dashboardController.Kuis)
		router.Get("/profil-risiko", dashboardController.ProfilRisiko)
		router.Get("/portofolio", dashboardController.Portofolio)
		router.Get("/asisten-ai", dashboardController.AsistenAI)
	})

	facades.Route().Middleware(
		middleware.NewAuthMiddleware(),
		middleware.NewRoleMiddleware("admin"),
	).Group(func(router route.Router) {
		router.Get("/admin", dashboardController.Admin)
		router.Get("/admin/{path}", dashboardController.Admin)
	})

	facades.Route().Static("public", "./public")

	userController := controllers.NewUserController()
	facades.Route().Get("/users", userController.Index)
}
