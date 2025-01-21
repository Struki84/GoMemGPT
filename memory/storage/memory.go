package storage

import "gorm.io/gorm"

type Memory struct {
	gorm.Model
	WorkingContext    string
	HistoricalContext string
	Messages          []Message
	ChatHistory       []Message
}

type Message struct {
	Role    string
	Content string
}
