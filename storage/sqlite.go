package storage

import (
	"errors"
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
	dbPath := "./storage/memory.db"

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
		log.Printf("Error loading messages: %v", err)
		return []llms.MessageContent{}, err
	}

	var msgs []Message
	query := db.DB.Where("memory_id = ? AND status = ?", memory.ID, current)
	query.Order("created_at ASC")

	err = query.Find(&msgs).Error
	if err != nil {
		log.Printf("Error loading messages: %v", err)
		return []llms.MessageContent{}, err
	}

	result := []llms.MessageContent{}
	for _, message := range msgs {
		var msgType llms.ChatMessageType = llms.ChatMessageType(message.Role)
		result = append(result, llms.TextParts(msgType, message.Content))
	}

	return result, nil
}

func (db SqliteStorage) SaveMessages(messages []llms.MessageContent) error {
	if len(messages) > 0 {
		messages = messages[1:]
	}

	memory := Memory{
		SessionID: db.sessionID,
	}

	err := db.DB.Where("session_id = ?", db.sessionID).FirstOrCreate(&memory).Error
	if err != nil {
		return err
	}

	var existingMsgs []Message

	err = db.DB.Where("memory_id = ?", memory.ID).Find(&existingMsgs).Error
	if err != nil {
		return err
	}

	existingMsgsSet := make(map[string]struct{})

	for _, msg := range existingMsgs {
		existingMsgsSet[msg.Role+msg.Content] = struct{}{}
	}

	newMsgs := []Message{}
	for _, msg := range messages {
		var msgContent string

		if msg.Role == llms.ChatMessageTypeTool {
			msgContent = msg.Parts[0].(llms.ToolCallResponse).Content
		} else {
			msgContent = msg.Parts[0].(llms.TextContent).String()
		}

		key := string(msg.Role) + msgContent

		if _, ok := existingMsgsSet[key]; !ok {
			newMsgs = append(newMsgs, Message{
				Role:     string(msg.Role),
				Content:  msgContent,
				MemoryID: memory.ID,
				Status:   current,
			})
		}
	}

	if len(newMsgs) > 0 {
		return db.DB.Save(&newMsgs).Error
	}

	return errors.New("no new messages to save")
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

func (db SqliteStorage) RecallMessages(search string, limit, offset int) (string, error) {
	var memory Memory
	err := db.DB.Where("session_id = ?", db.sessionID).First(&memory).Error
	if err != nil {
		return "", err
	}

	var msgs []Message

	query := db.DB.Where("memory_id = ? AND status = ? AND (role = ? OR role = ?)", memory.ID, archived, "human", "ai")
	query.Where("content LIKE ?", "%"+search+"%")
	query.Order("created_at DESC")
	query.Limit(limit).Offset(offset)

	err = query.Find(&msgs).Error
	if err != nil {
		return "", err
	}

	if len(msgs) == 0 {
		return "", errors.New("no messages found")
	}

	var results string
	for _, msg := range msgs {
		timestamp := msg.CreatedAt.Format("2006-01-02 15:04:05")
		role := msg.Role
		results += timestamp + ": " + role + " - " + msg.Content + "\n"
	}

	return results, nil
}

func (db SqliteStorage) ArchiveMessages(messages []llms.MessageContent) error {
	var memory Memory

	err := db.DB.Where("session_id = ?", db.sessionID).First(&memory).Error
	if err != nil {
		return err
	}

	var existingMsgs []Message

	query := db.DB.Where("memory_id = ? AND status = ?", memory.ID, current)
	query.Order("created_at DESC")

	err = query.Find(&existingMsgs).Error
	if err != nil {
		return err
	}

	archivedMsgs := []Message{}
	for _, msg := range existingMsgs[:len(existingMsgs)-3] {
		msg.Status = archived
		archivedMsgs = append(archivedMsgs, msg)
	}

	if len(archivedMsgs) > 0 {
		return db.DB.Save(&archivedMsgs).Error
	}

	return errors.New("no new messages to archive")
}
