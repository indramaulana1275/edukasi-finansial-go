package controllers

import (
	"time"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

type DashboardController struct{}

func NewDashboardController() *DashboardController {
	return &DashboardController{}
}

func (r *DashboardController) Dashboard(ctx http.Context) http.Response {
	return ctx.Response().View().Make("dashboard.html", map[string]any{
		"appName": facades.Config().GetString("app.name", "FINANIKA"),
		"year":    time.Now().Format("2006"),
	})
}

func (r *DashboardController) Landing(ctx http.Context) http.Response {
	return ctx.Response().View().Make("landing.html", map[string]any{
		"year": time.Now().Format("2006"),
	})
}

// Platform mengantar pengunjung yang membuka alamat /platform ke panel
// "Rekomendasi Platform" di dalam dashboard. Aplikasi ini menyajikan satu view
// saja, jadi halamannya hidup sebagai panel dengan alamat berbasis hash
// (/dasbor#/platform) dan route ini hanya bertugas mengarahkan ke sana — tanpa
// menduplikasi sidebar, topbar, dan seluruh skrip dashboard.
func (r *DashboardController) Platform(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/platform")
}

// Saham, Sbsn, dan LogamMulia mengantar alamat path langsung ke panel hash
// yang sesuai. Halaman-halamannya tetap panel di dashboard.html, bukan view
// terpisah, supaya sidebar, topbar, dan seluruh skrip tidak diduplikasi.
func (r *DashboardController) Pembelajaran(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/belajar")
}

func (r *DashboardController) Belajar(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/belajar")
}

func (r *DashboardController) Kuis(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/kuis")
}

func (r *DashboardController) ProfilRisiko(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/kuis")
}

func (r *DashboardController) Portofolio(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/portfolio")
}

func (r *DashboardController) AsistenAI(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/asisten-ai")
}

func (r *DashboardController) Admin(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/admin")
}

func (r *DashboardController) Saham(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/saham")
}

func (r *DashboardController) Sbsn(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/sbsn")
}

func (r *DashboardController) LogamMulia(ctx http.Context) http.Response {
	return ctx.Response().Redirect(http.StatusFound, "/dasbor#/logam-mulia")
}
