package types

const DefaultFeedLimit = 50

type FeedCursor struct {
	CreatedAt string
	ID        int
}

type Visibility string

const (
	VisibilityPublic  Visibility = "PUBLIC"
	VisibilityFriends Visibility = "FRIENDS"
	VisibilityPrivate Visibility = "PRIVATE"
)

type Story struct {
	ID         string     `json:"id"`
	AuthorID   string     `json:"author_id"`
	Text       string     `json:"text"`
	MediaKey   string     `json:"media_key"`
	Visibility Visibility `json:"visibility"`
	CreatedAt  string     `json:"created_at"`
	ExpiresAt  string     `json:"expires_at"`
	DeletedAt  string     `json:"deleted_at"`
}

// StoryWithMeta extends Story with preloaded metadata to avoid N+1 queries
type StoryWithMeta struct {
	Story
	// Author information
	AuthorEmail string `json:"author_email"`

	// Story statistics
	ViewCount     int `json:"view_count"`
	ReactionCount int `json:"reaction_count"`

	// User-specific flags
	UserHasViewed bool   `json:"user_has_viewed"`
	UserReaction  string `json:"user_reaction"`
}

type StoryPostRequest struct {
	Text            string     `json:"text"`
	MediaKey        string     `json:"media_key"`
	Visibility      Visibility `validate:"required" json:"visibility"`
	AudienceUserIDs []string   `validate:"required" json:"audience_user_ids"`
}

type ReactionType string

const (
	ReactionThumbsUp  ReactionType = "👍"
	ReactionHeart     ReactionType = "❤️"
	ReactionLaugh     ReactionType = "😂"
	ReactionSurprised ReactionType = "😮"
	ReactionSad       ReactionType = "😢"
	ReactionFire      ReactionType = "🔥"
)

type Reaction struct {
	ID        string       `json:"id"`
	StoryID   string       `json:"story_id"`
	UserID    string       `json:"user_id"`
	Emoji     ReactionType `json:"emoji"`
	ReactedAt string       `json:"reacted_at"`
}

type ReactionRequest struct {
	Emoji ReactionType `json:"emoji" validate:"required"`
}

type Follow struct {
	FollowerID string `json:"follower_id"`
	FollowedID string `json:"followed_id"`
	CreatedAt  string `json:"created_at"`
}
