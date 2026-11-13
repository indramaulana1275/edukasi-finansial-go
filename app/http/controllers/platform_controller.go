package controllers

import (
	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

type PlatformController struct{}

func NewPlatformController() *PlatformController {
	return &PlatformController{}
}

// Index mengirim daftar platform untuk halaman Rekomendasi Platform. Endpoint
// ini publik karena halaman tersebut juga bisa dibuka tanpa login. Urutannya
// mengikuti id, yaitu urutan seed yang sama dengan daftar lama: crypto, saham,
// lalu komoditas, dengan platform lokal sebelum luar negeri.
func (r *PlatformController) Index(ctx http.Context) http.Response {
	rows := make([]models.InvestmentPlatform, 0)
	if err := facades.Orm().Query().Order("id asc").Get(&rows); err != nil {
		facades.Log().Errorf("gagal membaca daftar platform: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal membaca daftar platform",
		})
	}

	data := make([]http.Json, 0, len(rows))
	for _, row := range rows {
		data = append(data, platformPayload(row))
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":  data,
		"total": len(data),
	})
}

func platformPayload(platform models.InvestmentPlatform) http.Json {
	return http.Json{
		"id":          platform.ID,
		"name":        platform.Name,
		"category":    platform.Category,
		"type":        platform.Type,
		"logo_domain": platform.LogoDomain,
		"description": platform.Description,
		"pros":        platform.ProsList(),
		"cons":        platform.ConsList(),
	}
}
