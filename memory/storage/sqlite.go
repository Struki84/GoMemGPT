package storage

import (
	"log"

	"github.com/tmc/langchaingo/llms"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SqliteStorage struct {
	DB   *gorm.DB
	Data Memory
}

func NewSqliteStorage() SqliteStorage {
	storage := SqliteStorage{}
	dbPath := "memory.db"

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

	err = db.AutoMigrate(Memory{})
	if err != nil {
		log.Printf("Error migrating DB: %v", err)
		return storage
	}

	storage.DB = db
	return storage
}

func (db SqliteStorage) LoadMessages() ([]llms.MessageContent, error) {
	err := db.DB.Find(&db.Data).Error
	if err != nil {
		return []llms.MessageContent{}, err
	}

	messages := []llms.MessageContent{}
	for _, message := range *db.Data.Messages {
		var msgType llms.ChatMessageType = llms.ChatMessageType(message.Role)
		messages = append(messages, llms.TextParts(msgType, message.Content))
	}

	return messages, nil
}

func (db SqliteStorage) SaveMessages(messages []llms.MessageContent) error {
	msgs := Messages{}
	for _, message := range messages {
		msgs = append(msgs, Message{
			Role:    string(message.Role),
			Content: message.Parts[0].(llms.TextContent).String(),
		})
	}

	db.Data.Messages = &msgs

	err := db.DB.Save(&db.Data).Error
	if err != nil {
		return err
	}

	return nil
}

func (db SqliteStorage) LoadWorkingContext() (string, error) {
	err := db.DB.Find(&db.Data).Error
	if err != nil {
		return "", err
	}

	return db.Data.WorkingContext, nil
}

func (db SqliteStorage) SaveWorkingContext(workingContext string) error {
	db.Data.WorkingContext = workingContext

	err := db.DB.Save(&db.Data).Error
	if err != nil {
		return err
	}

	return nil
}

func (db SqliteStorage) RecallMessages() ([]llms.MessageContent, error) {
	return []llms.MessageContent{}, nil
}

func (db SqliteStorage) ArchiveMessages(messages []llms.MessageContent) error {
	return nil
}

func (db SqliteStorage) RecallWorkingContext() (string, error) {
	return "", nil
}

func (db SqliteStorage) ArchiveWorkingContext(workingContext string) error {
	return nil
}
