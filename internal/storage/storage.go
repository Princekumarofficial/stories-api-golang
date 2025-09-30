package storage

type Storage interface {
	CreateStory(authorID, text, mediaKey, visibility string, audienceUserIDs []string) (string, error)
}
