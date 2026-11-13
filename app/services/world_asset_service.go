package services

import (
	"math"
	"sort"

	"edukasi-finansial-goedukasi-finansial-go/app/facades"
)

const (
	// Daftar Aset Terbesar Dunia disimpan satu jam, sama seperti daftar harga lain
	// di aplikasi ini: beberapa angkanya ikut harga pasar, jadi tidak perlu
	// dihitung ulang pada tiap permintaan.
	worldAssetCacheKey = "market_world_assets"

	// Satu ton metrik sama dengan 32.150,7466 troy ounce — satuan harga emas dan
	// perak di seluruh aplikasi ini (xaus.com dan Yahoo sama-sama mengutip per
	// troy ounce).
	troyOuncePerTonne = 32150.7466

	// Satuan fisik yang mewakili seluruh aset di baris "live". Jumlahnya tetap —
	// stok yang sudah ditambang atau yang sudah beredar — sehingga angkanya hanya
	// bergerak karena harganya berubah.
	goldOuncesWorld    = 216000 * troyOuncePerTonne
	silverOuncesWorld  = 1740000 * troyOuncePerTonne
	bitcoinSupplyWorld = 19940000
	etherSupplyWorld   = 120700000
)

// WorldAsset adalah satu baris tabel halaman "Aset Terbesar Dunia". Nilainya dalam
// triliun dolar; frontend yang mengubahnya ke rupiah sesuai tombol USD/IDR.
//
// Live menandai bahwa Value dihitung dari harga pasar terkini, bukan dari angka
// estimasi tetap — halaman menyebutkannya terus terang supaya estimasi dan harga
// berjalan tidak tercampur tanpa keterangan.
type WorldAsset struct {
	Name     string  `json:"name"`
	Ticker   string  `json:"ticker"`
	Icon     string  `json:"icon"`
	Category string  `json:"category"`
	Color    string  `json:"color"`
	Value    float64 `json:"value"`
	Basis    string  `json:"basis"`
	Live     bool    `json:"live"`
}

// worldAssetSeed adalah satu baris tabel sebelum harga pasar dipasang. LiveKey dan
// LiveUnits menghubungkan baris itu ke harga yang sedang berjalan: nilainya
// dihitung ulang sebagai jumlah satuan dikali harga terkini. Baris yang keduanya
// kosong memakai Value apa adanya sebagai estimasi.
//
// LiveKey memakai nama aset yang sudah dipakai tabel lain, bukan istilah baru:
// "gold" dan "bitcoin" dari payload pasar Dasbor, "silver" dari daftar komoditas,
// dan "ethereum" dari daftar koin halaman Crypto (lihat worldLiveAssetPrices).
type worldAssetSeed struct {
	WorldAsset
	LiveKey   string
	LiveUnits float64
}

// Tabel Aset Terbesar Dunia: dua puluh aset dan kelas aset terbesar, dari real
// estate dan surat utang sampai kripto dan armada pelayaran. Angka Value di sini
// adalah estimasi global — basisnya berbeda per kategori (nilai properti
// komersial, surat utang beredar, uang beredar, cadangan terbukti, stok fisik),
// dan itu memang diakui di catatan kaki halaman supaya tidak terbaca sebagai
// peringkat resmi.
//
// Empat baris terakhir yang memakai LiveKey punya jumlah fisik yang jelas, jadi
// nilainya mengikuti harga pasar terkini alih-alih dipatok sekali. Angka
// estimasinya tetap ditulis sebagai cadangan kalau harga tidak bisa diambil.
//
// Urutan tampil tidak ditentukan di sini — worldAssetRows yang mengurutkannya
// menurun, karena harga yang bergerak bisa memindahkan baris.
var worldAssetSeeds = []worldAssetSeed{
	{
		WorldAsset: WorldAsset{
			Name: "Kapitalisasi Pasar Saham Global", Ticker: "Global Equities", Icon: "📈", Category: "Saham", Color: "#4ade80", Value: 125.0,
			Basis: "kapitalisasi pasar seluruh bursa saham dunia (estimasi indeks global MSCI ACWI)",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Properti (Real Estate)", Ticker: "Real Estate", Icon: "🏢", Category: "Lainnya", Color: "#a78bfa", Value: 100.0,
			Basis: "estimasi nilai properti komersial & investasi dunia (perkantoran, ritel, industri, hunian sewa); tanah mentah dan rumah tinggal tidak dihitung",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Obligasi Pemerintah", Ticker: "Government Bonds", Icon: "📜", Category: "Lainnya", Color: "#60a5fa", Value: 80.0,
			Basis: "surat utang pemerintah dunia yang masih beredar (outstanding)",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Obligasi Korporasi Global", Ticker: "Corporate Bonds", Icon: "🏦", Category: "Lainnya", Color: "#38bdf8", Value: 45.0,
			Basis: "surat utang korporasi dunia yang masih beredar",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Aset Asuransi Global", Ticker: "Insurance Assets", Icon: "🛡️", Category: "Lainnya", Color: "#c084fc", Value: 40.0,
			Basis: "total aset industri asuransi dunia (portofolio investasi dan cadangan teknis)",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Emas", Ticker: "XAU/USD", Icon: "🏆", Category: "Komoditas", Color: "#facc15", Value: 29.0,
			Basis: "seluruh emas yang pernah ditambang (±216.000 ton), dihitung dengan harga emas terkini",
		},
		LiveKey: "gold", LiveUnits: goldOuncesWorld,
	},
	{
		WorldAsset: WorldAsset{
			Name: "Minyak Mentah", Ticker: "WTI Crude", Icon: "🛢️", Category: "Komoditas", Color: "#f472b6", Value: 24.0,
			Basis: "estimasi nilai cadangan minyak terbukti dunia",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Yuan", Ticker: "CNY", Icon: "🇨🇳", Category: "Mata Uang", Color: "#f87171", Value: 18.0,
			Basis: "nilai uang beredar Tiongkok",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Dolar AS", Ticker: "USD", Icon: "🇺🇸", Category: "Mata Uang", Color: "#34d399", Value: 16.0,
			Basis: "nilai uang beredar Amerika Serikat",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Cadangan Devisa Dunia", Ticker: "FX Reserves", Icon: "🏛️", Category: "Mata Uang", Color: "#fbbf24", Value: 14.0,
			Basis: "total cadangan devisa bank sentral seluruh dunia",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Dana Kekayaan Negara", Ticker: "Sovereign Wealth Funds", Icon: "💼", Category: "Lainnya", Color: "#2dd4bf", Value: 13.0,
			Basis: "total dana kelolaan sovereign wealth fund dunia (Norwegia, Abu Dhabi, Tiongkok, dan lainnya)",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Tembaga", Ticker: "Copper", Icon: "🟤", Category: "Komoditas", Color: "#fb923c", Value: 12.0,
			Basis: "estimasi nilai cadangan tembaga dunia",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Gas Alam", Ticker: "Natural Gas", Icon: "🔥", Category: "Komoditas", Color: "#22d3ee", Value: 9.0,
			Basis: "estimasi nilai cadangan gas alam dunia",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Lahan Pertanian Global", Ticker: "Farmland", Icon: "🌾", Category: "Lainnya", Color: "#a3e635", Value: 8.0,
			Basis: "estimasi nilai lahan pertanian produktif dunia",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Yen", Ticker: "JPY", Icon: "🇯🇵", Category: "Mata Uang", Color: "#818cf8", Value: 4.5,
			Basis: "nilai uang beredar Jepang",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Perak", Ticker: "XAG/USD", Icon: "🥈", Category: "Komoditas", Color: "#cbd5e1", Value: 2.5,
			Basis: "estimasi nilai seluruh perak yang pernah ditambang (±1,74 juta ton), dihitung dengan harga perak terkini",
		},
		LiveKey: "silver", LiveUnits: silverOuncesWorld,
	},
	{
		WorldAsset: WorldAsset{
			Name: "Bitcoin", Ticker: "BTC-USD", Icon: "₿", Category: "Kripto", Color: "#f7931a", Value: 1.6,
			Basis: "seluruh bitcoin yang sudah ditambang (±19,9 juta BTC), dihitung dengan harga Bitcoin terkini",
		},
		LiveKey: "bitcoin", LiveUnits: bitcoinSupplyWorld,
	},
	{
		WorldAsset: WorldAsset{
			Name: "Armada Pelayaran Dunia", Ticker: "Global Shipping Fleet", Icon: "🚢", Category: "Lainnya", Color: "#67e8f9", Value: 1.6,
			Basis: "nilai armada kapal niaga dunia (kapal curah, tanker, dan kapal peti kemas)",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Pesawat Komersial Dunia", Ticker: "Commercial Aircraft", Icon: "✈️", Category: "Lainnya", Color: "#fda4af", Value: 1.4,
			Basis: "nilai seluruh pesawat komersial yang beroperasi (sekitar 30.000 unit)",
		},
	},
	{
		WorldAsset: WorldAsset{
			Name: "Ethereum", Ticker: "ETH-USD", Icon: "💎", Category: "Kripto", Color: "#e879f9", Value: 0.45,
			Basis: "seluruh ether yang beredar (±120 juta ETH), dihitung dengan harga Ethereum terkini",
		},
		LiveKey: "ethereum", LiveUnits: etherSupplyWorld,
	},
}

type WorldAssetService struct{}

func NewWorldAssetService() *WorldAssetService {
	return &WorldAssetService{}
}

// List mengembalikan dua puluh baris halaman Aset Terbesar Dunia, sudah urut
// menurun dari yang nilainya paling besar.
//
// Method ini sengaja tidak mengembalikan error: estimatornya selalu ada, jadi
// kegagalan harga pasar hanya membuat baris live memakai angka estimasinya.
// Halaman edukasi ini tidak boleh kosong hanya karena Yahoo Finance membalas 429,
// dan baris yang benar-benar live sudah ditandai di datanya.
func (s *WorldAssetService) List() []WorldAsset {
	if cached := facades.Cache().Get(worldAssetCacheKey); cached != nil {
		if items, ok := cached.([]WorldAsset); ok {
			return items
		}
	}

	items := worldAssetRows()

	if err := facades.Cache().Put(worldAssetCacheKey, items, marketCacheTTL); err != nil {
		facades.Log().Warningf("gagal menyimpan cache aset dunia: %v", err)
	}

	return items
}

// worldAssetRows menghitung ulang baris yang punya harga pasar lalu mengurutkan
// seluruhnya menurun. Urutannya ditentukan di sini karena harga yang bergerak
// bisa memindahkan baris, berbeda dari tabel harga halaman lain yang urutannya
// tetap.
func worldAssetRows() []WorldAsset {
	prices := worldLiveAssetPrices()

	rows := make([]WorldAsset, 0, len(worldAssetSeeds))
	for _, seed := range worldAssetSeeds {
		row := seed.WorldAsset

		if price := prices[seed.LiveKey]; seed.LiveUnits > 0 && price > 0 {
			row.Value = roundTrillions(seed.LiveUnits * price / 1e12)
			row.Live = true
		}

		rows = append(rows, row)
	}

	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Value > rows[j].Value })

	return rows
}

// roundTrillions membulatkan nilai ke dua angka di belakang koma: cukup untuk
// memisahkan baris yang berdekatan tanpa mengirim angka panjang ke frontend,
// yang toh hanya menampilkan satu angka di belakang koma.
func roundTrillions(value float64) float64 {
	return math.Round(value*100) / 100
}

// worldLiveAssetPrices mengumpulkan harga yang dibutuhkan baris live, memakai
// jalur yang sudah ada supaya angkanya sama dengan yang tampil di halaman lain:
//
//   - emas (xaus.com) dan Bitcoin (CoinGecko) dari payload pasar Dasbor,
//   - perak dan Ethereum dari Yahoo Finance, dengan simbol yang diambil dari
//     daftar komoditas dan daftar koin halaman Crypto — bukan ticker yang ditulis
//     ulang di sini.
//
// Harga yang gagal diambil tidak menghalangi yang lain: baris itu tinggal memakai
// angka estimasinya, dan penyebabnya dicatat di log.
func worldLiveAssetPrices() map[string]float64 {
	prices := map[string]float64{}

	if payload, err := NewMarketService().GetCurrentPrices(); err != nil {
		facades.Log().Warningf("harga pasar gagal dipakai untuk aset dunia: %v", err)
	} else {
		if gold, ok := toMap(payload["gold"]); ok {
			prices["gold"] = toFloat64(gold["price_usd_oz"])
		}
		if bitcoin, ok := toMap(payload["bitcoin"]); ok {
			prices["bitcoin"] = toFloat64(bitcoin["price_usd"])
		}
	}

	instruments := make([]yahooInstrument, 0, 2)
	if symbol, ok := historyYahooSymbol("silver"); ok {
		instruments = append(instruments, yahooInstrument{Asset: "silver", Symbol: symbol, Name: "Perak"})
	}
	if coin, ok := cryptoCoinByID("ethereum"); ok {
		instruments = append(instruments, yahooInstrument{Asset: "ethereum", Symbol: coin.Yahoo, Name: "Ethereum"})
	}

	if quotes, err := fetchYahooQuotes(instruments); err != nil {
		facades.Log().Warningf("kuotasi pendukung aset dunia gagal: %v", err)
	} else {
		for _, quote := range quotes {
			prices[toStr(quote["asset"])] = toFloat64(quote["price"])
		}
	}

	return prices
}
