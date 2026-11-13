package models

import (
	"github.com/goravel/framework/database/orm"
)

// Role yang dikenal aplikasi. Nilai ini disimpan di kolom users.role dan juga
// dipakai sebagai subjek Casbin (lihat config/casbin_policy.csv).
const (
	UserRoleUser  = "user"
	UserRoleAdmin = "admin"
)

type User struct {
	orm.Model
	Name     string `gorm:"type:string;not null" json:"name"`
	Email    string `gorm:"type:string;unique;not null" json:"email"`
	Password string `gorm:"type:string;not null" json:"-"`
	Role     string `gorm:"type:string;not null" json:"role"`
	// Profil opsional yang diisi dari halaman Profil Saya. Nilai kosong
	// berarti user belum mengisinya.
	Gender    string `gorm:"type:string" json:"gender"`
	AvatarURL string `gorm:"type:string" json:"avatar_url"`
	Bio       string `gorm:"type:string" json:"bio"`
}
