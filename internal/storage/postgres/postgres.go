package postgres

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
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
			visibility VARCHAR(50) NOT NULL CHECK (visibility IN ('FRIENDS','PRIVATE', 'PUBLIC')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			expires_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP + INTERVAL '24 hours'),
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
	INSERT INTO stories (author_id, text, media_key, visibility, audience_user_ids)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id
	`

	err := p.Db.QueryRow(query, authorID, text, mediaKey, visibility, pq.Array(audienceUserIDs)).Scan(&storyID)
	if err != nil {
		return "", err
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
