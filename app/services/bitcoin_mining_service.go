package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

const (
	bitcoinMaxSupply       = 21000000.0
	bitcoinHalvingInterval = 210000
	bitcoinBlockMinutes    = 10

	// Data penambangan bergerak per blok (rata-rata 10 menit), tetapi halaman
	// tidak perlu menanyakan API setiap kali dibuka. Sepuluh menit cukup segar
	// untuk progress bar dan tetap jauh dari batas laju layanan publik.
	bitcoinMiningCacheKey = "market_bitcoin_mining"
	bitcoinMiningCacheTTL = 10 * time.Minute

	// Dua sumber tinggi blok. Keduanya membalas teks biasa, bukan JSON.
	mempoolTipHeightURL    = "https://mempool.space/api/blocks/tip/height"
	blockchainTipHeightURL = "https://blockchain.info/q/getblockcount"

	// Jaring pengaman kalau kedua API tidak bisa dihubungi: tinggi blok
	// diperkirakan dari halving 2024 (blok 840.000 pada 20 April 2024) memakai
	// rata-rata 10 menit per blok. Hasilnya ditandai estimated supaya halaman
	// tidak menyebutnya data live.
	bitcoinHalvingAnchorHeight = 840000
)

var (
	bitcoinHalvingAnchorTime = time.Date(2024, 4, 20, 0, 0, 0, 0, time.UTC)
	wibLocation              = time.FixedZone("WIB", 7*60*60)
)

// BitcoinMiningService menyajikan tinggi blok dan pasokan Bitcoin yang sudah
// ditambang. Datanya dipakai halaman Crypto dan Semua Pasar untuk progress bar
// penambangan serta hitungan blok menuju halving berikutnya.
type BitcoinMiningService struct{}

func NewBitcoinMiningService() *BitcoinMiningService {
	return &BitcoinMiningService{}
}

// Stats selalu mengembalikan sesuatu: live dari mempool.space/blockchain.info,
// salinan terakhir dari tabel market_cache kalau upstream sedang gagal, atau
// perkiraan dari jadwal halving. Endpoint pemanggilnya karena itu tidak pernah
// membalas 502 hanya karena data penambangan tidak tersedia.
func (s *BitcoinMiningService) Stats() map[string]any {
	fresh := !facades.Cache().Has(bitcoinMiningCacheKey)

	cached, err := facades.Cache().Remember(bitcoinMiningCacheKey, bitcoinMiningCacheTTL, func() (interface{}, error) {
		height, source, err := fetchBitcoinTipHeight()
		if err != nil {
			return nil, err
		}

		return bitcoinMiningStats(height, source, false, time.Now()), nil
	})
	if err != nil {
		var stored map[string]any
		if loadMarketCache(bitcoinMiningCacheKey, &stored) && len(stored) > 0 {
			facades.Log().Warningf("tinggi blok Bitcoin gagal diambil (%v), memakai cache terakhir dari database", err)
			stored["stale"] = true
			stored["data_note"] = "Data terakhir yang tersimpan; sumber live sedang tidak bisa dihubungi."

			return stored
		}

		facades.Log().Warningf("tinggi blok Bitcoin gagal diambil (%v), memakai perkiraan jadwal halving", err)

		return bitcoinMiningStats(estimatedBitcoinTipHeight(time.Now()), "estimasi", true, time.Now())
	}

	stats, ok := toMap(cached)
	if !ok {
		return bitcoinMiningStats(estimatedBitcoinTipHeight(time.Now()), "estimasi", true, time.Now())
	}

	if fresh {
		persistMarketCache(bitcoinMiningCacheKey, stats)
	}

	return stats
}

// fetchBitcoinTipHeight mencoba mempool.space lebih dulu, lalu blockchain.info.
// Satu sumber yang menjawab sudah cukup; kegagalan keduanya dilaporkan sebagai
// satu error supaya pemanggil bisa memakai perkiraan.
func fetchBitcoinTipHeight() (int64, string, error) {
	sources := []struct {
		url  string
		name string
	}{
		{url: mempoolTipHeightURL, name: "mempool.space"},
		{url: blockchainTipHeightURL, name: "blockchain.info"},
	}

	var lastErr error
	for _, source := range sources {
		body, err := fetchPlainText(source.url)
		if err != nil {
			lastErr = err
			continue
		}

		height, err := strconv.ParseInt(strings.TrimSpace(body), 10, 64)
		if err != nil || height <= 0 {
			lastErr = fmt.Errorf("%s membalas tinggi blok yang tidak valid: %q", source.name, body)
			continue
		}

		return height, source.name, nil
	}

	return 0, "", lastErr
}

// fetchPlainText membaca endpoint yang membalas angka/teks polos. fetchJson di
// market_service.go tidak bisa dipakai di sini karena kedua API tinggi blok
// tidak membungkus jawabannya dalam objek JSON.
func fetchPlainText(url string) (string, error) {
	response, err := facades.Http().
		FlushHeaders().
		WithHeader("User-Agent", marketUserAgent).
		Get(url)
	if err != nil {
		return "", fmt.Errorf("request to %s failed: %w", url, err)
	}

	if response.Failed() {
		return "", fmt.Errorf("request to %s returned status %d", url, response.Status())
	}

	body, err := response.Body()
	if err != nil {
		return "", fmt.Errorf("could not read response from %s: %w", url, err)
	}

	return strings.TrimSpace(body), nil
}

// estimatedBitcoinTipHeight menghitung perkiraan tinggi blok dari jadwal
// halving. Dipakai hanya saat API live dan cache persisten sama-sama kosong.
func estimatedBitcoinTipHeight(now time.Time) int64 {
	if now.Before(bitcoinHalvingAnchorTime) {
		return bitcoinHalvingAnchorHeight
	}

	blocks := int64(now.Sub(bitcoinHalvingAnchorTime) / (bitcoinBlockMinutes * time.Minute))

	return bitcoinHalvingAnchorHeight + blocks
}

// bitcoinMiningStats merakit payload satu kali. Semua angka diturunkan dari
// tinggi blok, bukan dari tabel statis, sehingga total koin dan sisa blok selalu
// konsisten satu sama lain.
func bitcoinMiningStats(height int64, source string, estimated bool, now time.Time) map[string]any {
	if height < 0 {
		height = 0
	}

	epoch := height / bitcoinHalvingInterval
	currentReward := 50.0
	for i := int64(0); i < epoch && i < 64; i++ {
		currentReward /= 2
	}

	nextHalvingHeight := (epoch + 1) * bitcoinHalvingInterval
	blocksRemaining := nextHalvingHeight - height
	if blocksRemaining < 0 {
		blocksRemaining = 0
	}

	totalMined := bitcoinMinedAt(height)
	minedPercent := totalMined / bitcoinMaxSupply * 100
	halvingDate := now.Add(time.Duration(blocksRemaining) * bitcoinBlockMinutes * time.Minute)

	return map[string]any{
		"height":                 height,
		"total_mined":            totalMined,
		"max_supply":             bitcoinMaxSupply,
		"mined_percent":          minedPercent,
		"remaining_supply":       bitcoinMaxSupply - totalMined,
		"next_halving_height":    nextHalvingHeight,
		"blocks_remaining":       blocksRemaining,
		"epoch":                  epoch,
		"current_reward":         currentReward,
		"next_reward":            currentReward / 2,
		"estimated_halving_date": halvingDate.UTC().Format(time.RFC3339),
		"source":                 source,
		"estimated":              estimated,
		"stale":                  false,
		"updated_at":             now.UTC().Format(time.RFC3339),
		"data_note":              bitcoinMiningNote(source, estimated, now),
	}
}

// bitcoinMinedAt menjumlahkan hadiah semua blok sampai tinggi tertentu. Blok
// genesis dihitung sebagai blok ke-0, jadi jumlah blok yang dibayar adalah
// height+1.
func bitcoinMinedAt(height int64) float64 {
	total := 0.0
	reward := 50.0
	remaining := height + 1

	for remaining > 0 && reward > 0 {
		if remaining >= bitcoinHalvingInterval {
			total += float64(bitcoinHalvingInterval) * reward
			remaining -= bitcoinHalvingInterval
		} else {
			total += float64(remaining) * reward
			remaining = 0
		}

		reward /= 2
	}

	return total
}

// bitcoinMiningNote adalah kalimat "Data per ..." yang ditampilkan di bawah
// progress bar, termasuk saat angkanya hanya perkiraan.
func bitcoinMiningNote(source string, estimated bool, now time.Time) string {
	stamp := now.In(wibLocation).Format("02 Jan 2006 15:04") + " WIB"

	if estimated {
		return "Data per " + stamp + " (perkiraan dari jadwal halving; sumber live belum bisa dihubungi)."
	}

	return "Data per " + stamp + " dari " + source + "."
}
