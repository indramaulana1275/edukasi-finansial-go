package models

import "github.com/goravel/framework/database/orm"

const (
	FeedbackStatusNew  = "baru"
	FeedbackStatusRead = "dibaca"
)

// FeedbackMessage adalah saran atau masukan yang dikirim user dari halaman
// Profil Saya. Status hanya dipakai panel admin untuk menandai sudah dibaca.
type FeedbackMessage struct {
	orm.Model
	UserID  uint   `gorm:"not null" json:"user_id"`
	Name    string `gorm:"type:string;not null" json:"name"`
	Email   string `gorm:"type:string;not null" json:"email"`
	Subject string `gorm:"type:string;not null" json:"subject"`
	Message string `gorm:"type:text;not null" json:"message"`
	Status  string `gorm:"type:string;not null" json:"status"`
}
