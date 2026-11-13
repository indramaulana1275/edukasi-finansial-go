package controllers

import (
	"time"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/services"
)

type InsightController struct {
	ai      *services.AIService
	insight *services.InsightService
}

func NewInsightController(ai *services.AIService, insight *services.InsightService) *InsightController {
	return &InsightController{ai: ai, insight: insight}
}

// Index menyajikan berita ekonomi global untuk halaman Insight Global, plus
// `source` yang menyebutkan dari mana daftarnya datang: "google_news" kalau feed
// sungguhan, "contoh" kalau yang dipakai daftar bawaan.
func (r *InsightController) Index(ctx http.Context) http.Response {
	items, source := r.insight.List(ctx.Request().Query("language", "id"))

	berita := make([]http.Json, 0, len(items))
	for _, item := range items {
		berita = append(berita, http.Json{
			"id":       item.ID,
			"title":    item.Title,
			"category": item.Category,
			"date":     item.Date,
			"summary":  item.Summary,
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":       berita,
		"source":     source,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// Explain menjawab tombol "Tanya FinGo AI" pada satu berita: dampaknya ke Emas,
// Bitcoin, Saham, dan USD.
//
// Body-nya {"id":"...","title":"..."}. Beritanya dicari di daftar yang sedang
// dipakai halaman, jadi yang sampai ke Gemini selalu judul dan ringkasan milik
// server — endpoint ini tidak bisa disuruh membahas topik lain.
func (r *InsightController) Explain(ctx http.Context) http.Response {
	if errors, ok := validate(ctx, map[string]any{"title": "required"}); !ok {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"errors": errors})
	}

	var input struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Language string `json:"language"`
	}
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{
			"error": "Format permintaan tidak dikenali",
		})
	}

	item, found := r.insight.Find(input.ID, input.Title, input.Language)
	if !found {
		return ctx.Response().Json(http.StatusNotFound, http.Json{
			"error": "Berita itu tidak ada di daftar Insight Global",
		})
	}

	explanation, err := r.ai.ExplainNews(item.ID, item.Title, item.Summary, input.Language)
	if err != nil {
		facades.Log().Errorf("gagal menjelaskan berita %q: %v", item.ID, err)

		return aiFailure(ctx, err)
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"id":          item.ID,
		"title":       item.Title,
		"explanation": explanation,
	})
}
