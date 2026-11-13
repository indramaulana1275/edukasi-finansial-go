package models

import "github.com/goravel/framework/database/orm"

// ActivityLog mencatat aktivitas masuk yang penting untuk Panel Developer.
// Role dan email disimpan sebagai salinan supaya riwayat tetap terbaca setelah
// akun diganti namanya atau dihapus.
type ActivityLog struct {
	orm.Model
	UserID    uint   `gorm:"not null;default:0" json:"user_id"`
	Name      string `gorm:"type:string;not null" json:"name"`
	Email     string `gorm:"type:string;not null" json:"email"`
	Role      string `gorm:"type:string;not null" json:"role"`
	Action    string `gorm:"type:string;not null" json:"action"`
	IP        string `gorm:"type:string" json:"ip"`
	UserAgent string `gorm:"type:text" json:"user_agent"`
}
