package storage

import (
	"log"

	"github.com/tmc/langchaingo/llms"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SqliteStorage struct {
	DB        *gorm.DB
	Data      Memory
	sessionID string
}

func NewSqliteStorage() SqliteStorage {
	storage := SqliteStorage{}
	dbPath := "./memory.db"

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		return storage
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting DB: %v", err)
		return storage
	}

	sqlDB.Exec("PRAGMA foreign_keys = ON;")
	sqlDB.Exec("PRAGMA journal_mode = WAL;")

	err = db.AutoMigrate(&Memory{}, &Message{})
	if err != nil {
		log.Printf("Error migrating DB: %v", err)
		return storage
	}

	storage.DB = db
	storage.sessionID = "session-1" // tmp hardcoded session id
	return storage
}

func (db SqliteStorage) LoadMessages() ([]llms.MessageContent, error) {
	var memory Memory
	err := db.DB.Where("session_id = ?", db.sessionID).First(&memory).Error
	if err != nil {
		return []llms.MessageContent{}, err
	}

	var msgs []Message
	query := db.DB.Where("memory_id = ? AND status = ?", memory.ID, current)
	query.Order("created_at DESC")

	err = query.Find(&msgs).Error
	if err != nil {
		return []llms.MessageContent{}, err
	}

	result := []llms.MessageContent{}
	for _, message := range memory.Messages {
		var msgType llms.ChatMessageType = llms.ChatMessageType(message.Role)
		result = append(result, llms.TextParts(msgType, message.Content))
	}

	return result, nil
}

func (db SqliteStorage) SaveMessages(messages []llms.MessageContent) error {
	var memory Memory

	err := db.DB.Where("session_id = ?", db.sessionID).First(&memory).Error
	if err != nil {
		return err
	}

	msgs := []Message{}
	for _, message := range messages {
		msgs = append(msgs, Message{
			Role:     string(message.Role),
			Content:  message.Parts[0].(llms.TextContent).String(),
			MemoryID: memory.ID,
			Status:   current,
		})
	}

	return db.DB.Save(&msgs).Error
}

func (db SqliteStorage) LoadWorkingContext() (string, error) {
	var memory Memory

	err := db.DB.Where("session_id = ?", db.sessionID).First(&memory).Error
	if err != nil {
		return "", err
	}

	return memory.Context, nil
}

func (db SqliteStorage) SaveWorkingContext(workingContext string) error {
	query := db.DB.Model(&Memory{})
	query.Where("session_id = ?", db.sessionID)

	return query.Update("context", workingContext).Error
}

func (db SqliteStorage) RecallMessages(search string, limit, offset int) ([]llms.MessageContent, error) {
	var memory Memory
	err := db.DB.Where("session_id = ?", db.sessionID).First(&memory).Error
	if err != nil {
		return []llms.MessageContent{}, err
	}

	var msgs []Message

	query := db.DB.Where("memory_id = ? AND status = ?", memory.ID, archived)
	query.Where("content LIKE ?", "%"+search+"%")
	query.Order("created_at DESC")
	query.Limit(limit).Offset(offset)

	err = query.Find(&msgs).Error
	if err != nil {
		return []llms.MessageContent{}, err
	}

	var results []llms.MessageContent
	for _, msg := range msgs {
		var msgType llms.ChatMessageType = llms.ChatMessageType(msg.Role)
		results = append(results, llms.TextParts(msgType, msg.Content))
	}

	return results, nil
}

func (db SqliteStorage) ArchiveMessages(messages []llms.MessageContent) error {
	var memory Memory

	err := db.DB.Where("session_id = ?", db.sessionID).First(&memory).Error
	if err != nil {
		return err
	}

	archivedMsgs := []Message{}

	for _, message := range messages {
		archivedMsgs = append(archivedMsgs, Message{
			Role:     string(message.Role),
			Content:  message.Parts[0].(llms.TextContent).String(),
			MemoryID: memory.ID,
			Status:   archived,
		})
	}

	return db.DB.Where("memory_id = ?", memory.ID).Delete(&archivedMsgs).Error
}
