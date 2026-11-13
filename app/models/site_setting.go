package models

import "github.com/goravel/framework/database/orm"

// SiteSetting adalah satu baris pengaturan situs yang bisa diubah admin.
// Nilai kosong berarti bagian itu tidak ditampilkan di halaman publik.
type SiteSetting struct {
	orm.Model
	WelcomeText    string `gorm:"type:text;not null" json:"welcome_text"`
	ContactEmail   string `gorm:"type:string" json:"contact_email"`
	ContactPhone   string `gorm:"type:string" json:"contact_phone"`
	ContactAddress string `gorm:"type:text" json:"contact_address"`
}
