package models

import (
	"github.com/goravel/framework/database/orm"
)

const (
	ChatRoleUser      = "user"
	ChatRoleAssistant = "assistant"
)

type ChatMessage struct {
	orm.Model
	UserID  uint   `gorm:"not null" json:"user_id"`
	Role    string `gorm:"type:string;not null" json:"role"`
	Message string `gorm:"type:text;not null" json:"message"`
}
