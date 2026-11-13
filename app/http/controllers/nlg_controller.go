package controllers

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

// NlgController menyimpan hasil penilaian narasi dari panel admin. Endpoint-nya
// berada di balik AuthMiddleware + RoleMiddleware("admin").
type NlgController struct{}

func NewNlgController() *NlgController {
	return &NlgController{}
}

type nlgTestInput struct {
	Profile        string `json:"profile"`
	Topic          string `json:"topic"`
	Narrative      string `json:"narrative"`
	ScoreData      int    `json:"score_data"`
	ScoreContent   int    `json:"score_content"`
	ScoreRelevance int    `json:"score_relevance"`
	ScoreClarity   int    `json:"score_clarity"`
}

var nlgProfiles = map[string]bool{
	"konservatif": true,
	"moderat":     true,
	"agresif":     true,
}

var nlgTopics = map[string]bool{
	"saham":       true,
	"sbsn":        true,
	"emas":        true,
	"fundamental": true,
	"teknikal":    true,
}

func (r *NlgController) Index(ctx http.Context) http.Response {
	rows := make([]models.NlgTestResult, 0)
	if err := facades.Orm().Query().
		Model(&models.NlgTestResult{}).
		Order("id desc").
		Limit(100).
		Get(&rows); err != nil {
		facades.Log().Errorf("gagal membaca hasil pengujian NLG: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal membaca hasil pengujian NLG",
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":  rows,
		"total": len(rows),
	})
}

func (r *NlgController) Store(ctx http.Context) http.Response {
	var input nlgTestInput
	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Format penilaian tidak dikenali"})
	}

	profile := strings.ToLower(strings.TrimSpace(input.Profile))
	topic := strings.ToLower(strings.TrimSpace(input.Topic))
	narrative := strings.TrimSpace(input.Narrative)

	if !nlgProfiles[profile] || !nlgTopics[topic] {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Profil atau topik pengujian tidak valid"})
	}
	if narrative == "" {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Narasi belum tersedia"})
	}
	if len([]rune(narrative)) > 20000 {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Narasi terlalu panjang"})
	}

	scores := []int{input.ScoreData, input.ScoreContent, input.ScoreRelevance, input.ScoreClarity}
	total := 0
	for _, score := range scores {
		if score < 1 || score > 5 {
			return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"error": "Setiap skor harus bernilai 1-5"})
		}
		total += score
	}

	row := models.NlgTestResult{
		Profile:        profile,
		Topic:          topic,
		Narrative:      narrative,
		ScoreData:      input.ScoreData,
		ScoreContent:   input.ScoreContent,
		ScoreRelevance: input.ScoreRelevance,
		ScoreClarity:   input.ScoreClarity,
		AverageScore:   float64(total) / float64(len(scores)),
	}

	if err := facades.Orm().Query().Create(&row); err != nil {
		facades.Log().Errorf("gagal menyimpan hasil pengujian NLG: %v", err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan hasil pengujian NLG"})
	}

	return ctx.Response().Json(http.StatusCreated, http.Json{
		"message": "Penilaian NLG tersimpan",
		"data":    row,
	})
}
