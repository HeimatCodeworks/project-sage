package chat

import "time"

// Message represents a single message from a Twilio conversation.
type Message struct {
	// SID is the unique ID from Twilio
	SID string `json:"sid"`
	// Author is the identity of the sender (eg. UserID or ExpertID)
	Author string `json:"author"`
	// Content is the text of the message
	Content string `json:"content"`
	// Timestamp is when the message was sent
	Timestamp time.Time `json:"timestamp"`
}
