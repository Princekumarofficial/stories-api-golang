package events

import (
	"time"

	"github.com/princekumarofficial/stories-service/internal/types"
)

// Publisher interface for publishing events
type Publisher interface {
	PublishStoryViewed(storyID, viewerID, authorID string) error
	PublishStoryReacted(storyID, userID, authorID string, emoji types.ReactionType) error
}

// EventPublisher implements the Publisher interface
type EventPublisher struct {
	hub WebSocketHub
}

// WebSocketHub interface for the WebSocket hub
type WebSocketHub interface {
	BroadcastToUser(userID string, event *types.Event)
	BroadcastToUsers(userIDs []string, event *types.Event)
	IsUserConnected(userID string) bool
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(hub WebSocketHub) *EventPublisher {
	return &EventPublisher{
		hub: hub,
	}
}

// PublishStoryViewed publishes a story viewed event to the story author
func (p *EventPublisher) PublishStoryViewed(storyID, viewerID, authorID string) error {
	// Don't send notification if the author viewed their own story
	if viewerID == authorID {
		return nil
	}

	// Only send if the author is connected
	if !p.hub.IsUserConnected(authorID) {
		return nil
	}

	eventData := &types.StoryViewedEvent{
		StoryID:  storyID,
		ViewerID: viewerID,
		ViewedAt: time.Now().UTC().Format(time.RFC3339),
	}

	event := types.NewEvent(types.EventStoryViewed, eventData)
	p.hub.BroadcastToUser(authorID, event)

	return nil
}

// PublishStoryReacted publishes a story reacted event to the story author
func (p *EventPublisher) PublishStoryReacted(storyID, userID, authorID string, emoji types.ReactionType) error {
	// Don't send notification if the author reacted to their own story
	if userID == authorID {
		return nil
	}

	// Only send if the author is connected
	if !p.hub.IsUserConnected(authorID) {
		return nil
	}

	eventData := &types.StoryReactedEvent{
		StoryID:   storyID,
		UserID:    userID,
		Emoji:     emoji,
		ReactedAt: time.Now().UTC().Format(time.RFC3339),
	}

	event := types.NewEvent(types.EventStoryReacted, eventData)
	p.hub.BroadcastToUser(authorID, event)

	return nil
}
