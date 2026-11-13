package models

import (
	"github.com/goravel/framework/database/orm"
)

const (
	RisikoKonservatif = "Konservatif"
	RisikoModerat     = "Moderat"
	RisikoAgresif     = "Agresif"
)

type RiskProfile struct {
	orm.Model
	UserID     uint   `gorm:"not null" json:"user_id"`
	Skor       int    `gorm:"not null" json:"skor"`
	TipeRisiko string `gorm:"type:string;not null" json:"tipe_risiko"`
}
