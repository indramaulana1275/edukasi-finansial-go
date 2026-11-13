package services

import (
	"encoding/json"
	"sync"
	"time"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

// marketCacheTable menyimpan salinan terakhir data pasar yang berhasil diambil
// dari upstream. Cache di memori (facades.Cache) hilang saat server restart,
// sedangkan tabel ini bertahan - itulah yang dipakai sebagai cadangan ketika
// CoinGecko atau Yahoo Finance membalas 429/503 sehingga grafik tidak kosong.
const marketCacheTable = "market_cache"

// marketCacheRow adalah satu baris cache. Kuncinya disimpan sebagai teks biasa
// (bukan primary key) supaya penulisan ulang cukup menghapus baris lama lalu
// membuat baris baru, tanpa bergantung pada dialek upsert masing-masing driver.
type marketCacheRow struct {
	ID       uint   `gorm:"primaryKey"`
	CacheKey string `gorm:"column:cache_key"`
	Payload  string `gorm:"column:payload"`
	CachedAt int64  `gorm:"column:cached_at"`
}

var marketCacheWarning sync.Once

// persistMarketCache menyalin sebuah nilai ke database. Kegagalan menulis cache
// tidak boleh menggagalkan permintaan yang datanya sudah berhasil diambil, jadi
// errornya hanya dicatat sekali supaya log tidak dibanjiri pesan yang sama.
func persistMarketCache(key string, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		warnMarketCache(err)
		return
	}

	if _, err := facades.Orm().Query().Table(marketCacheTable).
		Where("cache_key = ?", key).Delete(&marketCacheRow{}); err != nil {
		warnMarketCache(err)
		return
	}

	row := marketCacheRow{
		CacheKey: key,
		Payload:  string(payload),
		CachedAt: time.Now().Unix(),
	}

	if err := facades.Orm().Query().Table(marketCacheTable).Create(&row); err != nil {
		warnMarketCache(err)
	}
}

// loadMarketCache membaca salinan terakhir dari database. Nilai `false` berarti
// belum ada cache yang bisa dipakai, bukan berarti permintaan harus gagal.
func loadMarketCache(key string, target any) bool {
	var rows []marketCacheRow

	if err := facades.Orm().Query().Table(marketCacheTable).
		Where("cache_key = ?", key).Order("id desc").Limit(1).Get(&rows); err != nil {
		warnMarketCache(err)
		return false
	}

	if len(rows) == 0 || rows[0].Payload == "" {
		return false
	}

	if err := json.Unmarshal([]byte(rows[0].Payload), target); err != nil {
		warnMarketCache(err)
		return false
	}

	return true
}

// cachedHistorySeries membungkus loadMarketCache untuk seri grafik.
func cachedHistorySeries(key string) (historySeries, bool) {
	var series historySeries
	if !loadMarketCache(key, &series) || len(series.Points) == 0 {
		return historySeries{}, false
	}

	return series, true
}

func warnMarketCache(err error) {
	marketCacheWarning.Do(func() {
		facades.Log().Warningf("cache pasar persisten tidak bisa dipakai: %v", err)
	})
}
