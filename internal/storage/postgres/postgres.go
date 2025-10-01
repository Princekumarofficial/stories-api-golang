package postgres

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/princekumarofficial/stories-service/internal/config"
	"github.com/princekumarofficial/stories-service/internal/types"
)

type Postgres struct {
	Db *sql.DB
}

func NewPostgres(cfg *config.Config) (*Postgres, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.PGSQL.Host, cfg.PGSQL.Port, cfg.PGSQL.User, cfg.PGSQL.Password, cfg.PGSQL.DBName, cfg.PGSQL.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	log.Println("Connected to Postgres database")

	// Create tables if they don't exist
	pg := &Postgres{Db: db}
	err = pg.CreateTables()
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	return &Postgres{Db: db}, nil
}

func (p *Postgres) CreateTables() error {
	queries := []string{
		`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS stories (
			id SERIAL PRIMARY KEY,
			author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			text TEXT,
			media_key VARCHAR(255),
			visibility VARCHAR(50) NOT NULL CHECK (visibility IN ('FRIENDS', 'PRIVATE', 'PUBLIC')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP + INTERVAL '24 hours')
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS story_audience (
			story_id INTEGER NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			PRIMARY KEY (story_id, user_id)
		);
		`,
		`CREATE TABLE IF NOT EXISTS story_views (
			id SERIAL PRIMARY KEY,
			story_id INTEGER NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
			viewer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			viewed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS reactions (
			id SERIAL PRIMARY KEY,
			story_id INTEGER NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			reaction_type VARCHAR(50) NOT NULL,
			reacted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS follows (
			follower_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			followed_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (follower_id, followed_id)
		);`,
	}

	for _, q := range queries {
		if _, err := p.Db.Exec(q); err != nil {
			return err
		}
	}

	return nil
}

func (p *Postgres) CreateStory(authorID, text, mediaKey string, visibility types.Visibility, audienceUserIDs []string) (string, error) {
	var storyID int
	query := `
	INSERT INTO stories (author_id, text, media_key, visibility)
	VALUES ($1, $2, $3, $4)
	RETURNING id
	`
	queryAudience := `
	INSERT INTO story_audience (story_id, user_id)
	VALUES ($1, $2)
	`

	// Start a transaction
	tx, err := p.Db.Begin()
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// Insert the story
	err = tx.QueryRow(query, authorID, text, mediaKey, visibility).Scan(&storyID)
	if err != nil {
		return "", err
	}

	// Insert audience user IDs if visibility is PRIVATE or FRIENDS
	if visibility == types.VisibilityPrivate || visibility == types.VisibilityFriends {
		for _, userID := range audienceUserIDs {
			_, err := tx.Exec(queryAudience, storyID, userID)
			if err != nil {
				return "", err
			}
		}
	}

	return fmt.Sprintf("%d", storyID), nil
}

func (p *Postgres) CreateUser(email, password string) (string, error) {
	var userID int
	query := `
	INSERT INTO users (email, password)
	VALUES ($1, $2)
	RETURNING id
	`

	err := p.Db.QueryRow(query, email, password).Scan(&userID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", userID), nil
}

func (p *Postgres) GetUserByEmail(email string) (string, string, error) {
	var userID int
	var hashedPassword string
	query := `
	SELECT id, password FROM users WHERE email = $1
	`

	err := p.Db.QueryRow(query, email).Scan(&userID, &hashedPassword)
	if err != nil {
		return "", "", err
	}

	return fmt.Sprintf("%d", userID), hashedPassword, nil
}

func (p *Postgres) GetAllPublicStories() ([]types.Story, error) {
	query := `
	SELECT id, author_id, text, media_key, visibility, created_at, expires_at
	FROM stories
	WHERE visibility = 'PUBLIC'
	ORDER BY created_at DESC
	`
	rows, err := p.Db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stories []types.Story
	for rows.Next() {
		var s types.Story
		err := rows.Scan(&s.ID, &s.AuthorID, &s.Text, &s.MediaKey, &s.Visibility, &s.CreatedAt, &s.ExpiresAt)
		if err != nil {
			return nil, err
		}
		stories = append(stories, s)
	}
	return stories, nil
}

func (p *Postgres) GetStoriesForUser(userID string) ([]types.Story, error) {
	query := `
	SELECT DISTINCT s.id, s.author_id, s.text, s.media_key, s.visibility, s.created_at, s.expires_at
	FROM stories s
	LEFT JOIN story_audience sa ON s.id = sa.story_id
	WHERE 
		s.visibility = 'PUBLIC'
		OR (s.visibility = 'FRIENDS' AND sa.user_id = $1)
		OR (s.visibility = 'PRIVATE' AND sa.user_id = $1)
		OR s.author_id = $1::integer
	ORDER BY s.created_at DESC
	`
	rows, err := p.Db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stories []types.Story
	for rows.Next() {
		var s types.Story
		err := rows.Scan(&s.ID, &s.AuthorID, &s.Text, &s.MediaKey, &s.Visibility, &s.CreatedAt, &s.ExpiresAt)
		if err != nil {
			return nil, err
		}
		stories = append(stories, s)
	}
	return stories, nil
}

func (p *Postgres) GetStoryByID(storyID string) (types.Story, error) {
	query := `
	SELECT id, author_id, text, media_key, visibility, created_at, expires_at
	FROM stories
	WHERE id = $1
	`
	var s types.Story
	err := p.Db.QueryRow(query, storyID).Scan(&s.ID, &s.AuthorID, &s.Text, &s.MediaKey, &s.Visibility, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		return s, err
	}
	return s, nil
}

func (p *Postgres) CanUserViewStory(storyID, userID string) (bool, error) {
	query := `
	SELECT s.visibility, s.author_id,
		   (CASE WHEN sa.user_id IS NOT NULL THEN true ELSE false END) AS in_audience
	FROM stories s
	LEFT JOIN story_audience sa ON s.id = sa.story_id AND sa.user_id = $2::integer
	WHERE s.id = $1
	`

	var visibility types.Visibility
	var authorID string
	var inAudience bool

	err := p.Db.QueryRow(query, storyID, userID).Scan(&visibility, &authorID, &inAudience)
	if err != nil {
		return false, err
	}

	// Check permission rules based on visibility and graph
	switch visibility {
	case types.VisibilityPublic:
		return true, nil
	case types.VisibilityFriends:
		// User can view if they are the author or in the audience
		return authorID == userID || inAudience, nil
	case types.VisibilityPrivate:
		// User can view if they are the author or in the audience
		return authorID == userID || inAudience, nil
	default:
		return false, nil
	}
}

func (p *Postgres) RecordStoryView(storyID, viewerID string) error {
	query := `
	INSERT INTO story_views (story_id, viewer_id)
	VALUES ($1, $2)
	ON CONFLICT (story_id, viewer_id) DO NOTHING
	`
	_, err := p.Db.Exec(query, storyID, viewerID)
	return err
}

func (p *Postgres) AddReaction(storyID, userID string, emoji types.ReactionType) error {
	// First, remove any existing reaction from this user for this story
	deleteQuery := `DELETE FROM reactions WHERE story_id = $1 AND user_id = $2`
	_, err := p.Db.Exec(deleteQuery, storyID, userID)
	if err != nil {
		return err
	}

	// Then add the new reaction
	insertQuery := `
	INSERT INTO reactions (story_id, user_id, reaction_type)
	VALUES ($1, $2, $3)
	`
	_, err = p.Db.Exec(insertQuery, storyID, userID, string(emoji))
	return err
}
