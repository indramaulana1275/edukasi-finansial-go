package models

import (
	"github.com/goravel/framework/database/orm"
)

// NlgTestResult menyimpan satu penilaian narasi dari panel pengujian admin.
// Rata-rata dihitung di server agar klien tidak bisa mengirim nilai yang
// berbeda dari empat skor aspek.
type NlgTestResult struct {
	orm.Model
	Profile        string  `gorm:"type:string;not null" json:"profile"`
	Topic          string  `gorm:"type:string;not null" json:"topic"`
	Narrative      string  `gorm:"type:text;not null" json:"narrative"`
	ScoreData      int     `gorm:"not null" json:"score_data"`
	ScoreContent   int     `gorm:"not null" json:"score_content"`
	ScoreRelevance int     `gorm:"not null" json:"score_relevance"`
	ScoreClarity   int     `gorm:"not null" json:"score_clarity"`
	AverageScore   float64 `gorm:"type:float;not null" json:"average_score"`
}

func (r *NlgTestResult) TableName() string {
	return "nlg_test_results"
}
