package controllers

import (
	"fmt"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

// defaultRiskProfile dipakai kalau user belum pernah mengerjakan kuis.
const defaultRiskProfile = models.RisikoModerat

// Ambang batas skor kuis.
//
// Asumsi skala: 5 pertanyaan dengan opsi bernilai 1-4, sehingga total skor
// berkisar 5-20. Ubah dua konstanta ini kalau jumlah atau bobot pertanyaannya
// berbeda.
const (
	skorMaksimalKonservatif = 10
	skorMaksimalModerat     = 15
)

// Batas skor yang dianggap sah, dipakai untuk menolak kiriman dari luar skala
// kuis (5 pertanyaan x nilai 1-4).
const (
	skorMinimal  = 5
	skorMaksimal = 20
)

type RiskController struct{}

func NewRiskController() *RiskController {
	return &RiskController{}
}

func (r *RiskController) SubmitQuiz(ctx http.Context) http.Response {
	var input struct {
		Skor int `json:"skor"`
	}

	if errors, ok := validate(ctx, map[string]any{
		"skor": fmt.Sprintf("required|integer|gte:%d|lte:%d", skorMinimal, skorMaksimal),
	}); !ok {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"errors": errors})
	}

	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": err.Error()})
	}

	profile := models.RiskProfile{
		UserID:     currentUserID(ctx),
		Skor:       input.Skor,
		TipeRisiko: TipeRisikoDariSkor(input.Skor),
	}

	if err := facades.Orm().Query().Create(&profile); err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Gagal menyimpan profil risiko"})
	}

	return ctx.Response().Json(http.StatusCreated, http.Json{
		"message": "Profil risiko berhasil disimpan",
		"profile": http.Json{
			"id":          profile.ID,
			"skor":        profile.Skor,
			"tipe_risiko": profile.TipeRisiko,
			"created_at":  profile.CreatedAt,
		},
	})
}

// Profile mengembalikan profil risiko milik user yang sedang login.
//
// has_profile memisahkan "belum pernah mengerjakan kuis" dari "hasil kuisnya
// Moderat", sesuatu yang tidak bisa dibedakan oleh satu field tipe saja.
func (r *RiskController) Profile(ctx http.Context) http.Response {
	profile, found := latestRiskProfileRecord(currentUserID(ctx))
	if !found {
		return ctx.Response().Json(http.StatusOK, http.Json{
			"has_profile": false,
			"profile":     nil,
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"has_profile": true,
		"profile": http.Json{
			"id":          profile.ID,
			"skor":        profile.Skor,
			"tipe_risiko": profile.TipeRisiko,
			"created_at":  profile.CreatedAt,
		},
	})
}

// TipeRisikoDariSkor menerjemahkan skor kuis menjadi tipe risiko.
func TipeRisikoDariSkor(skor int) string {
	switch {
	case skor <= skorMaksimalKonservatif:
		return models.RisikoKonservatif
	case skor <= skorMaksimalModerat:
		return models.RisikoModerat
	default:
		return models.RisikoAgresif
	}
}

// latestRiskProfileRecord mengambil baris kuis terbaru milik user. Boolean
// kedua membedakan "belum pernah kuis" dari kegagalan baca, tanpa bergantung
// pada orm.Model yang terisi.
func latestRiskProfileRecord(userID uint) (models.RiskProfile, bool) {
	var profile models.RiskProfile

	if err := facades.Orm().Query().
		Where("user_id = ?", userID).
		Order("id desc").
		First(&profile); err != nil || profile.ID == 0 {
		return models.RiskProfile{}, false
	}

	return profile, true
}

// latestRiskProfile mengambil tipe risiko dari kuis terbaru milik user.
// Kalau user belum pernah mengerjakan kuis, dipakai defaultRiskProfile.
func latestRiskProfile(userID uint) string {
	profile, found := latestRiskProfileRecord(userID)
	if !found {
		return defaultRiskProfile
	}

	return profile.TipeRisiko
}
