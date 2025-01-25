package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pkoukk/tiktoken-go"
	"gorm.io/gorm"
)

type Memory struct {
	gorm.Model
	WorkingContext string    `json:"workingContext"`
	Messages       *Messages `json:"messages"`
}

type Messages []Message

func (messages Messages) TokenSize() (int, error) {
	encoder, err := tiktoken.GetEncoding("cl100k_base")

	if err != nil {
		return 0, err
	}

	totalTokens := 0

	for _, msg := range messages {
		contentToEncode := fmt.Sprintf("%s: %s", msg.Role, msg.Content)
		tokenIDs := encoder.Encode(contentToEncode, nil, nil)
		totalTokens += len(tokenIDs)
	}

	// The chat format typically has an extra 2 tokens at the end
	totalTokens += 2

	return totalTokens, nil
}

type Message struct {
	Role    string `json:"type"`
	Content string `json:"text"`
}

// Value implements the driver.Valuer interface, this method allows us to
// customize how we store the Message type in the database.
func (m Messages) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface, this method allows us to
// define how we convert the Message data from the database into our Message type.
func (m *Messages) Scan(src interface{}) error {
	if bytes, ok := src.([]byte); ok {
		return json.Unmarshal(bytes, m)
	}
	return errors.New("could not scan type into Message")
}
