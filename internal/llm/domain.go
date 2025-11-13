package llm

// This passes data to and from the Gemini API.
type ChatMessage struct {
	// Role is who sent the message, e.g., "user" or "model".
	Role string `json:"role"`
	// Content is the text of the message.
	Content string `json:"content"`
}
