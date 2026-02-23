package models

import (
	"encoding/json"
)

// Message represents a chat message
type Message struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Content  string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// BlockedUser represents a blocked user entry
type BlockedUser struct {
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
}

// BlockedUsersList represents the list of blocked users for a specific user
type BlockedUsersList struct {
	Users map[string]BlockedUser `json:"users"`
}

// CensoredWord represents a censored word entry
type CensoredWord struct {
	Word      string `json:"word"`
	Timestamp int64  `json:"timestamp"`
}

// CensoredWordsList represents the global list of censored words
type CensoredWordsList struct {
	Words map[string]CensoredWord `json:"words"`
}

// BlockAction represents an action to block a user
type BlockAction struct {
	BlockerID string `json:"blocker_id"` // User who is blocking
	BlockedID string `json:"blocked_id"` // User being blocked
	Action    string `json:"action"`     // "block" or "unblock"
	Timestamp int64  `json:"timestamp"`
}

// CensorAction represents an action to add/remove censored word
type CensorAction struct {
	Word      string `json:"word"`
	Action    string `json:"action"` // "add" or "remove"
	Timestamp int64  `json:"timestamp"`
}

// MessageCodec is the codec for Message
type MessageCodec struct{}

func (c *MessageCodec) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (c *MessageCodec) Decode(data []byte) (interface{}, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// BlockedUsersListCodec is the codec for BlockedUsersList
type BlockedUsersListCodec struct{}

func (c *BlockedUsersListCodec) Encode(value interface{}) ([]byte, error) {
	if value == nil {
		return json.Marshal(&BlockedUsersList{Users: make(map[string]BlockedUser)})
	}
	return json.Marshal(value)
}

func (c *BlockedUsersListCodec) Decode(data []byte) (interface{}, error) {
	var list BlockedUsersList
	err := json.Unmarshal(data, &list)
	if err != nil {
		return &BlockedUsersList{Users: make(map[string]BlockedUser)}, err
	}
	if list.Users == nil {
		list.Users = make(map[string]BlockedUser)
	}
	return &list, nil
}

// BlockActionCodec is the codec for BlockAction
type BlockActionCodec struct{}

func (c *BlockActionCodec) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (c *BlockActionCodec) Decode(data []byte) (interface{}, error) {
	var action BlockAction
	err := json.Unmarshal(data, &action)
	return &action, err
}

// CensoredWordsListCodec is the codec for CensoredWordsList
type CensoredWordsListCodec struct{}

func (c *CensoredWordsListCodec) Encode(value interface{}) ([]byte, error) {
	if value == nil {
		return json.Marshal(&CensoredWordsList{Words: make(map[string]CensoredWord)})
	}
	return json.Marshal(value)
}

func (c *CensoredWordsListCodec) Decode(data []byte) (interface{}, error) {
	var list CensoredWordsList
	err := json.Unmarshal(data, &list)
	if err != nil {
		return &CensoredWordsList{Words: make(map[string]CensoredWord)}, err
	}
	if list.Words == nil {
		list.Words = make(map[string]CensoredWord)
	}
	return &list, nil
}

// CensorActionCodec is the codec for CensorAction
type CensorActionCodec struct{}

func (c *CensorActionCodec) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (c *CensorActionCodec) Decode(data []byte) (interface{}, error) {
	var action CensorAction
	err := json.Unmarshal(data, &action)
	return &action, err
}
