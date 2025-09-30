package storage

type Storage interface {
	CreateStory(authorID, text, mediaKey, visibility string, audienceUserIDs []string) (string, error)
	CreateUser(email, password string) (string, error)
	GetUserByEmail(email string) (string, string, error)
}
