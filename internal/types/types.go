package types

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityFriends Visibility = "friends"
	VisibilityPrivate Visibility = "private"
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
