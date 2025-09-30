package postgres

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/princekumarofficial/stories-service/internal/config"
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
	query := `
	CREATE TABLE IF NOT EXISTS stories (
		id SERIAL PRIMARY KEY,
		author_id VARCHAR(255) NOT NULL,
		text TEXT,
		media_key VARCHAR(255),
		visibility VARCHAR(50) NOT NULL,
		audience_user_ids TEXT[],
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := p.Db.Exec(query)
	return err
}

func (p *Postgres) CreateStory(authorID, text, mediaKey, visibility string, audienceUserIDs []string) (string, error) {
	var storyID string
	query := `
	INSERT INTO stories (author_id, text, media_key, visibility, audience_user_ids)
	VALUES ($1, $2, $3, $4, $5)
	`
	stmt, err := p.Db.Prepare(query)
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	result, err := stmt.Exec(authorID, text, mediaKey, visibility, pq.Array(audienceUserIDs))
	if err != nil {
		return "", err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return "", err
	}

	storyID = fmt.Sprintf("%d", id)
	return storyID, nil
}
