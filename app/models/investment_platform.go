package models

import (
	"encoding/json"
	"strings"

	"github.com/goravel/framework/database/orm"
)

// Kategori dan asal platform yang dikenal halaman Rekomendasi Platform.
const (
	PlatformCategoryCrypto    = "crypto"
	PlatformCategorySaham     = "saham"
	PlatformCategoryKomoditas = "komoditas"

	PlatformTypeLocal   = "Lokal"
	PlatformTypeForeign = "Luar Negeri"
)

// InvestmentPlatform adalah satu baris daftar platform investasi. Pros dan Cons
// disimpan sebagai JSON array di kolom teks supaya isinya tidak bergantung pada
// tabel anak, tetapi API selalu mengirimnya sebagai array.
type InvestmentPlatform struct {
	orm.Model
	Name        string `gorm:"type:string;not null" json:"name"`
	Category    string `gorm:"type:string;not null" json:"category"`
	Type        string `gorm:"type:string;not null" json:"type"`
	LogoDomain  string `gorm:"type:string" json:"logo_domain"`
	Description string `gorm:"type:text;not null" json:"description"`
	Pros        string `gorm:"type:text;not null" json:"-"`
	Cons        string `gorm:"type:text;not null" json:"-"`
}

// ProsList mengubah kolom JSON menjadi daftar teks. Nilai yang rusak dianggap
// daftar kosong supaya satu baris tidak menggagalkan seluruh endpoint.
func (p InvestmentPlatform) ProsList() []string {
	return platformStringsFromJSON(p.Pros)
}

// ConsList sama seperti ProsList untuk kolom kekurangan.
func (p InvestmentPlatform) ConsList() []string {
	return platformStringsFromJSON(p.Cons)
}

// PlatformStringsToJSON menyusun kolom pros/cons dari daftar teks.
func PlatformStringsToJSON(items []string) string {
	clean := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			clean = append(clean, trimmed)
		}
	}

	encoded, err := json.Marshal(clean)
	if err != nil {
		return "[]"
	}

	return string(encoded)
}

func platformStringsFromJSON(raw string) []string {
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}

	return items
}
