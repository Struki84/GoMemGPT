package storage

import "gorm.io/gorm"

type Memory struct {
	gorm.Model
	WorkingContext    string
	HistoricalContext string
	Messages          *Messages
	ChatHistory       *Messages
}

type Messages []Message

type Message struct {
	Role    string
	Content string
}
