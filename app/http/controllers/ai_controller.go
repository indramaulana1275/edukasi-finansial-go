package controllers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/services"
)

// Batas panjang deret harga yang diterima endpoint analisis. Frontend mengirim
// 30 hari terakhir; batas atasnya menjaga prompt tetap wajar kalau ada klien
// lain yang mengirim jauh lebih banyak.
const (
	analysisMinPoints = 5
	analysisMaxPoints = 400
)

type AIController struct {
	ai      *services.AIService
	market  *services.MarketService
	insight *services.InsightService
}

func NewAIController(ai *services.AIService, market *services.MarketService, insight *services.InsightService) *AIController {
	return &AIController{ai: ai, market: market, insight: insight}
}

// MarketSummary menyajikan narasi kartu "FINANIKA Financial Advisor" di Dasbor.
//
// Jawabannya dirakit dari dua sumber dengan umur berbeda: narasinya datang dari
// cache 30 menit (lihat AIService.MarketSummary), sedangkan harga acuannya
// diambil di luar cache supaya balasan tetap memuat angka pasar terbaru walau
// narasinya masih dari beberapa menit lalu. `cached` memberi tahu frontend
// apakah balasan ini memakai kuota Gemini atau tidak.
//
// Berita Insight Global ikut dikirim sebagai konteks supaya sarannya bisa
// menyebut penyebab pergerakan harga; daftarnya sendiri sudah di-cache satu jam di
// InsightService, jadi ini bukan panggilan jaringan tambahan tiap kali Dasbor dibuka.
func (r *AIController) MarketSummary(ctx http.Context) http.Response {
	marketData, err := r.market.GetCurrentPrices()
	if err != nil {
		return ctx.Response().Json(http.StatusBadGateway, http.Json{
			"error":  "Gagal mengambil data pasar",
			"detail": err.Error(),
		})
	}

	// Ringkasan Dasbor kini berfokus pada IHSG, emas, dan SBSN/Sukuk. Harga IHSG
	// diambil dari daftar saham yang sama dengan kartu Dasbor, sedangkan Bitcoin
	// dibuang supaya model tidak membahas Crypto.
	if saham, err := r.market.GetSaham(); err == nil {
		for _, row := range saham {
			if asset, ok := row["asset"].(string); ok && asset == "ihsg" {
				marketData["ihsg"] = map[string]interface{}{
					"price":          row["price"],
					"change_percent": row["change_percent"],
					"previous_close": row["previous_close"],
					"currency":       row["currency"],
				}
				break
			}
		}
	}
	delete(marketData, "bitcoin")

	language := ctx.Request().Query("language", "id")

	berita, _ := r.insight.List(language)

	summary, err := r.ai.MarketSummary(latestRiskProfile(currentUserID(ctx)), language, marketData, berita)
	if err != nil {
		facades.Log().Errorf("gagal menyusun ringkasan pasar: %v", err)

		return aiFailure(ctx, err)
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"summary":      summary.Summary,
		"profile":      summary.Profile,
		"generated_at": summary.GeneratedAt,
		"cached":       summary.Cached,
		"prices":       marketData,
	})
}

// AnalyzeAsset menganalisis pola harga satu aset lewat Gemini.
//
// Body: {"asset":"bitcoin","label":"Bitcoin","days":30,
// "history":[{"date":"2026-10-18","price":81100.1}, ...]}
//
// Deret harganya ikut dikirim frontend supaya yang dianalisis persis sama dengan
// yang digambar di grafik; server hanya memeriksa bentuknya, tidak mengambil ulang.
func (r *AIController) AnalyzeAsset(ctx http.Context) http.Response {
	if errors, ok := validate(ctx, map[string]any{"asset": "required"}); !ok {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"errors": errors})
	}

	var input struct {
		Asset   string                     `json:"asset"`
		Label   string                     `json:"label"`
		Days    int                        `json:"days"`
		History []services.AssetPricePoint `json:"history"`
	}
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{
			"error": "Format data analisis tidak dikenali",
		})
	}

	asset := strings.ToLower(strings.TrimSpace(input.Asset))
	label := strings.TrimSpace(input.Label)
	if label == "" {
		label = asset
	}

	points := cleanPricePoints(input.History)
	if len(points) < analysisMinPoints {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{
			"error": "Kirim minimal " + strconv.Itoa(analysisMinPoints) +
				" titik harga historis supaya polanya bisa dibaca",
		})
	}

	analysis, err := r.ai.AnalyzeAsset(label, asset, points)
	if err != nil {
		facades.Log().Errorf("gagal menganalisis %s: %v", asset, err)

		return aiFailure(ctx, err)
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"asset":    asset,
		"label":    label,
		"days":     input.Days,
		"points":   len(points),
		"analysis": analysis,
	})
}

// cleanPricePoints membuang titik yang harganya tidak masuk akal (nol, negatif,
// atau gagal dibaca dari JSON) lalu memotong deretnya di batas atas.
func cleanPricePoints(points []services.AssetPricePoint) []services.AssetPricePoint {
	cleaned := make([]services.AssetPricePoint, 0, len(points))
	for _, point := range points {
		if point.Price <= 0 {
			continue
		}

		cleaned = append(cleaned, point)
	}

	if len(cleaned) > analysisMaxPoints {
		cleaned = cleaned[len(cleaned)-analysisMaxPoints:]
	}

	return cleaned
}

// aiFailure memetakan kegagalan AIService jadi balasan yang bisa ditindaklanjuti
// user, bukan JSON mentah dari Google:
//
//   - beban tinggi (503)  -> 503, minta coba lagi sebentar
//   - kuota habis (429)   -> 429, minta tunggu satu menit
//   - sisanya             -> 502, sertakan detailnya untuk penelusuran
func aiFailure(ctx http.Context, err error) http.Response {
	return aiFailureWith(ctx, err, "FINANIKA sedang tidak bisa menjawab")
}

// aiFailureWith sama dengan aiFailure, tetapi bisa memakai kalimat pembuka
// sendiri untuk jalur yang berbeda — analisis gambar gagal bukan karena pasar,
// dan detail seperti "context deadline exceeded" tidak berarti apa pun bagi user.
func aiFailureWith(ctx http.Context, err error, message string) http.Response {
	switch {
	case errors.Is(err, services.ErrAIHighDemand):
		return ctx.Response().Json(http.StatusServiceUnavailable, http.Json{
			"error": "Sistem AI sedang sibuk karena banyak pengguna. Coba beberapa saat lagi ya.",
		})
	case errors.Is(err, services.ErrAIRateLimited):
		return ctx.Response().Json(http.StatusTooManyRequests, http.Json{
			"error": "Sistem AI sedang sibuk karena banyak permintaan. Mohon tunggu 1 menit lagi ya.",
		})
	default:
		return ctx.Response().Json(http.StatusBadGateway, http.Json{
			"error":  message,
			"detail": err.Error(),
		})
	}
}
