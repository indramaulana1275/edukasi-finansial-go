package models

import (
	"github.com/goravel/framework/database/orm"
)

// QuizResult adalah satu percobaan kuis modul. Berbeda dari
// user_module_progress yang hanya menyimpan status selesai, tabel ini menyimpan
// riwayat skor setiap kali user menekan "Kirim Jawaban".
type QuizResult struct {
	orm.Model
	UserID   uint `gorm:"not null" json:"user_id"`
	ModuleID uint `gorm:"not null" json:"module_id"`
	Score    int  `gorm:"not null" json:"score"`
	Total    int  `gorm:"not null" json:"total"`
	Passed   bool `gorm:"not null" json:"passed"`
}

func (r *QuizResult) TableName() string {
	return "quiz_results"
}
