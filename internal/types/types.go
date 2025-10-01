package types

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
