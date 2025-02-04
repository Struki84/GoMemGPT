package storage

import "gorm.io/gorm"

type MsgStatus int

const (
	current MsgStatus = iota
	archived
)

type Memory struct {
	gorm.Model
	SessionID string    `json:"sessionId"`
	Context   string    `json:"workingContext"`
	Messages  []Message `json:"messages" gorm:"foreignKey:MemoryID"`
}

type Message struct {
	gorm.Model
	MemoryID uint
	Role     string    `json:"type"`
	Content  string    `json:"text"`
	Status   MsgStatus `json:"status"`
}
