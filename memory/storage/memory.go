package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

type Memory struct {
	gorm.Model
	WorkingContext  string    `json:"workingContext"`
	WorkingMessages *Messages `json:"workingMessage"`
	Messages        *Messages `json:"messages"`
}

type Messages []Message

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
