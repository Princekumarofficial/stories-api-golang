package cache

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/princekumarofficial/stories-service/internal/types"
)

// OptimizedFeedQuery represents an optimized feed with preloaded data
type OptimizedFeedQuery struct {
	db *sql.DB
}

// NewOptimizedFeedQuery creates a new optimized feed query service
func NewOptimizedFeedQuery(db *sql.DB) *OptimizedFeedQuery {
	return &OptimizedFeedQuery{db: db}
}

// GetOptimizedFeedForUser returns feed with preloaded author data and counters
// This avoids N+1 queries by joining all necessary data in a single query
func (ofq *OptimizedFeedQuery) GetOptimizedFeedForUser(ctx context.Context, userID string) ([]types.StoryWithMeta, error) {
	query := `
	WITH user_stories AS (
		SELECT DISTINCT s.id, s.author_id, s.text, s.media_key, s.visibility, s.created_at, s.expires_at, s.deleted_at
		FROM stories s
		LEFT JOIN story_audience sa ON s.id = sa.story_id
		LEFT JOIN follows f ON s.author_id = f.followed_id
		WHERE 
			s.deleted_at IS NULL 
			AND s.expires_at > NOW()  -- Only non-expired stories
			AND (
				s.visibility = 'PUBLIC'
				OR (s.visibility = 'FRIENDS' AND f.follower_id = $1::integer)
				OR (s.visibility = 'PRIVATE' AND sa.user_id = $1)
				OR s.author_id = $1::integer
			)
	),
	story_stats AS (
		SELECT 
			s.id as story_id,
			COUNT(DISTINCT sv.viewer_id) as view_count,
			COUNT(DISTINCT r.user_id) as reaction_count,
			COALESCE(
				JSON_OBJECT_AGG(
					r.reaction_type, 
					reaction_type_count
				) FILTER (WHERE r.reaction_type IS NOT NULL), 
				'{}'::json
			) as reaction_breakdown
		FROM user_stories s
		LEFT JOIN story_views sv ON s.id = sv.story_id
		LEFT JOIN (
			SELECT 
				story_id, 
				reaction_type, 
				COUNT(*) as reaction_type_count
			FROM reactions 
			GROUP BY story_id, reaction_type
		) r ON s.id = r.story_id
		GROUP BY s.id
	)
	SELECT 
		us.id,
		us.author_id,
		us.text,
		us.media_key,
		us.visibility,
		us.created_at,
		us.expires_at,
		COALESCE(us.deleted_at::TEXT, '') as deleted_at,
		-- Author email (for display)
		u.email as author_email,
		-- Story stats
		COALESCE(ss.view_count, 0) as view_count,
		COALESCE(ss.reaction_count, 0) as reaction_count,
		COALESCE(ss.reaction_breakdown::text, '{}') as reaction_breakdown,
		-- User interaction flags
		EXISTS(
			SELECT 1 FROM story_views sv2 
			WHERE sv2.story_id = us.id AND sv2.viewer_id = $1
		) as user_has_viewed,
		COALESCE(
			(SELECT reaction_type FROM reactions r2 
			 WHERE r2.story_id = us.id AND r2.user_id = $1), 
			''
		) as user_reaction
	FROM user_stories us
	LEFT JOIN users u ON us.author_id = u.id
	LEFT JOIN story_stats ss ON us.id = ss.story_id
	ORDER BY us.created_at DESC
	LIMIT 50  -- Reasonable feed limit
	`

	rows, err := ofq.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch optimized feed: %w", err)
	}
	defer rows.Close()

	var stories []types.StoryWithMeta
	for rows.Next() {
		var story types.StoryWithMeta
		var reactionBreakdownJSON string

		err := rows.Scan(
			&story.ID,
			&story.AuthorID,
			&story.Text,
			&story.MediaKey,
			&story.Visibility,
			&story.CreatedAt,
			&story.ExpiresAt,
			&story.DeletedAt,
			&story.AuthorEmail,
			&story.ViewCount,
			&story.ReactionCount,
			&reactionBreakdownJSON,
			&story.UserHasViewed,
			&story.UserReaction,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan story: %w", err)
		}

		// Parse reaction breakdown JSON
		if reactionBreakdownJSON != "" && reactionBreakdownJSON != "{}" {
			// You can implement JSON parsing here if needed
			// For now, we'll store it as a string
		}

		stories = append(stories, story)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return stories, nil
}

// GetOptimizedStoryByID returns a single story with all metadata
func (ofq *OptimizedFeedQuery) GetOptimizedStoryByID(ctx context.Context, storyID, userID string) (types.StoryWithMeta, error) {
	query := `
	WITH story_stats AS (
		SELECT 
			s.id as story_id,
			COUNT(DISTINCT sv.viewer_id) as view_count,
			COUNT(DISTINCT r.user_id) as reaction_count,
			COALESCE(
				JSON_OBJECT_AGG(
					r.reaction_type, 
					reaction_type_count
				) FILTER (WHERE r.reaction_type IS NOT NULL), 
				'{}'::json
			) as reaction_breakdown
		FROM stories s
		LEFT JOIN story_views sv ON s.id = sv.story_id
		LEFT JOIN (
			SELECT 
				story_id, 
				reaction_type, 
				COUNT(*) as reaction_type_count
			FROM reactions 
			GROUP BY story_id, reaction_type
		) r ON s.id = r.story_id
		WHERE s.id = $1
		GROUP BY s.id
	)
	SELECT 
		s.id,
		s.author_id,
		s.text,
		s.media_key,
		s.visibility,
		s.created_at,
		s.expires_at,
		COALESCE(s.deleted_at::TEXT, '') as deleted_at,
		-- Author email (for display)
		u.email as author_email,
		-- Story stats
		COALESCE(ss.view_count, 0) as view_count,
		COALESCE(ss.reaction_count, 0) as reaction_count,
		COALESCE(ss.reaction_breakdown::text, '{}') as reaction_breakdown,
		-- User interaction flags
		EXISTS(
			SELECT 1 FROM story_views sv2 
			WHERE sv2.story_id = s.id AND sv2.viewer_id = $2
		) as user_has_viewed,
		COALESCE(
			(SELECT reaction_type FROM reactions r2 
			 WHERE r2.story_id = s.id AND r2.user_id = $2), 
			''
		) as user_reaction
	FROM stories s
	LEFT JOIN users u ON s.author_id = u.id
	LEFT JOIN story_stats ss ON s.id = ss.story_id
	WHERE s.id = $1 AND s.deleted_at IS NULL
	`

	var story types.StoryWithMeta
	var reactionBreakdownJSON string

	err := ofq.db.QueryRowContext(ctx, query, storyID, userID).Scan(
		&story.ID,
		&story.AuthorID,
		&story.Text,
		&story.MediaKey,
		&story.Visibility,
		&story.CreatedAt,
		&story.ExpiresAt,
		&story.DeletedAt,
		&story.AuthorEmail,
		&story.ViewCount,
		&story.ReactionCount,
		&reactionBreakdownJSON,
		&story.UserHasViewed,
		&story.UserReaction,
	)

	if err != nil {
		return story, fmt.Errorf("failed to fetch optimized story: %w", err)
	}

	return story, nil
}
