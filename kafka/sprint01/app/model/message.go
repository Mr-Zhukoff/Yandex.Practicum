package model

import (
	"encoding/json"
	"time"
)

// Message represents a structured message to be sent/received via Kafka.
type Message struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}

// NewMessage creates a new message with the given id, content, and source.
func NewMessage(id int, content, source string) *Message {
	return &Message{
		ID:        id,
		Content:   content,
		Timestamp: time.Now(),
		Source:    source,
	}
}

// ToJSON serializes the message to JSON bytes.
// Returns the JSON bytes and an error if serialization fails.
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes JSON bytes into a Message.
// Returns a pointer to the Message and an error if deserialization fails.
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
