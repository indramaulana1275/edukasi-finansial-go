package controllers

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

type PortfolioController struct{}

func NewPortfolioController() *PortfolioController {
	return &PortfolioController{}
}

// Index mengembalikan seluruh holding milik user yang sedang login. Harga
// sekarang tidak dihitung di sini: frontend sudah punya daftar crypto, saham,
// dan komoditas beserta /api/market, jadi nilainya dihitung dari sumber yang
// sama dengan tabel aset tanpa menambah satu sumber harga lagi.
func (r *PortfolioController) Index(ctx http.Context) http.Response {
	holdings := make([]models.Portfolio, 0)

	if err := facades.Orm().Query().
		Where("user_id = ?", currentUserID(ctx)).
		Order("id desc").
		Get(&holdings); err != nil {
		facades.Log().Errorf("gagal membaca portfolio user %d: %v", currentUserID(ctx), err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal membaca portfolio",
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"data": holdings})
}

// Store menambahkan satu holding. Jenis aset ditentukan server dari simbolnya,
// jadi asset_type tidak bisa dipalsukan lewat body request.
func (r *PortfolioController) Store(ctx http.Context) http.Response {
	var input struct {
		AssetSymbol string  `json:"asset_symbol"`
		Quantity    float64 `json:"quantity"`
		BuyPrice    float64 `json:"buy_price"`
	}

	if errors, ok := validate(ctx, map[string]any{
		"asset_symbol": "required",
		"quantity":     "required|numeric",
		"buy_price":    "required|numeric",
	}); !ok {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{"errors": errors})
	}

	if err := ctx.Request().Bind(&input); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": err.Error()})
	}

	symbol := strings.ToUpper(strings.TrimSpace(input.AssetSymbol))
	assetType, known := models.AssetTypeForSymbol(symbol)
	if !known {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{
			"error": "asset_symbol tidak dikenal. Pilih aset dari daftar portofolio.",
		})
	}

	// Validator tidak punya aturan "lebih besar dari", jadi angka nol dan
	// negatif disaring di sini (jumlah unit boleh pecahan, mis. 0,5 BTC).
	if input.Quantity <= 0 || input.BuyPrice <= 0 {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{
			"error": "quantity dan buy_price harus lebih besar dari nol",
		})
	}

	holding := models.Portfolio{
		UserID:      currentUserID(ctx),
		AssetSymbol: symbol,
		AssetType:   assetType,
		Quantity:    input.Quantity,
		BuyPrice:    input.BuyPrice,
	}

	if err := facades.Orm().Query().Create(&holding); err != nil {
		facades.Log().Errorf("gagal menyimpan holding user %d: %v", holding.UserID, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal menyimpan holding",
		})
	}

	return ctx.Response().Json(http.StatusCreated, http.Json{
		"message": "Holding ditambahkan",
		"data":    holding,
	})
}

// Destroy menghapus satu holding milik user. Filternya selalu menyertakan
// user_id supaya id milik user lain tidak bisa dihapus dari luar.
func (r *PortfolioController) Destroy(ctx http.Context) http.Response {
	id := ctx.Request().RouteInt("id")
	if id <= 0 {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{
			"error": "id holding tidak valid",
		})
	}

	result, err := facades.Orm().Query().
		Where("id = ?", id).
		Where("user_id = ?", currentUserID(ctx)).
		Delete(&models.Portfolio{})
	if err != nil {
		facades.Log().Errorf("gagal menghapus holding %d: %v", id, err)

		return ctx.Response().Json(http.StatusInternalServerError, http.Json{
			"error": "Gagal menghapus holding",
		})
	}

	if result.RowsAffected == 0 {
		return ctx.Response().Json(http.StatusNotFound, http.Json{"error": "Holding tidak ditemukan"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{"message": "Holding dihapus"})
}
