package types

import "time"

// EventType represents the type of real-time event
type EventType string

const (
	EventStoryViewed  EventType = "story.viewed"
	EventStoryReacted EventType = "story.reacted"
)

// Event represents a real-time event that can be sent over WebSocket
type Event struct {
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// StoryViewedEvent represents when a user views a story
type StoryViewedEvent struct {
	StoryID  string `json:"story_id"`
	ViewerID string `json:"viewer_id"`
	ViewedAt string `json:"viewed_at"`
}

// StoryReactedEvent represents when a user reacts to a story
type StoryReactedEvent struct {
	StoryID   string       `json:"story_id"`
	UserID    string       `json:"user_id"`
	Emoji     ReactionType `json:"emoji"`
	ReactedAt string       `json:"reacted_at"`
}

// NewEvent creates a new event with the current timestamp
func NewEvent(eventType EventType, data interface{}) *Event {
	return &Event{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
