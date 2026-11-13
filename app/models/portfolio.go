package models

import (
	"github.com/goravel/framework/database/orm"
)

// Jenis aset yang dipakai di portfolio. Simbolnya sengaja datar dan stabil
// supaya holding lama tetap terbaca; frontend memakai kategori + sub-kategori
// hanya untuk menyusun pilihan, bukan untuk menyimpan jenis aset.
const (
	AssetTypeCrypto    = "crypto"
	AssetTypeStock     = "stock"
	AssetTypeCommodity = "commodity"
	AssetTypeGold      = "gold"
	AssetTypeSukuk     = "sukuk"
	AssetTypeFiat      = "fiat"
)

// Simbol portfolio yang dikenal beserta jenis asetnya. Daftar ini mencakup
// aset yang harganya sudah tersedia lewat /api/saham dan /api/komoditas.
// XAU/ANTAM memakai harga emas per gram dari /api/market, SBR014 memakai nilai
// nominal, sedangkan USD dipertahankan untuk holding lama.
var portfolioAssetTypes = map[string]string{
	"BTC": AssetTypeCrypto, "ETH": AssetTypeCrypto, "BNB": AssetTypeCrypto,
	"SOL": AssetTypeCrypto, "XRP": AssetTypeCrypto, "ADA": AssetTypeCrypto,
	"SUI": AssetTypeCrypto, "TRX": AssetTypeCrypto, "DOT": AssetTypeCrypto,
	"LTC": AssetTypeCrypto,

	"IHSG": AssetTypeStock, "BBCA": AssetTypeStock, "BBRI": AssetTypeStock,
	"TLKM": AssetTypeStock, "ASII": AssetTypeStock, "BMRI": AssetTypeStock,
	"UNVR": AssetTypeStock, "ICBP": AssetTypeStock, "INDF": AssetTypeStock,
	"KLBF": AssetTypeStock, "AAPL": AssetTypeStock, "MSFT": AssetTypeStock,
	"GOOGL": AssetTypeStock, "AMZN": AssetTypeStock, "TSLA": AssetTypeStock,
	"NVDA": AssetTypeStock, "META": AssetTypeStock,

	"SBR014": AssetTypeSukuk,

	"ANTAM": AssetTypeGold, "XAU": AssetTypeGold, "XAG": AssetTypeCommodity, "HG": AssetTypeCommodity,
	"ALI": AssetTypeCommodity, "TIO": AssetTypeCommodity, "MTF": AssetTypeCommodity,
	"CHILI": AssetTypeCommodity, "SHALLOT": AssetTypeCommodity,
	"GARLIC": AssetTypeCommodity, "RAWIT": AssetTypeCommodity,
	"COFFEE": AssetTypeCommodity, "WHEAT": AssetTypeCommodity,
	"CORN": AssetTypeCommodity, "SOYBEAN": AssetTypeCommodity,
	"RICE": AssetTypeCommodity, "SUGAR": AssetTypeCommodity,
	"COCOA": AssetTypeCommodity, "CPO": AssetTypeCommodity,
	"BRENT": AssetTypeCommodity, "GASOLINE": AssetTypeCommodity,
	"URANIUM": AssetTypeCommodity, "LITHIUM": AssetTypeCommodity,
	"STEEL": AssetTypeCommodity, "ZINC": AssetTypeCommodity,
	"LEAD": AssetTypeCommodity, "COTTON": AssetTypeCommodity,
	"LUMBER": AssetTypeCommodity, "COKINGCOAL": AssetTypeCommodity,

	"USD": AssetTypeFiat,
}

// AssetTypeForSymbol mengembalikan jenis aset untuk sebuah simbol portfolio.
// Boolean kedua bernilai false kalau simbolnya tidak dikenal.
func AssetTypeForSymbol(symbol string) (string, bool) {
	assetType, known := portfolioAssetTypes[symbol]

	return assetType, known
}

type Portfolio struct {
	orm.Model
	UserID      uint    `gorm:"not null" json:"user_id"`
	AssetSymbol string  `gorm:"type:string;not null" json:"asset_symbol"`
	AssetType   string  `gorm:"type:string;not null" json:"asset_type"`
	Quantity    float64 `gorm:"not null" json:"quantity"`
	BuyPrice    float64 `gorm:"not null" json:"buy_price"`
}
