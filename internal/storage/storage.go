package storage

import "github.com/princekumarofficial/stories-service/internal/types"

type Storage interface {
	CreateStory(authorID, text, mediaKey string, visibility types.Visibility, audienceUserIDs []string) (string, error)
	CreateUser(email, password string) (string, error)
	GetUserByEmail(email string) (string, string, error)
	GetAllPublicStories() ([]types.Story, error)
	GetStoriesForUser(userID string) ([]types.Story, error)
	GetStoryByID(storyID string) (types.Story, error)
}
