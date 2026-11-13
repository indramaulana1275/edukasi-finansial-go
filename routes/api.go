package routes

import (
	"github.com/goravel/framework/contracts/route"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/http/controllers"
	"edukasi-finansial-goedukasi-finansial-go/app/http/middleware"
	"edukasi-finansial-goedukasi-finansial-go/app/services"
)

func Api() {
	authController := &controllers.AuthController{}
	marketService := services.NewMarketService()
	aiService := services.NewAIService()
	insightService := services.NewInsightService()
	calendarService := services.NewCalendarService()
	worldService := services.NewWorldAssetService()
	miningService := services.NewBitcoinMiningService()
	marketController := controllers.NewMarketController(marketService, worldService, miningService)
	chatController := controllers.NewChatController(marketService, aiService, insightService)
	aiController := controllers.NewAIController(aiService, marketService, insightService)
	insightController := controllers.NewInsightController(aiService, insightService)
	calendarController := controllers.NewCalendarController(calendarService, aiService)
	riskController := controllers.NewRiskController()
	portfolioController := controllers.NewPortfolioController()
	educationController := controllers.NewEducationController()
	platformController := controllers.NewPlatformController()
	profileController := controllers.NewProfileController()
	adminController := controllers.NewAdminController()
	adminContentController := controllers.NewAdminContentController()
	adminOpsController := controllers.NewAdminOpsController()
	nlgController := controllers.NewNlgController()

	facades.Route().Prefix("/api").Group(func(router route.Router) {
		// Public routes
		router.Post("/register", authController.Register)
		router.Post("/login", authController.Login)
		router.Post("/logout", authController.Logout)
		// Halaman Rekomendasi Platform juga terbuka untuk tamu, jadi daftarnya
		// tidak boleh berada di balik AuthMiddleware.
		router.Get("/platforms", platformController.Index)
		router.Get("/settings", adminOpsController.SiteSettings)

		// Protected routes
		router.Middleware(middleware.NewAuthMiddleware()).Group(func(protected route.Router) {
			protected.Get("/me", authController.Me)
			protected.Put("/profile", profileController.Update)
			protected.Get("/market", marketController.Index)
			protected.Get("/market/history", marketController.History)
			protected.Get("/market/candlestick", marketController.Candlestick)
			protected.Get("/crypto", marketController.Crypto)
			protected.Get("/forex", marketController.Forex)
			protected.Get("/saham", marketController.Saham)
			protected.Get("/komoditas", marketController.Komoditas)
			protected.Get("/world-assets", marketController.WorldAssets)
			protected.Get("/bitcoin/mining", marketController.BitcoinMining)
			protected.Post("/chat/send", chatController.SendMessage)
			protected.Get("/ai/summary", aiController.MarketSummary)
			protected.Post("/ai/analyze-asset", aiController.AnalyzeAsset)
			protected.Get("/insight", insightController.Index)
			protected.Post("/insight/explain", insightController.Explain)
			protected.Get("/calendar", calendarController.Index)
			protected.Post("/calendar/explain", calendarController.Explain)
			protected.Get("/risk/profile", riskController.Profile)
			protected.Post("/risk/quiz", riskController.SubmitQuiz)
			protected.Get("/portfolio", portfolioController.Index)
			protected.Post("/portfolio", portfolioController.Store)
			protected.Delete("/portfolio/{id}", portfolioController.Destroy)
			protected.Get("/education", educationController.Index)
			protected.Get("/education/{id}", educationController.Show)
			protected.Get("/education/{id}/quiz", educationController.Quiz)
			protected.Post("/education/{id}/quiz/submit", educationController.QuizSubmit)
			protected.Post("/education/{id}/complete", educationController.Complete)
			protected.Post("/feedback", adminOpsController.SubmitFeedback)
		})

		// Admin-only NLG evaluation endpoints. Kept outside the /api/admin
		// prefix because the dashboard panel already calls /api/nlg-test.
		router.Middleware(middleware.NewAuthMiddleware(), middleware.NewRoleMiddleware("admin")).
			Group(func(adminOnly route.Router) {
				adminOnly.Get("/nlg-test", nlgController.Index)
				adminOnly.Post("/nlg-test", nlgController.Store)
			})

		// Admin routes
		router.Prefix("/admin").
			Middleware(middleware.NewAuthMiddleware(), middleware.NewRoleMiddleware("admin")).
			Group(func(admin route.Router) {
				admin.Get("/dashboard", adminController.Dashboard)
				admin.Get("/users", adminController.Users)
				admin.Patch("/users/{id}/role", adminController.UpdateRole)
				admin.Delete("/users/{id}", adminController.DeleteUser)

				admin.Get("/modules", adminContentController.Modules)
				admin.Post("/modules", adminContentController.StoreModule)
				admin.Put("/modules/{id}", adminContentController.UpdateModule)
				admin.Delete("/modules/{id}", adminContentController.DeleteModule)

				admin.Get("/quizzes", adminContentController.Quizzes)
				admin.Post("/quizzes", adminContentController.StoreQuiz)
				admin.Put("/quizzes/{id}", adminContentController.UpdateQuiz)
				admin.Delete("/quizzes/{id}", adminContentController.DeleteQuiz)

				admin.Get("/platforms", adminContentController.Platforms)
				admin.Post("/platforms", adminContentController.StorePlatform)
				admin.Put("/platforms/{id}", adminContentController.UpdatePlatform)
				admin.Delete("/platforms/{id}", adminContentController.DeletePlatform)

				admin.Get("/settings", adminOpsController.SiteSettings)
				admin.Put("/settings", adminOpsController.UpdateSettings)
				admin.Get("/activity", adminOpsController.Activity)
				admin.Get("/feedback", adminOpsController.Feedback)
				admin.Patch("/feedback/{id}", adminOpsController.UpdateFeedback)
				admin.Delete("/feedback/{id}", adminOpsController.DeleteFeedback)
			})
	})
}
