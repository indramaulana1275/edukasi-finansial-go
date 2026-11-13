package services

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

const (
	marketCacheKey = "market_prices"
	// Satu jam: cukup untuk menjauhkan CoinGecko/Yahoo dari batas laju, tetapi
	// harga di kartu masih terasa terkini. Nilai ini juga dipakai daftar koin,
	// kurs, saham, dan komoditas.
	marketCacheTTL = time.Hour

	cryptoCacheKey    = "market_crypto"
	forexCacheKey     = "market_forex"
	sahamCacheKey     = "market_saham"
	komoditasCacheKey = "market_komoditas"

	coinGeckoBitcoinURL = "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd&include_24hr_change=true"
	goldSpotURL         = "https://xaus.com/api/v1/spot?unit=oz"

	// Daftar harga koin. Parameter ids tidak ditulis di sini, melainkan disusun
	// dari cryptoCoins supaya daftar koin di halaman Crypto dan daftar asset yang
	// boleh diminta ke /api/market/history tidak pernah berbeda.
	coinGeckoMarketsBaseURL = "https://api.coingecko.com/api/v3/coins/markets"

	// Yahoo Finance menyajikan kuotasi terbaru di blok meta setiap chart, jadi
	// satu bentuk permintaan melayani kurs, indeks saham, dan komoditas. Semua
	// pasangan FX terhadap rupiah bernama <kode>IDR=X, sedangkan indeks memakai
	// tanda ^ (di-escape saat dipanggil).
	yahooQuoteURL = "https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d"

	// Seri harga harian untuk grafik. CoinGecko hanya dipakai sampai 365 hari —
	// tier anonim membalas 401 untuk days>365 — sedangkan rentang lebih panjang
	// dilayani Yahoo Finance, sama seperti emas dan kurs.
	coinGeckoHistoryURL = "https://api.coingecko.com/api/v3/coins/%s/market_chart?vs_currency=usd&days=365"
	yahooChartURL       = "https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=%s"

	// Rentang 24J dan 3H butuh titik per jam, bukan penutupan harian. Yahoo
	// melayani interval 1h untuk range 5d, jadi satu permintaan menutupi
	// keduanya lalu dipotong sesuai hari yang diminta.
	yahooHourlyChartURL = "https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1h&range=5d"
	historyHourlyLayout = "2006-01-02 15:04"

	// Simbol Yahoo Finance untuk emas: harga live-nya datang dari xaus.com, tapi
	// riwayatnya dari futures COMEX. Simbol koin dan aset halaman lain ada di
	// daftarnya masing-masing (cryptoCoins, sahamInstruments, …).
	yahooGoldSymbol = "GC=F"

	// Panel Harga Bank Indonesia (PIHPS) menyajikan harga pangan nasional
	// lewat grid DevExpress. Satu permintaan mengembalikan baris "Semua
	// Provinsi" berisi kolom tanggal; bentuknya dipakai baik untuk harga
	// terkini maupun seri grafik. tipe_laporan 1 adalah laporan harian.
	pihpsGridURL     = "https://www.bi.go.id/hargapangan/WebSite/TabelHarga/GetGridDataKomoditas"
	pihpsDailyReport = 1

	// Grid PIHPS menghitung seluruh provinsi sekaligus, jadi satu permintaan
	// setahun bisa memakan puluhan detik. Rentangnya dipecah per 30 hari dan
	// dijalankan beberapa sekaligus: 12 potongan selesai sekitar 3 detik,
	// sedangkan satu permintaan setahun bisa 40 detik.
	pihpsChunkDays    = 30
	pihpsChunkWorkers = 4

	historyCacheTTL   = time.Hour
	historyDateLayout = "2006-01-02"

	// Jumlah permintaan kuotasi Yahoo yang boleh berjalan bersamaan; lihat
	// catatan di fetchYahooQuotes.
	yahooQuoteWorkers = 6

	// CoinGecko anonim membatasi laju permintaan market_chart, jadi menjelajah
	// beberapa rentang berturut-turut gampang menghasilkan 429. Permintaan
	// seperti itu diulang dengan jeda bertingkat sebelum menyerah.
	historyFetchAttempts = 3
	historyRetryDelay    = 1200 * time.Millisecond

	// Yahoo Finance answers 429 to requests without a browser-like User-Agent.
	marketUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

// Aset khusus di luar daftar harga halaman mana pun, plus batas rentang hari.
// Bitcoin tetap terdaftar di sini karena dipakai sebagai nilai default query
// `asset`. Emas hanya ada di sini karena kartunya memakai harga spot xaus.com,
// sementara riwayatnya datang dari futures COMEX Yahoo.
const (
	HistoryAssetBitcoin = "bitcoin"
	HistoryAssetGold    = "gold"

	HistoryMinDays = 1
	// 10T. CoinGecko melewati batas ini dengan days=max, Yahoo dengan range=10y.
	HistoryMaxDays = 3650

	// Sampai tiga hari, grafik memakai titik per jam supaya tombol 24J dan 3H
	// benar-benar menunjukkan pergerakan intraday, bukan tiga titik harian.
	HistoryHourlyMaxDays = 3
)

// Rentang upstream yang dipakai sebagai kunci cache seri penuh. Nilainya juga
// berlaku sebagai range Yahoo Finance, jadi "1y" melayani 3H..1T dan "10y"
// melayani 5T sekaligus 10T.
const (
	historySpanYear   = "1y"
	historySpanDecade = "10y"
)

// cryptoCoin memasangkan id CoinGecko dengan ticker Yahoo Finance-nya.
type cryptoCoin struct {
	ID    string
	Yahoo string
}

// Koin di halaman Crypto. Id CoinGecko dipakai untuk daftar harga sekaligus
// sebagai nilai parameter asset di /api/market/history, sehingga halaman Crypto
// dan grafiknya selalu bicara soal koin yang sama.
//
// Ticker Yahoo menemani tiap koin karena rentang di atas setahun tidak dilayani
// CoinGecko anonim, dan bentuk tickernya tidak selalu <simbol>-USD: Sui terdaftar
// sebagai SUI20947-USD (SUI-USD sudah tidak ada).
var cryptoCoins = []cryptoCoin{
	{ID: "bitcoin", Yahoo: "BTC-USD"},
	{ID: "ethereum", Yahoo: "ETH-USD"},
	{ID: "binancecoin", Yahoo: "BNB-USD"},
	{ID: "solana", Yahoo: "SOL-USD"},
	{ID: "ripple", Yahoo: "XRP-USD"},
	{ID: "cardano", Yahoo: "ADA-USD"},
	{ID: "sui", Yahoo: "SUI20947-USD"},
	{ID: "tron", Yahoo: "TRX-USD"},
	{ID: "polkadot", Yahoo: "DOT-USD"},
	{ID: "litecoin", Yahoo: "LTC-USD"},
}

// cryptoCoinByID mencari koin dari id CoinGecko-nya. Daftarnya hanya sepuluh, jadi
// pemindaian linear lebih jelas daripada map kedua yang harus dijaga sinkron.
func cryptoCoinByID(id string) (cryptoCoin, bool) {
	for _, coin := range cryptoCoins {
		if coin.ID == id {
			return coin, true
		}
	}

	return cryptoCoin{}, false
}

// coinGeckoMarketsURL menyusun URL daftar harga dari cryptoCoins.
func coinGeckoMarketsURL() string {
	ids := make([]string, 0, len(cryptoCoins))
	for _, coin := range cryptoCoins {
		ids = append(ids, coin.ID)
	}

	return coinGeckoMarketsBaseURL + "?vs_currency=usd&ids=" + strings.Join(ids, ",") +
		"&price_change_percentage=24h&order=market_cap_desc"
}

type MarketService struct{}

func NewMarketService() *MarketService {
	return &MarketService{}
}

// GetCurrentPrices returns the latest bitcoin, gold and USD/IDR quotes in a
// single payload. The result is cached for an hour because CoinGecko and
// Yahoo Finance both rate limit anonymous traffic aggressively.
//
// Salinan terakhir juga ditulis ke tabel market_cache. Kalau ketiga sumber
// gagal (mis. kuota habis) dan cache memori sudah kosong setelah restart, data
// terakhir yang pernah berhasil tetap disajikan alih-alih membuat kartu harga
// kosong.
func (s *MarketService) GetCurrentPrices() (map[string]interface{}, error) {
	fresh := !facades.Cache().Has(marketCacheKey)

	cached, err := facades.Cache().Remember(marketCacheKey, marketCacheTTL, func() (interface{}, error) {
		return s.fetchAll()
	})
	if err != nil {
		var stored map[string]interface{}
		if loadMarketCache(marketCacheKey, &stored) && len(stored) > 0 {
			facades.Log().Warningf("harga pasar gagal diambil (%v), memakai cache terakhir dari database", err)
			return stored, nil
		}

		return nil, err
	}

	prices, ok := cached.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected market cache payload of type %T", cached)
	}

	if fresh {
		persistMarketCache(marketCacheKey, prices)
	}

	return prices, nil
}

// fetchAll queries the three providers independently so that one failing source
// does not take down the whole endpoint. It only fails when every source fails.
func (s *MarketService) fetchAll() (map[string]interface{}, error) {
	prices := map[string]interface{}{}
	failures := map[string]string{}

	if bitcoin, err := s.fetchBitcoin(); err != nil {
		failures["bitcoin"] = err.Error()
	} else {
		prices["bitcoin"] = bitcoin
	}

	if gold, err := s.fetchGold(); err != nil {
		failures["gold"] = err.Error()
	} else {
		prices["gold"] = gold
	}

	if usd, err := s.fetchUsdIdr(); err != nil {
		failures["usd"] = err.Error()
	} else {
		prices["usd"] = usd
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("all market data providers failed: %v", failures)
	}

	if len(failures) > 0 {
		prices["errors"] = failures
	}

	prices["fetched_at"] = time.Now().UTC().Format(time.RFC3339)

	return prices, nil
}

func (s *MarketService) fetchBitcoin() (map[string]any, error) {
	payload, err := fetchJson(coinGeckoBitcoinURL)
	if err != nil {
		return nil, err
	}

	bitcoin, ok := toMap(payload["bitcoin"])
	if !ok {
		return nil, fmt.Errorf("unexpected CoinGecko payload: %v", payload)
	}

	return map[string]any{
		"price_usd":  toFloat64(bitcoin["usd"]),
		"change_24h": toFloat64(bitcoin["usd_24h_change"]),
		"currency":   "USD",
		"source":     "coingecko",
	}, nil
}

func (s *MarketService) fetchGold() (map[string]any, error) {
	payload, err := fetchJson(goldSpotURL)
	if err != nil {
		return nil, err
	}

	priceOunce := toFloat64(payload["spot_usd_oz"])
	if priceOunce == 0 {
		if xau, ok := toMap(payload["xau"]); ok {
			priceOunce = toFloat64(xau["price"])
		}
	}

	if priceOunce == 0 {
		return nil, fmt.Errorf("gold spot price missing from xaus.com payload")
	}

	gold := map[string]any{
		"price_usd_oz":   priceOunce,
		"price_usd_gram": toFloat64(payload["per_gram_usd"]),
		"currency":       "USD",
		"unit":           "troy_oz",
		"source":         "xaus.com",
	}

	// xaus.com hanya mengirim harga spot tanpa pembanding harian, jadi persentase
	// 24 jam diambil dari futures COMEX (GC=F) lewat Yahoo. Ini angka pendukung
	// saja: kalau Yahoo sedang tidak bisa dihubungi, kartu emas tetap tampil
	// lengkap dengan harganya, hanya badge persennya yang kosong.
	if quote, err := fetchYahooQuote(yahooGoldSymbol); err == nil {
		gold["change_percent"] = quote["change_percent"]
	} else {
		facades.Log().Warningf("gagal mengambil persentase 24 jam emas dari Yahoo: %v", err)
	}

	return gold, nil
}

func (s *MarketService) fetchUsdIdr() (map[string]any, error) {
	quote, err := fetchYahooIdrRate("USD")
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"pair":           "USD/IDR",
		"idr_rate":       quote["rate"],
		"change_percent": quote["change_percent"],
		"previous_close": quote["previous_close"],
		"source":         "yahoo_finance",
	}, nil
}

// Mata uang di halaman Kurs. Semuanya dikutip terhadap rupiah supaya angkanya
// langsung bisa dibaca tanpa konversi silang.
//
// Isinya sepuluh mata uang dengan pangsa transaksi terbesar di dunia menurut
// survei BIS, dan urutan di sini adalah urutan tombol "Sebelumnya / Selanjutnya"
// di kartu besar halaman Kurs — enam yang lama tetap di posisinya supaya daftar
// yang sudah dikenal tidak berubah, lalu empat pendatang baru menyusul.
var forexCurrencies = []struct {
	Code string
	Name string
}{
	{"USD", "Dolar Amerika Serikat"},
	{"EUR", "Euro"},
	{"JPY", "Yen Jepang"},
	{"GBP", "Pound Sterling"},
	{"SGD", "Dolar Singapura"},
	{"CNY", "Yuan Tiongkok"},
	{"AUD", "Dolar Australia"},
	{"CAD", "Dolar Kanada"},
	{"CHF", "Franc Swiss"},
	{"HKD", "Dolar Hong Kong"},
}

// yahooInstrument memasangkan nama aset yang dipakai API dengan simbol Yahoo dan
// nama tampilan di UI. Namanya ditulis sendiri karena Yahoo memberi nama kontrak
// berjangka untuk komoditas, bukan nama barangnya (SI=F tampil sebagai
// "Silver Dec 26").
//
// Asset inilah nilai parameter `asset` di /api/market/history, sekaligus yang
// dikirim balik ke frontend pada setiap baris tabel — jadi grafik di halaman ini
// tidak perlu daftar pemetaan kedua yang harus dijaga sinkron.
//
// Market hanya diisi oleh halaman yang memecah tabelnya jadi beberapa bursa
// (halaman Saham punya dua: emiten Indonesia dan luar negeri); daftar lain
// membiarkannya kosong dan field-nya tidak ikut dikirim.
type yahooInstrument struct {
	Asset  string
	Symbol string
	Name   string
	Unit   string
	Market string

	// PIHPSID menandai komoditas pangan yang harganya datang dari Panel Harga
	// Bank Indonesia (PIHPS), bukan Yahoo Finance. Nilainya adalah id komoditas
	// di endpoint GetGridDataKomoditas (mis. com_11 untuk bawang merah).
	PIHPSID string
}

// Saham di halaman Saham, dua puluh baris terbagi dua tab. Emiten Indonesia
// memakai ticker berakhiran .JK dan harganya dalam rupiah, sedangkan tab luar
// negeri berisi tiga indeks utama Amerika plus tujuh saham besarnya. Indeks dan
// emiten sama-sama terdaftar di sini supaya keduanya bisa diminta ke
// /api/market/history lewat field Asset-nya (lihat historyYahooSymbol).
var sahamInstruments = []yahooInstrument{
	{Asset: "ihsg", Symbol: "^JKSE", Name: "IHSG (Indeks Harga Saham Gabungan)", Market: "id"},
	{Asset: "bbca", Symbol: "BBCA.JK", Name: "Bank Central Asia (BBCA)", Market: "id"},
	{Asset: "bbri", Symbol: "BBRI.JK", Name: "Bank Rakyat Indonesia (BBRI)", Market: "id"},
	{Asset: "tlkm", Symbol: "TLKM.JK", Name: "Telkom Indonesia (TLKM)", Market: "id"},
	{Asset: "asii", Symbol: "ASII.JK", Name: "Astra International (ASII)", Market: "id"},
	{Asset: "bmri", Symbol: "BMRI.JK", Name: "Bank Mandiri (BMRI)", Market: "id"},
	{Asset: "unvr", Symbol: "UNVR.JK", Name: "Unilever Indonesia (UNVR)", Market: "id"},
	{Asset: "icbp", Symbol: "ICBP.JK", Name: "Indofood CBP (ICBP)", Market: "id"},
	{Asset: "indf", Symbol: "INDF.JK", Name: "Indofood Sukses Makmur (INDF)", Market: "id"},
	{Asset: "klbf", Symbol: "KLBF.JK", Name: "Kalbe Farma (KLBF)", Market: "id"},
	{Asset: "nasdaq", Symbol: "^IXIC", Name: "NASDAQ Composite", Market: "us"},
	{Asset: "sp500", Symbol: "^GSPC", Name: "S&P 500", Market: "us"},
	{Asset: "dowjones", Symbol: "^DJI", Name: "Dow Jones Industrial Average", Market: "us"},
	{Asset: "aapl", Symbol: "AAPL", Name: "Apple (AAPL)", Market: "us"},
	{Asset: "msft", Symbol: "MSFT", Name: "Microsoft (MSFT)", Market: "us"},
	{Asset: "googl", Symbol: "GOOGL", Name: "Alphabet / Google (GOOGL)", Market: "us"},
	{Asset: "amzn", Symbol: "AMZN", Name: "Amazon (AMZN)", Market: "us"},
	{Asset: "tsla", Symbol: "TSLA", Name: "Tesla (TSLA)", Market: "us"},
	{Asset: "nvda", Symbol: "NVDA", Name: "NVIDIA (NVDA)", Market: "us"},
	{Asset: "meta", Symbol: "META", Name: "Meta Platforms (META)", Market: "us"},
}

// Komoditas di halaman Komoditas, tiga puluh baris: dua puluh enam kontrak
// berjangka/proxy Yahoo Finance dan empat harga pangan Bank Indonesia (PIHPS).
//
// Emas ikut di tabel dengan perak, minyak, dan gas — harganya dari futures
// COMEX (GC=F), bukan spot xaus.com, karena Yahoo tidak menyediakan harga spot
// emas yang sama dengan xaus.com. Empat baris terakhir sengaja bukan futures:
// cabai dan bawang adalah harga rata-rata nasional pasar tradisional dari
// Panel Harga Bank Indonesia, jadi angkanya rupiah per kilogram dan grafiknya
// juga berasal dari sumber yang sama.
//
// Satuan tiap baris ditulis sendiri di sini karena Yahoo menamai kontraknya
// dengan bulan ("Copper Dec 26") dan tidak mengirim satuan yang bisa dipakai —
// sama seperti halaman Saham. Kopi dan biji-bijian dilaporkan dalam sen oleh
// Yahoo; lihat yahooPriceScale.
//
// Lithium dan batu bara kokas memakai proxy yang tersedia di Yahoo (ETF LIT
// dan saham HCC) karena tidak ada kontrak berjangka yang likuid di sana.
var komoditasInstruments = []yahooInstrument{
	{Asset: "gold", Symbol: "GC=F", Name: "Emas", Unit: "per troy ounce"},
	{Asset: "silver", Symbol: "SI=F", Name: "Perak", Unit: "per troy ounce"},
	{Asset: "oil", Symbol: "CL=F", Name: "Minyak Mentah WTI", Unit: "per barel"},
	{Asset: "gas", Symbol: "NG=F", Name: "Gas Alam", Unit: "per MMBtu"},
	{Asset: "copper", Symbol: "HG=F", Name: "Tembaga", Unit: "per pon"},
	{Asset: "coffee", Symbol: "KC=F", Name: "Biji Kopi Arabika", Unit: "per pon"},
	{Asset: "wheat", Symbol: "ZW=F", Name: "Gandum", Unit: "per bushel"},
	{Asset: "corn", Symbol: "ZC=F", Name: "Jagung", Unit: "per bushel"},
	{Asset: "soybean", Symbol: "ZS=F", Name: "Kedelai", Unit: "per bushel"},
	{Asset: "coal", Symbol: "MTF=F", Name: "Batu Bara Thermal", Unit: "per ton"},
	{Asset: "cpo", Symbol: "CPO=F", Name: "Minyak Sawit (CPO)", Unit: "per ton"},
	{Asset: "cocoa", Symbol: "CC=F", Name: "Kakao", Unit: "per ton"},
	{Asset: "rice", Symbol: "ZR=F", Name: "Beras", Unit: "per cwt"},
	{Asset: "sugar", Symbol: "SB=F", Name: "Gula", Unit: "per pon"},
	{Asset: "aluminum", Symbol: "ALI=F", Name: "Aluminium", Unit: "per ton"},
	{Asset: "ironore", Symbol: "TIO=F", Name: "Bijih Besi", Unit: "per ton"},
	{Asset: "brent", Symbol: "BZ=F", Name: "Minyak Brent", Unit: "per barel"},
	{Asset: "gasoline", Symbol: "RB=F", Name: "Bensin", Unit: "per galon"},
	{Asset: "uranium", Symbol: "UX=F", Name: "Uranium", Unit: "per pon U3O8"},
	{Asset: "lithium", Symbol: "LIT", Name: "Lithium", Unit: "per saham ETF"},
	{Asset: "steel", Symbol: "HRC=F", Name: "Baja", Unit: "per ton"},
	{Asset: "zinc", Symbol: "ZNC=F", Name: "Seng", Unit: "per ton"},
	{Asset: "lead", Symbol: "TECK", Name: "Timbal", Unit: "per saham acuan"},
	{Asset: "cotton", Symbol: "CT=F", Name: "Kapas", Unit: "per pon"},
	{Asset: "lumber", Symbol: "LBR=F", Name: "Kayu", Unit: "per 1.000 board feet"},
	{Asset: "cokingcoal", Symbol: "AMR", Name: "Batu Bara Kokas", Unit: "per saham acuan"},
	{Asset: "chili", Name: "Cabai Merah", Unit: "per kg", PIHPSID: "com_13"},
	{Asset: "shallot", Name: "Bawang Merah", Unit: "per kg", PIHPSID: "com_11"},
	{Asset: "garlic", Name: "Bawang Putih", Unit: "per kg", PIHPSID: "com_12"},
	{Asset: "rawit", Name: "Cabai Rawit", Unit: "per kg", PIHPSID: "com_16"},
}

// GetCryptoPrices returns the coins listed on the Crypto page. CoinGecko's
// markets endpoint already carries name, ticker, logo and market cap, so the
// page does not need a hardcoded coin table — dan seluruh sepuluh koin datang
// dari satu permintaan, bukan satu permintaan per koin.
func (s *MarketService) GetCryptoPrices() ([]map[string]any, error) {
	return cachedQuotes(cryptoCacheKey, fetchCryptoMarkets)
}

// GetForexPrices returns one <code>/IDR rate per entry in forexCurrencies.
func (s *MarketService) GetForexPrices() ([]map[string]any, error) {
	return cachedQuotes(forexCacheKey, fetchForexRates)
}

// GetSaham returns the twenty rows on the Saham page — ten Indonesian listings
// and ten foreign ones — in one request, each tagged with its `market`.
func (s *MarketService) GetSaham() ([]map[string]any, error) {
	return cachedQuotes(sahamCacheKey, func() ([]map[string]any, error) {
		return fetchYahooQuotes(sahamInstruments)
	})
}

// GetKomoditas returns the thirty rows on the Komoditas page. Dua puluh enam
// baris datang dari Yahoo Finance dan empat harga pangan dari Bank Indonesia
// (PIHPS); keduanya digabung di fetchKomoditasQuotes supaya urutan tabel tetap
// sesuai komoditasInstruments.
func (s *MarketService) GetKomoditas() ([]map[string]any, error) {
	return cachedQuotes(komoditasCacheKey, func() ([]map[string]any, error) {
		return fetchKomoditasQuotes(komoditasInstruments)
	})
}

// cachedQuotes menyimpan daftar kuotasi satu jam. Empat daftar (koin, kurs,
// saham, komoditas) memakai jalur yang sama persis, jadi bentuk cache-nya hanya
// ditulis sekali di sini.
//
// Setiap hasil yang berhasil diambil ikut disalin ke tabel market_cache. Saat
// upstream membalas 429/503, salinan terakhir itu yang dikembalikan supaya
// tabel tidak berubah menjadi pesan error kosong.
func cachedQuotes(key string, fetch func() ([]map[string]any, error)) ([]map[string]any, error) {
	fresh := !facades.Cache().Has(key)

	cached, err := facades.Cache().Remember(key, marketCacheTTL, func() (interface{}, error) {
		return fetch()
	})
	if err != nil {
		var stored []map[string]any
		if loadMarketCache(key, &stored) && len(stored) > 0 {
			facades.Log().Warningf("daftar %s gagal diambil (%v), memakai cache terakhir dari database", key, err)
			return stored, nil
		}

		return nil, err
	}

	quotes, ok := cached.([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected cache payload of type %T for %s", cached, key)
	}

	if fresh {
		persistMarketCache(key, quotes)
	}

	return quotes, nil
}

func fetchCryptoMarkets() ([]map[string]any, error) {
	payload, err := fetchJsonArray(coinGeckoMarketsURL())
	if err != nil {
		return nil, err
	}

	coins := make([]map[string]any, 0, len(payload))
	for _, entry := range payload {
		coin, ok := toMap(entry)
		if !ok {
			continue
		}

		coins = append(coins, map[string]any{
			"id":         toStr(coin["id"]),
			"symbol":     strings.ToUpper(toStr(coin["symbol"])),
			"name":       toStr(coin["name"]),
			"image":      toStr(coin["image"]),
			"price_usd":  toFloat64(coin["current_price"]),
			"change_24h": toFloat64(coin["price_change_percentage_24h"]),
			"market_cap": toFloat64(coin["market_cap"]),
		})
	}

	if len(coins) == 0 {
		return nil, fmt.Errorf("CoinGecko markets returned no coin")
	}

	return coins, nil
}

// fetchForexRates queries every currency at once: a whole page load should not
// wait for six Yahoo requests laid out end to end. Yahoo is only hit six times
// per cache window, so this stays far below its rate limit.
func fetchForexRates() ([]map[string]any, error) {
	rates := make([]map[string]any, len(forexCurrencies))
	failures := make([]error, len(forexCurrencies))

	// Sepuluh pasangan sekaligus membuat Yahoo membalas 429 pada sebagian
	// permintaan, jadi batasnya sama dengan daftar saham dan komoditas.
	slots := make(chan struct{}, yahooQuoteWorkers)
	var wg sync.WaitGroup

	for i, currency := range forexCurrencies {
		wg.Add(1)
		go func(index int, code string) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()

			rates[index], failures[index] = fetchYahooIdrRate(code)
		}(i, currency.Code)
	}
	wg.Wait()

	available := make([]map[string]any, 0, len(forexCurrencies))
	for i, currency := range forexCurrencies {
		if failures[i] != nil {
			// Satu pasangan yang hilang tidak perlu menjatuhkan seluruh halaman;
			// penyebabnya tetap dicatat supaya ketahuan kalau simbolnya berubah.
			facades.Log().Warningf("gagal mengambil kurs %s/IDR: %v", currency.Code, failures[i])
			continue
		}

		rates[i]["code"] = currency.Code
		rates[i]["name"] = currency.Name
		available = append(available, rates[i])
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("semua sumber kurs gagal: %v", failures[0])
	}

	return available, nil
}

// fetchKomoditasQuotes menggabungkan dua sumber halaman Komoditas: kontrak
// Yahoo Finance dan harga pangan Bank Indonesia. Keduanya boleh gagal
// sendiri-sendiri — satu baris yang hilang lebih baik daripada halaman kosong —
// dan urutan hasilnya tetap mengikuti komoditasInstruments.
func fetchKomoditasQuotes(instruments []yahooInstrument) ([]map[string]any, error) {
	quotes := make(map[string]map[string]any, len(instruments))

	yahooQuotes, err := fetchYahooQuotes(instruments)
	if err != nil {
		facades.Log().Warningf("kuotasi Yahoo untuk komoditas gagal: %v", err)
	}
	for _, quote := range yahooQuotes {
		quotes[toStr(quote["asset"])] = quote
	}

	for asset, quote := range fetchPihpsQuotes(instruments) {
		quotes[asset] = quote
	}

	available := make([]map[string]any, 0, len(instruments))
	for _, instrument := range instruments {
		if quote, ok := quotes[instrument.Asset]; ok {
			available = append(available, quote)
		}
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("semua sumber komoditas gagal")
	}

	return available, nil
}

// fetchYahooQuotes queries the instruments concurrently and drops the ones that
// failed. Hanya kegagalan seluruhnya yang dianggap error, sama seperti daftar
// kurs. Instrumen ber-PIHPSID dilewati: sumbernya bukan Yahoo, dan
// fetchKomoditasQuotes yang mengisinya.
//
// Permintaannya dibatasi sejumlah `yahooQuoteWorkers` sekaligus: halaman Saham
// meminta dua puluh simbol dalam satu kali muat, dan menembakkan dua puluh
// permintaan berbarengan ke Yahoo membuat sebagian dibalas 429 sehingga barisnya
// hilang tanpa pesan. Dengan batas ini urutannya berjalan bergelombang, dan
// seluruh daftar tetap selesai dalam hitungan ratusan milidetik karena hasilnya
// di-cache satu jam.
func fetchYahooQuotes(instruments []yahooInstrument) ([]map[string]any, error) {
	quotes := make([]map[string]any, len(instruments))
	failures := make([]error, len(instruments))

	slots := make(chan struct{}, yahooQuoteWorkers)
	var wg sync.WaitGroup

	for i, instrument := range instruments {
		if instrument.PIHPSID != "" || instrument.Symbol == "" {
			continue
		}

		wg.Add(1)
		go func(index int, symbol string) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()

			quotes[index], failures[index] = fetchYahooQuote(symbol)
		}(i, instrument.Symbol)
	}
	wg.Wait()

	available := make([]map[string]any, 0, len(instruments))
	for i, instrument := range instruments {
		if instrument.PIHPSID != "" || instrument.Symbol == "" {
			continue
		}
		if failures[i] != nil {
			// Satu simbol yang hilang tidak perlu menjatuhkan seluruh halaman;
			// penyebabnya tetap dicatat supaya ketahuan kalau simbolnya berubah.
			facades.Log().Warningf("gagal mengambil kuotasi %s: %v", instrument.Symbol, failures[i])
			continue
		}

		quotes[i]["asset"] = instrument.Asset
		quotes[i]["symbol"] = instrument.Symbol
		quotes[i]["name"] = instrument.Name
		if instrument.Unit != "" {
			quotes[i]["unit"] = instrument.Unit
		}
		if instrument.Market != "" {
			quotes[i]["market"] = instrument.Market
		}

		available = append(available, quotes[i])
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("semua kuotasi Yahoo Finance gagal: %v", firstFailure(failures))
	}

	return available, nil
}

// firstFailure mengambil kesalahan pertama yang tidak nil, untuk pesan error
// ketika sebuah daftar gagal seluruhnya.
func firstFailure(failures []error) error {
	for _, err := range failures {
		if err != nil {
			return err
		}
	}

	return fmt.Errorf("tidak ada instrumen yang bisa diminta")
}

// fetchPihpsQuotes mengambil harga terkini seluruh instrumen PIHPS sekaligus.
// Permintaannya dibatasi pihpsChunkWorkers supaya empat baris selesai dalam
// satu gelombang tanpa membanjiri server publik.
func fetchPihpsQuotes(instruments []yahooInstrument) map[string]map[string]any {
	quotes := make(map[string]map[string]any)

	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, pihpsChunkWorkers)

	for _, instrument := range instruments {
		if instrument.PIHPSID == "" {
			continue
		}

		wg.Add(1)
		go func(instrument yahooInstrument) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()

			quote, err := fetchPihpsQuote(instrument)
			if err != nil {
				facades.Log().Warningf("gagal mengambil harga PIHPS %s: %v", instrument.Name, err)
				return
			}

			mu.Lock()
			quotes[instrument.Asset] = quote
			mu.Unlock()
		}(instrument)
	}
	wg.Wait()

	return quotes
}

// fetchPihpsQuote membaca seri sepuluh hari terakhir dari Panel Harga Bank
// Indonesia dan memakai titik terakhir sebagai harga terkini, dengan titik
// sebelumnya sebagai pembanding. Sepuluh hari dipilih supaya akhir pekan dan
// hari libur tidak membuat barisnya kosong.
func fetchPihpsQuote(instrument yahooInstrument) (map[string]any, error) {
	end := time.Now()
	points, err := fetchPihpsSeries(instrument.PIHPSID, end.AddDate(0, 0, -10), end, pihpsDailyReport)
	if err != nil {
		return nil, err
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("PIHPS tidak mengembalikan harga untuk %s", instrument.Name)
	}

	price := toFloat64(points[len(points)-1]["price"])
	previous := price
	if len(points) >= 2 {
		previous = toFloat64(points[len(points)-2]["price"])
	}

	change := 0.0
	if previous > 0 {
		change = (price - previous) / previous * 100
	}

	return map[string]any{
		"asset":          instrument.Asset,
		"symbol":         strings.ToUpper(instrument.Asset),
		"name":           instrument.Name,
		"unit":           instrument.Unit,
		"price":          price,
		"change_percent": change,
		"previous_close": previous,
		"currency":       "IDR",
		"source":         "pihps",
	}, nil
}

// pihpsInstrumentByAsset mencari instrumen komoditas yang harganya dari PIHPS.
func pihpsInstrumentByAsset(asset string) (yahooInstrument, bool) {
	for _, instrument := range komoditasInstruments {
		if instrument.Asset == asset && instrument.PIHPSID != "" {
			return instrument, true
		}
	}

	return yahooInstrument{}, false
}

// isPihpsAsset melaporkan apakah sebuah aset riwayat dilayani Panel Harga Bank
// Indonesia, bukan Yahoo Finance.
func isPihpsAsset(asset string) bool {
	_, ok := pihpsInstrumentByAsset(asset)

	return ok
}

// fetchYahooIdrRate reads a single <code>/IDR quote.
func fetchYahooIdrRate(code string) (map[string]any, error) {
	pair := code + "/IDR"

	quote, err := fetchYahooQuote(yahooForexSymbol(code))
	if err != nil {
		return nil, fmt.Errorf("kurs %s tidak tersedia di Yahoo Finance: %w", pair, err)
	}

	return map[string]any{
		"asset":          forexAsset(code),
		"pair":           pair,
		"rate":           quote["price"],
		"change_percent": quote["change_percent"],
		"previous_close": quote["previous_close"],
		"source":         quote["source"],
	}, nil
}

// Yahoo Finance melaporkan kopi dan biji-bijian (KC=F, ZW=F, ZC=F, ZS=F) dalam
// satuan sen — blok meta-nya bertanda currency "USX" — sedangkan kontrak lain dan
// seluruh halaman lain di aplikasi ini memakai dolar. Tanpa dikonversi, kopi
// 305,55 sen/pon akan tercetak sebagai "$305,56". Faktor ini dipakai harga live
// maupun harga penutupan harian supaya tabel dan grafiknya tidak berbeda seratus
// kali.
func yahooPriceScale(currency string) float64 {
	if currency == "USX" {
		return 0.01
	}

	return 1
}

// fetchYahooQuote reads one Yahoo Finance quote from the chart meta block. Yahoo
// answers with the previous close while a market is shut, so a zero price means
// the symbol itself is wrong — bukan pasarnya yang tutup.
func fetchYahooQuote(symbol string) (map[string]any, error) {
	payload, err := fetchJson(fmt.Sprintf(yahooQuoteURL, url.PathEscape(symbol)))
	if err != nil {
		return nil, err
	}

	chart, ok := toMap(payload["chart"])
	if !ok {
		return nil, fmt.Errorf("unexpected Yahoo Finance payload for %s", symbol)
	}

	results, ok := toSlice(chart["result"])
	if !ok || len(results) == 0 {
		return nil, fmt.Errorf("Yahoo Finance returned no result for %s", symbol)
	}

	first, ok := toMap(results[0])
	if !ok {
		return nil, fmt.Errorf("unexpected Yahoo Finance result for %s", symbol)
	}

	meta, ok := toMap(first["meta"])
	if !ok {
		return nil, fmt.Errorf("Yahoo Finance result has no meta section for %s", symbol)
	}

	price := toFloat64(meta["regularMarketPrice"])
	if price == 0 {
		// Sebagian kontrak tipis (mis. LED=F untuk timbal) hanya membawa
		// penutupan sebelumnya tanpa regularMarketPrice. Harga itu masih
		// berguna untuk tabel, jadi dipakai alih-alih menghilangkan barisnya.
		price = toFloat64(meta["chartPreviousClose"])
	}
	if price == 0 {
		return nil, fmt.Errorf("Yahoo Finance has no price for %s", symbol)
	}

	currency := toStr(meta["currency"])
	scale := yahooPriceScale(currency)
	if scale != 1 {
		// Harganya sudah jadi dolar, jadi mata uangnya juga harus ikut berubah.
		currency = "USD"
	}

	return map[string]any{
		"price":          price * scale,
		"change_percent": toFloat64(meta["regularMarketChangePercent"]),
		"previous_close": toFloat64(meta["chartPreviousClose"]) * scale,
		"currency":       currency,
		"source":         "yahoo_finance",
	}, nil
}

// GetHistoricalPrices returns a daily close series for one asset, plus the
// upstream that actually served it. Koin apa pun yang terdaftar di cryptoCoins
// dilayani lewat CoinGecko market_chart dengan Yahoo sebagai cadangan; emas dan
// USD/IDR selalu datang dari Yahoo Finance.
//
// Hasil per kombinasi asset+days di-cache satu jam. Yang benar-benar diambil dari
// upstream hanyalah satu seri harian per aset dan rentang upstream (lihat
// historySpan), lalu dipotong sesuai permintaan: CoinGecko anonim membatasi laju
// permintaan, jadi menjelajah tombol 3H..10T tidak boleh berarti delapan
// permintaan berturut-turut.
//
// Jika upstream tetap gagal, salinan terakhir dari tabel market_cache dipakai
// supaya grafik menampilkan data lama, bukan kotak kosong.
func (s *MarketService) GetHistoricalPrices(asset string, days int) ([]map[string]interface{}, string, string, error) {
	days = clampHistoryDays(days)

	// 24J dan 3H sengaja memakai titik per jam. Tanpa cabang ini keduanya
	// hanya mengembalikan satu sampai tiga titik harian dan grafiknya terlihat
	// kosong walaupun datanya ada.
	if days <= HistoryHourlyMaxDays {
		return s.getHourlyHistory(asset, days)
	}

	cacheKey := fmt.Sprintf("market_history_%s_%d", asset, days)

	if cached := facades.Cache().Get(cacheKey); cached != nil {
		if series, ok := cached.(historySeries); ok {
			return series.Points, series.Source, "daily", nil
		}
	}

	full, err := s.fullDailySeries(asset, days)
	if err != nil {
		if series, ok := cachedHistorySeries(cacheKey); ok {
			facades.Log().Warningf("riwayat %s gagal diambil (%v), memakai cache terakhir dari database", asset, err)
			return series.Points, series.Source, "daily", nil
		}

		return nil, "", "", err
	}

	series := historySeries{Source: full.Source, Points: lastNDays(full.Points, days)}
	if err := facades.Cache().Put(cacheKey, series, historyCacheTTL); err != nil {
		return nil, "", "", err
	}

	persistMarketCache(cacheKey, series)

	return series.Points, series.Source, "daily", nil
}

// getHourlyHistory melayani rentang 24J dan 3H. Koin, emas, saham, kurs, dan
// komoditas semuanya diambil dari chart Yahoo interval 1h supaya bentuk
// datanya seragam. PIHPS tidak punya data per jam, jadi empat harga pangan
// tetap memakai seri harian dan grafiknya menampilkan titik seadanya.
//
// Sebagian kontrak (mis. uranium UX=F dan seng ZNC=F) tidak punya bar per jam
// di Yahoo. Kalau permintaan 1h gagal atau tidak menghasilkan garis, seri
// harian dipakai sebagai cadangan supaya halaman tidak berakhir dengan 502;
// interval yang dikembalikan ikut jujur menyebut "daily".
func (s *MarketService) getHourlyHistory(asset string, days int) ([]map[string]interface{}, string, string, error) {
	// v2: kunci baru supaya seri per jam lama yang terpotong di jam buka pasar
	// tidak dipakai lagi dari cache memori maupun market_cache.
	cacheKey := fmt.Sprintf("market_history_hourly_v2_%s_%d", asset, days)

	if cached := facades.Cache().Get(cacheKey); cached != nil {
		if series, ok := cached.(historySeries); ok {
			interval := series.Interval
			if interval == "" {
				interval = "1h"
			}
			return series.Points, series.Source, interval, nil
		}
	}

	var series historySeries

	if isPihpsAsset(asset) {
		full, err := s.fullDailySeries(asset, days)
		if err != nil {
			if stored, ok := cachedHistorySeries(cacheKey); ok {
				facades.Log().Warningf("riwayat %s gagal diambil (%v), memakai cache terakhir dari database", asset, err)
				return stored.Points, stored.Source, stored.Interval, nil
			}

			return nil, "", "", err
		}
		series = historySeries{Source: full.Source, Points: lastNDays(full.Points, days), Interval: "daily"}
	} else {
		symbol, ok := hourlyYahooSymbol(asset)
		if !ok {
			return nil, "", "", fmt.Errorf("asset riwayat tidak dikenal: %q", asset)
		}

		points, hourlyErr := fetchYahooHourlyHistory(symbol, days)
		if hourlyErr == nil && len(points) >= 2 {
			series = historySeries{Source: "yahoo_finance", Points: points, Interval: "1h"}
		} else {
			if hourlyErr != nil {
				facades.Log().Warningf("riwayat per jam %s gagal (%v), memakai seri harian", asset, hourlyErr)
			}
			full, err := s.fullDailySeries(asset, days)
			if err != nil {
				if stored, ok := cachedHistorySeries(cacheKey); ok {
					facades.Log().Warningf("riwayat %s gagal diambil (%v), memakai cache terakhir dari database", asset, err)
					return stored.Points, stored.Source, stored.Interval, nil
				}

				return nil, "", "", err
			}
			series = historySeries{Source: full.Source, Points: lastNDays(full.Points, days), Interval: "daily"}
		}
	}

	if err := facades.Cache().Put(cacheKey, series, historyCacheTTL); err != nil {
		return nil, "", "", err
	}

	persistMarketCache(cacheKey, series)

	return series.Points, series.Source, series.Interval, nil
}

// hourlyYahooSymbol memilih ticker untuk seri per jam. Koin memakai ticker
// Yahoo-nya sendiri karena endpoint market_chart CoinGecko tidak dipanggil di
// jalur ini; aset lain memakai pemetaan yang sudah ada.
func hourlyYahooSymbol(asset string) (string, bool) {
	if coin, ok := cryptoCoinByID(asset); ok {
		return coin.Yahoo, true
	}

	return historyYahooSymbol(asset)
}

// historySeries adalah seri harga harian beserta sumber yang benar-benar
// melayaninya. Sumbernya dibawa bersama datanya supaya balasan API tidak
// berbohong ketika CoinGecko membatasi laju permintaan lalu Yahoo menggantikan.
type historySeries struct {
	Source   string
	Interval string
	Points   []map[string]interface{}
}

// fullDailySeries mengambil seri harian terpanjang yang dibutuhkan untuk sebuah
// rentang dan menyimpannya satu jam. Semua rentang yang lebih pendek dipotong
// dari seri ini, sehingga satu permintaan melayani banyak tombol rentang.
//
// Seri penuh ikut disalin ke tabel market_cache. Kalau upstream gagal, seri
// penuh terakhir dipakai lagi - termasuk untuk rentang yang belum pernah
// diminta sebelumnya - lalu dipotong oleh pemanggilnya.
func (s *MarketService) fullDailySeries(asset string, days int) (historySeries, error) {
	span := historySpan(days)
	// PIHPS hanya menyimpan satu tahun seri harian; rentang 5T/10T tetap
	// memakai seri yang sama dan grafiknya memotong lewat sumbu X. Tanpa
	// normalisasi ini kunci cache ikut berbeda dan satu seri yang sama
	// diunduh dua kali.
	if isPihpsAsset(asset) {
		span = historySpanYear
	}
	cacheKey := fmt.Sprintf("market_history_full_%s_%s", asset, span)
	fresh := !facades.Cache().Has(cacheKey)

	cached, err := facades.Cache().Remember(cacheKey, historyCacheTTL, func() (interface{}, error) {
		return fetchDailySeries(asset, span)
	})
	if err != nil {
		if stored, ok := cachedHistorySeries(cacheKey); ok {
			facades.Log().Warningf("seri penuh %s gagal diambil (%v), memakai cache terakhir dari database", asset, err)
			return stored, nil
		}

		return historySeries{}, err
	}

	series, ok := cached.(historySeries)
	if !ok {
		return historySeries{}, fmt.Errorf("unexpected market history cache payload of type %T", cached)
	}

	if fresh {
		persistMarketCache(cacheKey, series)
	}

	return series, nil
}

// fetchDailySeries memilih sumber untuk sebuah aset. Koin dilayani CoinGecko
// selama tier anonimnya masih membalas; begitu CoinGecko membatasi laju
// permintaan (429), ticker Yahoo koin itu dipakai sebagai gantinya supaya grafik
// tetap terisi alih-alih memunculkan 502. Aset lain selalu dari Yahoo.
func fetchDailySeries(asset string, span string) (historySeries, error) {
	coin, isCoin := cryptoCoinByID(asset)

	switch {
	case isCoin && span == historySpanYear:
		points, err := fetchCoinGeckoHistory(coin.ID)
		if err == nil {
			return historySeries{Source: "coingecko", Points: points}, nil
		}

		facades.Log().Warningf("CoinGecko gagal untuk %s, beralih ke Yahoo: %v", coin.ID, err)

		points, fallbackErr := fetchYahooHistory(coin.Yahoo, span)
		if fallbackErr != nil {
			return historySeries{}, fmt.Errorf("CoinGecko: %w; Yahoo: %v", err, fallbackErr)
		}

		return historySeries{Source: "yahoo_finance", Points: points}, nil
	case isCoin:
		// CoinGecko anonim membalas 401 untuk days>365, jadi rentang panjang
		// memakai ticker Yahoo koin tersebut.
		points, err := fetchYahooHistory(coin.Yahoo, span)

		return historySeries{Source: "yahoo_finance", Points: points}, err
	}

	// Empat harga pangan (cabai, bawang) datang dari Panel Harga Bank Indonesia,
	// bukan Yahoo Finance, jadi serinya diambil dari endpoint yang sama dengan
	// harga terkini.
	if instrument, ok := pihpsInstrumentByAsset(asset); ok {
		points, err := fetchPihpsSeries(instrument.PIHPSID, time.Now().AddDate(-1, 0, 0), time.Now(), pihpsDailyReport)

		return historySeries{Source: "pihps", Points: points}, err
	}

	// Semua aset non-koin lainnya — indeks saham, kontrak berjangka, kurs, dan
	// emas — datang dari Yahoo Finance lewat ticker yang dicatat di daftarnya
	// masing-masing.
	symbol, ok := historyYahooSymbol(asset)
	if !ok {
		return historySeries{}, fmt.Errorf("asset riwayat tidak dikenal: %q", asset)
	}

	points, err := fetchYahooHistory(symbol, span)

	return historySeries{Source: "yahoo_finance", Points: points}, err
}

// fetchCoinGeckoHistory reads market_chart for one coin and collapses the
// intraday points into a single close per date.
func fetchCoinGeckoHistory(coinID string) ([]map[string]interface{}, error) {
	payload, err := fetchJsonWithRetry(fmt.Sprintf(coinGeckoHistoryURL, coinID))
	if err != nil {
		return nil, err
	}

	prices, ok := toSlice(payload["prices"])
	if !ok || len(prices) == 0 {
		return nil, fmt.Errorf("CoinGecko market_chart %s tidak mengembalikan array prices", coinID)
	}

	series := make([]map[string]interface{}, 0, len(prices))
	lastDate := ""

	for _, entry := range prices {
		point, ok := toSlice(entry)
		if !ok || len(point) < 2 {
			continue
		}

		seconds := int64(toFloat64(point[0]) / 1000)
		price := toFloat64(point[1])
		if seconds <= 0 || price <= 0 {
			continue
		}

		date := time.Unix(seconds, 0).UTC().Format(historyDateLayout)
		if date == lastDate && len(series) > 0 {
			// Titik terakhir di hari yang sama menggantikan yang sebelumnya.
			series[len(series)-1]["price"] = price
			series[len(series)-1]["ts"] = seconds * 1000
			continue
		}

		series = append(series, map[string]interface{}{"date": date, "price": price, "ts": seconds * 1000})
		lastDate = date
	}

	if len(series) == 0 {
		return nil, fmt.Errorf("CoinGecko market_chart %s tidak berisi titik harga yang bisa dipakai", coinID)
	}

	return series, nil
}

// fetchYahooHistory reads the daily close series from a Yahoo Finance chart.
// span adalah rentang Yahoo yang valid (1y, 10y, …) dan sekaligus kunci cache.
func fetchYahooHistory(symbol string, span string) ([]map[string]interface{}, error) {
	payload, err := fetchJsonWithRetry(fmt.Sprintf(yahooChartURL, symbol, span))
	if err != nil {
		return nil, err
	}

	chart, ok := toMap(payload["chart"])
	if !ok {
		return nil, fmt.Errorf("payload Yahoo Finance untuk %s tidak punya bagian chart", symbol)
	}

	results, ok := toSlice(chart["result"])
	if !ok || len(results) == 0 {
		return nil, fmt.Errorf("Yahoo Finance tidak mengembalikan hasil untuk %s", symbol)
	}

	first, ok := toMap(results[0])
	if !ok {
		return nil, fmt.Errorf("hasil Yahoo Finance untuk %s tidak valid", symbol)
	}

	timestamps, ok := toSlice(first["timestamp"])
	if !ok || len(timestamps) == 0 {
		return nil, fmt.Errorf("Yahoo Finance tidak mengembalikan timestamp untuk %s", symbol)
	}

	closes, err := yahooCloses(first)
	if err != nil {
		return nil, fmt.Errorf("%w (symbol %s)", err, symbol)
	}

	// Zona waktu bursa dipakai supaya label tanggal tidak bergeser satu hari.
	location := yahooLocation(first)

	// Kopi dan biji-bijian juga datang dalam sen di seri harian, jadi faktor yang
	// sama dipakai supaya grafiknya sejalan dengan harga di tabel.
	scale := yahooPriceScale(yahooMetaCurrency(first))

	series := make([]map[string]interface{}, 0, len(timestamps))
	for index, entry := range timestamps {
		if index >= len(closes) {
			break
		}

		seconds := int64(toFloat64(entry))
		price := toFloat64(closes[index]) * scale
		if seconds <= 0 || price <= 0 {
			// Hari libur atau sesi tanpa transaksi: Yahoo mengirim null.
			continue
		}

		series = append(series, map[string]interface{}{
			"date":  time.Unix(seconds, 0).In(location).Format(historyDateLayout),
			"price": price,
			"ts":    seconds * 1000,
		})
	}

	if len(series) == 0 {
		return nil, fmt.Errorf("Yahoo Finance tidak punya harga penutupan untuk %s", symbol)
	}

	return series, nil
}

// fetchYahooHourlyHistory membaca chart Yahoo interval 1h untuk rentang 5d,
// lalu memotong titik terakhir `days` hari berdasarkan timestamp terbaru.
// Memakai timestamp terbaru (bukan jam sekarang) membuat akhir pekan dan
// pasar yang sudah tutup tetap menampilkan sesi terakhir alih-alih grafik
// kosong.
func fetchYahooHourlyHistory(symbol string, days int) ([]map[string]interface{}, error) {
	payload, err := fetchJsonWithRetry(fmt.Sprintf(yahooHourlyChartURL, url.PathEscape(symbol)))
	if err != nil {
		return nil, err
	}

	chart, ok := toMap(payload["chart"])
	if !ok {
		return nil, fmt.Errorf("payload Yahoo Finance untuk %s tidak punya bagian chart", symbol)
	}

	results, ok := toSlice(chart["result"])
	if !ok || len(results) == 0 {
		return nil, fmt.Errorf("Yahoo Finance tidak mengembalikan hasil untuk %s", symbol)
	}

	first, ok := toMap(results[0])
	if !ok {
		return nil, fmt.Errorf("hasil Yahoo Finance untuk %s tidak valid", symbol)
	}

	timestamps, ok := toSlice(first["timestamp"])
	if !ok || len(timestamps) == 0 {
		return nil, fmt.Errorf("Yahoo Finance tidak mengembalikan timestamp untuk %s", symbol)
	}

	closes, err := yahooCloses(first)
	if err != nil {
		return nil, fmt.Errorf("%w (symbol %s)", err, symbol)
	}

	location := yahooLocation(first)
	scale := yahooPriceScale(yahooMetaCurrency(first))

	type hourlyPoint struct {
		seconds int64
		price   float64
	}

	points := make([]hourlyPoint, 0, len(timestamps))
	for index, entry := range timestamps {
		if index >= len(closes) {
			break
		}

		seconds := int64(toFloat64(entry))
		price := toFloat64(closes[index]) * scale
		if seconds <= 0 || price <= 0 {
			continue
		}

		points = append(points, hourlyPoint{seconds: seconds, price: price})
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("Yahoo Finance tidak punya harga per jam untuk %s", symbol)
	}

	// Jendela 24J/3H dihitung dari waktu sekarang, bukan dari titik data
	// terakhir. Titik terakhir bisa tertinggal di jam buka pasar, sehingga
	// potongan "24 jam terakhir" ikut bergeser dan grafik hanya terisi sejak
	// sesi itu dimulai. Dengan patokan sekarang, rentangnya selalu 24 jam
	// penuh ke belakang dan sumbu X di frontend (yang juga dipatok ke
	// sekarang) berakhir di titik yang sama.
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour).Unix()

	// Akhir pekan atau hari libur bisa membuat tidak ada satu pun bar dalam
	// 24 jam terakhir. Untuk kasus itu seri sesi terakhir tetap ditampilkan
	// supaya grafiknya tidak kosong.
	if len(points) > 0 && points[len(points)-1].seconds < cutoff {
		cutoff = points[len(points)-1].seconds - int64(days)*24*60*60
	}

	series := make([]map[string]interface{}, 0, len(points))
	for _, point := range points {
		if point.seconds < cutoff {
			continue
		}

		series = append(series, map[string]interface{}{
			"date":  time.Unix(point.seconds, 0).In(location).Format(historyHourlyLayout),
			"price": point.price,
			"ts":    point.seconds * 1000,
		})
	}

	if len(series) == 0 {
		return nil, fmt.Errorf("Yahoo Finance tidak punya harga per jam untuk %s", symbol)
	}

	return series, nil
}

// fetchPihpsSeries membaca seri harga satu komoditas pangan dari Panel Harga
// Bank Indonesia. Rentang panjang dipecah karena grid PIHPS menghitung seluruh
// provinsi sekaligus: satu permintaan setahun bisa 40 detik, sedangkan 12
// potongan 30 hari selesai sekitar 3 detik bila dijalankan bergelombang.
func fetchPihpsSeries(comcatID string, start time.Time, end time.Time, reportType int) ([]map[string]interface{}, error) {
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	if end.Before(start) {
		return nil, fmt.Errorf("rentang PIHPS terbalik")
	}

	spanDays := int(end.Sub(start).Hours()/24) + 1
	if spanDays <= pihpsChunkDays {
		return fetchPihpsSeriesWindow(comcatID, start, end, reportType)
	}

	type pihpsWindow struct {
		start time.Time
		end   time.Time
	}

	windows := make([]pihpsWindow, 0, spanDays/pihpsChunkDays+1)
	for cursor := start; !cursor.After(end); cursor = cursor.AddDate(0, 0, pihpsChunkDays) {
		stop := cursor.AddDate(0, 0, pihpsChunkDays-1)
		if stop.After(end) {
			stop = end
		}
		windows = append(windows, pihpsWindow{start: cursor, end: stop})
	}

	merged := make(map[string]map[string]interface{})
	failures := make([]error, len(windows))
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, pihpsChunkWorkers)

	for i, window := range windows {
		wg.Add(1)
		go func(index int, window pihpsWindow) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()

			points, err := fetchPihpsSeriesWindow(comcatID, window.start, window.end, reportType)
			if err != nil {
				failures[index] = err
				return
			}

			mu.Lock()
			for _, point := range points {
				merged[toStr(point["date"])] = point
			}
			mu.Unlock()
		}(i, window)
	}
	wg.Wait()

	points := make([]map[string]interface{}, 0, len(merged))
	for _, point := range merged {
		points = append(points, point)
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("PIHPS tidak punya harga yang bisa dipakai untuk %s: %v", comcatID, firstFailure(failures))
	}

	for i, err := range failures {
		if err != nil {
			facades.Log().Warningf(
				"potongan PIHPS %s %s..%s gagal: %v",
				comcatID,
				windows[i].start.Format(historyDateLayout),
				windows[i].end.Format(historyDateLayout),
				err,
			)
		}
	}

	sort.SliceStable(points, func(i, j int) bool {
		return toStr(points[i]["date"]) < toStr(points[j]["date"])
	})

	return points, nil
}

// fetchPihpsSeriesWindow mengambil satu potongan rentang dari grid PIHPS.
// Endpoint gridnya mengembalikan satu baris per provinsi dengan kolom tanggal
// berformat dd/mm/yyyy; baris "Semua Provinsi" adalah rata-rata nasional yang
// dipakai di sini, dan kolom yang bernilai "-" (data belum masuk) dilewati.
func fetchPihpsSeriesWindow(comcatID string, start time.Time, end time.Time, reportType int) ([]map[string]interface{}, error) {
	params := url.Values{}
	params.Set("price_type_id", "1")
	params.Set("comcat_id", comcatID)
	params.Set("province_id", "")
	params.Set("regency_id", "")
	params.Set("showKota", "false")
	params.Set("showPasar", "false")
	params.Set("tipe_laporan", strconv.Itoa(reportType))
	params.Set("start_date", start.Format(historyDateLayout))
	params.Set("end_date", end.Format(historyDateLayout))

	payload, err := fetchPihpsJson(pihpsGridURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}

	rows, ok := toSlice(payload["data"])
	if !ok || len(rows) == 0 {
		return nil, fmt.Errorf("PIHPS tidak mengembalikan baris data untuk %s", comcatID)
	}

	nationwide := map[string]any(nil)
	for _, entry := range rows {
		row, ok := toMap(entry)
		if !ok {
			continue
		}
		if toStr(row["name"]) == "Semua Provinsi" {
			nationwide = row
			break
		}
	}
	if nationwide == nil {
		return nil, fmt.Errorf("PIHPS tidak punya baris Semua Provinsi untuk %s", comcatID)
	}

	points := make([]map[string]interface{}, 0, len(nationwide))
	for key, raw := range nationwide {
		date, parseErr := time.Parse("02/01/2006", key)
		if parseErr != nil {
			continue
		}

		price, priceErr := pihpsPrice(raw)
		if priceErr != nil {
			continue
		}

		// PIHPS melaporkan harga per hari tanpa jam, jadi tengah malam waktu
		// lokal dipakai sebagai timestamp agar titiknya jatuh di tanggal yang
		// sama saat digambar di browser.
		localDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)

		points = append(points, map[string]interface{}{
			"date":  date.Format(historyDateLayout),
			"price": price,
			"ts":    localDay.UnixMilli(),
		})
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("PIHPS tidak punya harga yang bisa dipakai untuk %s", comcatID)
	}

	sort.SliceStable(points, func(i, j int) bool {
		return toStr(points[i]["date"]) < toStr(points[j]["date"])
	})

	return points, nil
}

// pihpsPrice mengubah satu sel grid PIHPS jadi angka. Nilainya string bergaya
// Indonesia ("37,150"), dan sel kosong atau "-" berarti hari itu tidak ada
// pencatatan.
func pihpsPrice(raw any) (float64, error) {
	text := strings.TrimSpace(fmt.Sprint(raw))
	if text == "" || text == "-" || text == "<nil>" {
		return 0, fmt.Errorf("sel PIHPS kosong")
	}

	text = strings.ReplaceAll(text, ",", "")
	price, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, err
	}
	if price <= 0 {
		return 0, fmt.Errorf("harga PIHPS tidak positif")
	}

	return price, nil
}

// fetchPihpsJson memanggil endpoint PIHPS dengan header yang sama seperti
// halaman webnya. Tanpa X-Requested-With dan Referer, gridnya membalas halaman
// HTML alih-alih JSON.
func fetchPihpsJson(url string) (map[string]any, error) {
	response, err := facades.Http().
		FlushHeaders().
		WithHeader("User-Agent", marketUserAgent).
		WithHeader("Referer", "https://www.bi.go.id/hargapangan/TabelHarga/PasarTradisionalKomoditas").
		WithHeader("X-Requested-With", "XMLHttpRequest").
		AcceptJSON().
		Get(url)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}

	if response.Failed() {
		return nil, fmt.Errorf("request to %s returned status %d", url, response.Status())
	}

	payload, err := response.Json()
	if err != nil {
		return nil, fmt.Errorf("could not decode response from %s: %w", url, err)
	}

	return payload, nil
}

func yahooCloses(result map[string]any) ([]any, error) {
	indicators, ok := toMap(result["indicators"])
	if !ok {
		return nil, fmt.Errorf("hasil Yahoo Finance tidak punya bagian indicators")
	}

	quotes, ok := toSlice(indicators["quote"])
	if !ok || len(quotes) == 0 {
		return nil, fmt.Errorf("hasil Yahoo Finance tidak punya array quote")
	}

	quote, ok := toMap(quotes[0])
	if !ok {
		return nil, fmt.Errorf("array quote Yahoo Finance tidak valid")
	}

	closes, ok := toSlice(quote["close"])
	if !ok || len(closes) == 0 {
		return nil, fmt.Errorf("hasil Yahoo Finance tidak punya harga close")
	}

	return closes, nil
}

// yahooLocation memakai gmtoffset dari meta Yahoo dan membangun zona tetap,
// supaya tidak bergantung pada database timezone sistem.
func yahooLocation(result map[string]any) *time.Location {
	meta, ok := toMap(result["meta"])
	if !ok {
		return time.UTC
	}

	offset := int(toFloat64(meta["gmtoffset"]))
	if offset == 0 {
		return time.UTC
	}

	return time.FixedZone("exchange", offset)
}

// yahooMetaCurrency membaca satuan harga dari meta Yahoo ("USD", "IDR", "USX").
// Dipakai seri harian untuk tahu apakah nilainya perlu dikonversi dari sen.
func yahooMetaCurrency(result map[string]any) string {
	meta, ok := toMap(result["meta"])
	if !ok {
		return ""
	}

	return toStr(meta["currency"])
}

// historySpan memilih rentang upstream untuk sebuah jumlah hari. Nilainya juga
// dipakai sebagai kunci cache seri penuh: 1B tidak perlu mengunduh data 10 tahun,
// tapi 5T dan 10T berbagi satu permintaan.
func historySpan(days int) string {
	if days <= 365 {
		return historySpanYear
	}

	return historySpanDecade
}

func clampHistoryDays(days int) int {
	if days < HistoryMinDays {
		return HistoryMinDays
	}
	if days > HistoryMaxDays {
		return HistoryMaxDays
	}

	return days
}

// lastNDays memotong seri kronologis supaya panjangnya sesuai permintaan user,
// bukan sesuai panjang rentang yang harus diminta ke upstream.
func lastNDays(series []map[string]interface{}, days int) []map[string]interface{} {
	if len(series) > days {
		return series[len(series)-days:]
	}

	return series
}

// IsHistoryAsset melaporkan apakah sebuah nama aset punya sumber riwayat. Koin
// yang terdaftar di cryptoCoins diterima apa adanya; aset lain dicari lewat
// ticker Yahoo-nya (lihat historyYahooSymbol).
func IsHistoryAsset(asset string) bool {
	if _, isCoin := cryptoCoinByID(asset); isCoin {
		return true
	}

	if isPihpsAsset(asset) {
		return true
	}

	_, ok := historyYahooSymbol(asset)

	return ok
}

// HistoryAssetNames menyebut semua asset yang diterima /api/market/history, untuk
// pesan validasi 422. Daftarnya disusun dari tabel yang sama dengan yang dipakai
// endpoint harga, jadi menambah satu instrumen otomatis muncul di sini.
func HistoryAssetNames() string {
	names := make([]string, 0, len(cryptoCoins)+len(sahamInstruments)+len(komoditasInstruments)+len(forexCurrencies)+1)

	for _, coin := range cryptoCoins {
		names = append(names, coin.ID)
	}
	for _, instrument := range sahamInstruments {
		names = append(names, instrument.Asset)
	}
	for _, instrument := range komoditasInstruments {
		names = append(names, instrument.Asset)
	}
	for _, currency := range forexCurrencies {
		names = append(names, forexAsset(currency.Code))
	}
	names = append(names, HistoryAssetGold)

	return strings.Join(names, ", ")
}

// historyYahooSymbol mencari ticker Yahoo Finance sebuah aset riwayat non-koin.
// Semua daftar yang sudah ada dipakai sebagai sumber tunggal — saham, komoditas,
// dan kurs — jadi menambah satu instrumen di halaman mana pun otomatis membuat
// grafiknya bisa diminta tanpa menyentuh kode lain. Pemindaian linear di sini
// lebih jelas daripada map kedua yang harus dijaga sinkron (sama seperti
// cryptoCoinByID).
func historyYahooSymbol(asset string) (string, bool) {
	if asset == HistoryAssetGold {
		return yahooGoldSymbol, true
	}

	for _, instrument := range sahamInstruments {
		if instrument.Asset == asset {
			return instrument.Symbol, true
		}
	}

	for _, instrument := range komoditasInstruments {
		if instrument.Asset == asset {
			return instrument.Symbol, instrument.Symbol != ""
		}
	}

	for _, currency := range forexCurrencies {
		if strings.EqualFold(currency.Code, asset) {
			return yahooForexSymbol(currency.Code), true
		}
	}

	return "", false
}

// forexAsset adalah nilai `asset` untuk sebuah kode mata uang: kode kecilnya
// sendiri, sehingga "usd" di dasbor dan baris USD di halaman Kurs menunjuk hal
// yang sama.
func forexAsset(code string) string {
	return strings.ToLower(code)
}

// yahooForexSymbol merangkai ticker pasangan FX terhadap rupiah. Bentuknya
// seragam (<kode>IDR=X), jadi ditulis di satu tempat saja.
func yahooForexSymbol(code string) string {
	return code + "IDR=X"
}

func fetchJson(url string) (map[string]any, error) {
	payload, _, err := fetchJsonStatus(url)

	return payload, err
}

// fetchJsonArray sama seperti fetchJson, tetapi untuk endpoint yang membalas
// array JSON di akar dokumen (CoinGecko /coins/markets).
func fetchJsonArray(url string) ([]any, error) {
	response, err := facades.Http().
		FlushHeaders().
		WithHeader("User-Agent", marketUserAgent).
		AcceptJSON().
		Get(url)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}

	if response.Failed() {
		return nil, fmt.Errorf("request to %s returned status %d", url, response.Status())
	}

	var payload []any
	if err := response.Bind(&payload); err != nil {
		return nil, fmt.Errorf("could not decode response from %s: %w", url, err)
	}

	return payload, nil
}

// fetchJsonStatus sama seperti fetchJson, tetapi juga mengembalikan status HTTP
// supaya pemanggil bisa membedakan kegagalan yang layak diulang.
func fetchJsonStatus(url string) (map[string]any, int, error) {
	response, err := facades.Http().
		FlushHeaders().
		WithHeader("User-Agent", marketUserAgent).
		AcceptJSON().
		Get(url)
	if err != nil {
		return nil, 0, fmt.Errorf("request to %s failed: %w", url, err)
	}

	if response.Failed() {
		return nil, response.Status(), fmt.Errorf("request to %s returned status %d", url, response.Status())
	}

	payload, err := response.Json()
	if err != nil {
		return nil, response.Status(), fmt.Errorf("could not decode response from %s: %w", url, err)
	}

	return payload, response.Status(), nil
}

// fetchJsonWithRetry mengulang permintaan yang dibatasi upstream (429) atau
// sedang tidak siap (503). Hanya jalur riwayat harga yang memakainya agar
// grafik tetap terisi walau user berpindah rentang dengan cepat.
func fetchJsonWithRetry(url string) (map[string]any, error) {
	var lastErr error

	for attempt := 0; attempt < historyFetchAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(historyRetryDelay * time.Duration(attempt))
		}

		payload, status, err := fetchJsonStatus(url)
		if err == nil {
			return payload, nil
		}

		lastErr = err
		if status != http.StatusTooManyRequests && status != http.StatusServiceUnavailable {
			break
		}
	}

	return nil, lastErr
}

func toMap(value any) (map[string]any, bool) {
	typed, ok := value.(map[string]any)
	return typed, ok
}

func toStr(value any) string {
	typed, ok := value.(string)
	if !ok {
		return ""
	}

	return typed
}

func toSlice(value any) ([]any, bool) {
	typed, ok := value.([]any)
	return typed, ok
}

func toFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint:
		return float64(typed)
	case uint64:
		return float64(typed)
	default:
		return 0
	}
}
