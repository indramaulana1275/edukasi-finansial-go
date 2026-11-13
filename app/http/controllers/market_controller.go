package controllers

import (
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/services"
)

type MarketController struct {
	market *services.MarketService
	world  *services.WorldAssetService
	mining *services.BitcoinMiningService
}

func NewMarketController(market *services.MarketService, world *services.WorldAssetService, mining *services.BitcoinMiningService) *MarketController {
	return &MarketController{market: market, world: world, mining: mining}
}

func (r *MarketController) Index(ctx http.Context) http.Response {
	prices, err := r.market.GetCurrentPrices()
	if err != nil {
		return ctx.Response().Json(http.StatusBadGateway, http.Json{
			"error": err.Error(),
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json(prices))
}

// History menyajikan seri harga untuk grafik. Rentang 1-3 hari memakai titik
// per jam (tombol 24J dan 3H); rentang lebih panjang memakai penutupan harian.
//
// Query: asset (nilai `asset` di tabel halaman mana pun — id koin CoinGecko
// seperti bitcoin/sui, saham seperti bbca/nasdaq/aapl, komoditas seperti silver,
// atau kurs usd — plus gold; default bitcoin) dan days (1-3650, default 30).
// Rentang yang keluar dari batas dipangkas, bukan ditolak, supaya dashboard tetap
// dapat data walau nilainya sedikit aneh.
func (r *MarketController) History(ctx http.Context) http.Response {
	asset := strings.ToLower(strings.TrimSpace(ctx.Request().Query("asset", services.HistoryAssetBitcoin)))
	if !services.IsHistoryAsset(asset) {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{
			"error": "asset harus salah satu dari: " + services.HistoryAssetNames(),
		})
	}

	days := ctx.Request().QueryInt("days", 30)
	if days < services.HistoryMinDays {
		days = services.HistoryMinDays
	}
	if days > services.HistoryMaxDays {
		days = services.HistoryMaxDays
	}

	series, source, interval, err := r.market.GetHistoricalPrices(asset, days)
	if err != nil {
		// Penyebab teknisnya (mis. 429 dari CoinGecko) cukup masuk log; user
		// hanya perlu tahu bahwa ini bisa dicoba lagi.
		facades.Log().Errorf("gagal mengambil riwayat harga %s: %v", asset, err)

		return ctx.Response().Json(http.StatusBadGateway, http.Json{
			"error": "Gagal mengambil riwayat harga " + asset + ". Coba beberapa saat lagi ya.",
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"asset":      asset,
		"days":       days,
		"interval":   interval,
		"source":     source,
		"data":       series,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// Candlestick menyajikan bar OHLC harian untuk grafik lilin halaman Saham.
// Query: asset (mis. ihsg/bbca/aapl) dan days (5-365, default 60).
func (r *MarketController) Candlestick(ctx http.Context) http.Response {
	asset := strings.ToLower(strings.TrimSpace(ctx.Request().Query("asset", "ihsg")))
	if !services.IsHistoryAsset(asset) {
		return ctx.Response().Json(http.StatusUnprocessableEntity, http.Json{
			"error": "asset harus salah satu dari: " + services.HistoryAssetNames(),
		})
	}

	days := ctx.Request().QueryInt("days", 60)
	if days < 5 {
		days = 5
	}
	if days > 365 {
		days = 365
	}

	candles, source, err := r.market.GetCandlestick(asset, days)
	if err != nil {
		facades.Log().Errorf("gagal mengambil candlestick %s: %v", asset, err)

		return ctx.Response().Json(http.StatusBadGateway, http.Json{
			"error": "Gagal mengambil data candlestick " + asset + ". Coba beberapa saat lagi ya.",
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"asset":      asset,
		"days":       days,
		"source":     source,
		"data":       candles,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// Crypto menyajikan sepuluh koin untuk halaman Crypto.
func (r *MarketController) Crypto(ctx http.Context) http.Response {
	coins, err := r.market.GetCryptoPrices()
	if err != nil {
		facades.Log().Errorf("gagal mengambil daftar crypto: %v", err)

		return ctx.Response().Json(http.StatusBadGateway, http.Json{
			"error": "Gagal mengambil data crypto. Coba beberapa saat lagi ya.",
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":       coins,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// Forex menyajikan kurs enam mata uang terhadap rupiah untuk halaman Kurs.
func (r *MarketController) Forex(ctx http.Context) http.Response {
	rates, err := r.market.GetForexPrices()
	if err != nil {
		facades.Log().Errorf("gagal mengambil daftar kurs: %v", err)

		return ctx.Response().Json(http.StatusBadGateway, http.Json{
			"error": "Gagal mengambil data kurs. Coba beberapa saat lagi ya.",
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":       rates,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// Saham menyajikan sepuluh baris halaman Saham: lima emiten Indonesia (market
// "id") dan lima aset luar negeri (market "us"). Frontend memecahnya jadi dua
// tabel dari field `market` itu.
func (r *MarketController) Saham(ctx http.Context) http.Response {
	stocks, err := r.market.GetSaham()
	if err != nil {
		facades.Log().Errorf("gagal mengambil daftar saham: %v", err)

		return ctx.Response().Json(http.StatusBadGateway, http.Json{
			"error": "Gagal mengambil data saham. Coba beberapa saat lagi ya.",
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":       stocks,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// Komoditas menyajikan perak, minyak, dan gas untuk halaman Komoditas.
func (r *MarketController) Komoditas(ctx http.Context) http.Response {
	commodities, err := r.market.GetKomoditas()
	if err != nil {
		facades.Log().Errorf("gagal mengambil data komoditas: %v", err)

		return ctx.Response().Json(http.StatusBadGateway, http.Json{
			"error": "Gagal mengambil data komoditas. Coba beberapa saat lagi ya.",
		})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":       commodities,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// WorldAssets menyajikan tabel halaman "Aset Terbesar Dunia": dua puluh aset dan
// kelas aset terbesar, sudah urut menurun.
// Endpoint ini tidak pernah membalas 502 seperti daftar harga lain: isinya
// estimasi global yang selalu tersedia, dan bagian yang ikut harga pasar (emas,
// perak, Bitcoin, Ethereum) hanya kehilangan angka live-nya bila upstream gagal.
// Jumlah baris live dikirim terpisah supaya frontend bisa menyebut dari mana
// angkanya datang, bukan menampilkan estimasi seolah-olah harga terkini.
func (r *MarketController) WorldAssets(ctx http.Context) http.Response {
	assets := r.world.List()

	live := 0
	for _, asset := range assets {
		if asset.Live {
			live++
		}
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"data":       assets,
		"live":       live,
		"fetched_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// BitcoinMining menyajikan tinggi blok dan pasokan Bitcoin yang sudah ditambang
// untuk kartu penambangan di halaman Crypto dan Semua Pasar.
//
// Endpoint ini tidak pernah membalas 502: kalau mempool.space dan blockchain.info
// sama-sama tidak bisa dihubungi, layanan memakai salinan terakhir dari
// market_cache atau perkiraan dari jadwal halving, dan field estimated/stale
// memberi tahu frontend bahwa angkanya bukan data live.
func (r *MarketController) BitcoinMining(ctx http.Context) http.Response {
	return ctx.Response().Json(http.StatusOK, http.Json{
		"data": r.mining.Stats(),
	})
}
