package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/princekumarofficial/stories-service/internal/storage"
	"github.com/princekumarofficial/stories-service/internal/types"
)

// CacheService wraps storage with Redis caching
type CacheService struct {
	storage storage.Storage
	redis   *redis.Client
}

// NewCacheService creates a new cache service
func NewCacheService(storage storage.Storage, redisClient *redis.Client) *CacheService {
	return &CacheService{
		storage: storage,
		redis:   redisClient,
	}
}

// Cache key patterns
const (
	UserFolloweesKey = "user:followees:%s" // user:followees:userID
	FeedCacheKey     = "feed:user:%s"      // feed:user:userID
	StoryKey         = "story:%s"          // story:storyID
	UserStatsKey     = "user:stats:%s"     // user:stats:userID
)

// Cache durations
const (
	FolloweesCacheDuration = 5 * time.Minute  // Followees don't change often
	FeedCacheDuration      = 45 * time.Second // Hot feed cache (30-60s)
	StoryCacheDuration     = 10 * time.Minute // Individual stories
	StatsCacheDuration     = 2 * time.Minute  // User stats
)

// GetUserFollowees returns cached followee IDs or fetches from DB
func (c *CacheService) GetUserFollowees(userID string) ([]string, error) {
	ctx := context.Background()
	key := fmt.Sprintf(UserFolloweesKey, userID)

	// Try cache first
	cached, err := c.redis.Get(ctx, key).Result()
	if err == nil {
		var followees []string
		if err := json.Unmarshal([]byte(cached), &followees); err == nil {
			return followees, nil
		}
	}

	// Cache miss - fetch from database
	followees, err := c.storage.GetUserFollowees(userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	data, _ := json.Marshal(followees)
	c.redis.Set(ctx, key, data, FolloweesCacheDuration)

	return followees, nil
}

func (c *CacheService) GetUserFollowers(userID string) ([]string, error) {
	// For now, just pass through to storage since this is less frequently accessed
	return c.storage.GetUserFollowers(userID)
}

// GetCachedFeed returns cached feed or fetches from DB
func (c *CacheService) GetCachedFeed(ctx context.Context, userID string) ([]types.Story, error) {
	key := fmt.Sprintf(FeedCacheKey, userID)

	// Try cache first
	cached, err := c.redis.Get(ctx, key).Result()
	if err == nil {
		var stories []types.Story
		if err := json.Unmarshal([]byte(cached), &stories); err == nil {
			return stories, nil
		}
	}

	// Cache miss - fetch from database (with optimizations)
	stories, err := c.storage.GetStoriesForUser(userID)
	if err != nil {
		return nil, err
	}

	// Cache the result for 30-60 seconds
	data, _ := json.Marshal(stories)
	c.redis.Set(ctx, key, data, FeedCacheDuration)

	return stories, nil
}

// InvalidateUserCache clears user-related caches
func (c *CacheService) InvalidateUserCache(ctx context.Context, userID string) {
	keys := []string{
		fmt.Sprintf(UserFolloweesKey, userID),
		fmt.Sprintf(FeedCacheKey, userID),
		fmt.Sprintf(UserStatsKey, userID),
	}

	for _, key := range keys {
		c.redis.Del(ctx, key)
	}
}

// InvalidateFeedCaches clears feed caches for multiple users
func (c *CacheService) InvalidateFeedCaches(ctx context.Context, userIDs []string) {
	if len(userIDs) == 0 {
		return
	}

	keys := make([]string, len(userIDs))
	for i, userID := range userIDs {
		keys[i] = fmt.Sprintf(FeedCacheKey, userID)
	}

	c.redis.Del(ctx, keys...)
}

// CacheStory caches an individual story
func (c *CacheService) CacheStory(ctx context.Context, story types.Story) {
	key := fmt.Sprintf(StoryKey, story.ID)
	data, _ := json.Marshal(story)
	c.redis.Set(ctx, key, data, StoryCacheDuration)
}

// GetCachedStory returns cached story or fetches from DB
func (c *CacheService) GetCachedStory(ctx context.Context, storyID string) (types.Story, error) {
	key := fmt.Sprintf(StoryKey, storyID)

	// Try cache first
	cached, err := c.redis.Get(ctx, key).Result()
	if err == nil {
		var story types.Story
		if err := json.Unmarshal([]byte(cached), &story); err == nil {
			return story, nil
		}
	}

	// Cache miss - fetch from database
	story, err := c.storage.GetStoryByID(storyID)
	if err != nil {
		return story, err
	}

	// Cache the result
	c.CacheStory(ctx, story)

	return story, nil
}

// GetCachedUserStats returns cached user stats or fetches from DB
func (c *CacheService) GetCachedUserStats(ctx context.Context, userID string) (int, int, int, map[string]int, error) {
	key := fmt.Sprintf(UserStatsKey, userID)

	// Try cache first
	cached, err := c.redis.Get(ctx, key).Result()
	if err == nil {
		var stats struct {
			Posted         int            `json:"posted"`
			Views          int            `json:"views"`
			UniqueViewers  int            `json:"unique_viewers"`
			ReactionCounts map[string]int `json:"reaction_counts"`
		}
		if err := json.Unmarshal([]byte(cached), &stats); err == nil {
			return stats.Posted, stats.Views, stats.UniqueViewers, stats.ReactionCounts, nil
		}
	}

	// Cache miss - fetch from database
	posted, views, uniqueViewers, reactionCounts, err := c.storage.GetUserStats(userID)
	if err != nil {
		return 0, 0, 0, nil, err
	}

	// Cache the result
	stats := struct {
		Posted         int            `json:"posted"`
		Views          int            `json:"views"`
		UniqueViewers  int            `json:"unique_viewers"`
		ReactionCounts map[string]int `json:"reaction_counts"`
	}{
		Posted:         posted,
		Views:          views,
		UniqueViewers:  uniqueViewers,
		ReactionCounts: reactionCounts,
	}

	data, _ := json.Marshal(stats)
	c.redis.Set(ctx, key, data, StatsCacheDuration)

	return posted, views, uniqueViewers, reactionCounts, nil
}

// Methods to pass through to storage (implement storage.Storage interface)
func (c *CacheService) CreateStory(authorID, text, mediaKey string, visibility types.Visibility, audienceUserIDs []string) (string, error) {
	storyID, err := c.storage.CreateStory(authorID, text, mediaKey, visibility, audienceUserIDs)
	if err != nil {
		return "", err
	}

	// Invalidate relevant caches
	ctx := context.Background()
	c.InvalidateUserCache(ctx, authorID)

	// Invalidate feed caches for followers if public/friends story
	if visibility == types.VisibilityPublic || visibility == types.VisibilityFriends {
		followers, _ := c.GetUserFollowers(authorID)
		c.InvalidateFeedCaches(ctx, followers)
	}

	// Invalidate specific users for private stories
	if visibility == types.VisibilityPrivate {
		c.InvalidateFeedCaches(ctx, audienceUserIDs)
	}

	return storyID, nil
}

func (c *CacheService) CreateUser(email, password string) (string, error) {
	return c.storage.CreateUser(email, password)
}

func (c *CacheService) GetUserByEmail(email string) (string, string, error) {
	return c.storage.GetUserByEmail(email)
}

func (c *CacheService) GetAllPublicStories() ([]types.Story, error) {
	return c.storage.GetAllPublicStories()
}

func (c *CacheService) GetStoriesForUser(userID string) ([]types.Story, error) {
	ctx := context.Background()
	return c.GetCachedFeed(ctx, userID)
}

func (c *CacheService) GetStoryByID(storyID string) (types.Story, error) {
	ctx := context.Background()
	return c.GetCachedStory(ctx, storyID)
}

func (c *CacheService) CanUserViewStory(storyID, userID string) (bool, error) {
	return c.storage.CanUserViewStory(storyID, userID)
}

func (c *CacheService) RecordStoryView(storyID, viewerID string) error {
	return c.storage.RecordStoryView(storyID, viewerID)
}

func (c *CacheService) AddReaction(storyID, userID string, emoji types.ReactionType) error {
	return c.storage.AddReaction(storyID, userID, emoji)
}

func (c *CacheService) GetUserStats(userID string) (int, int, int, map[string]int, error) {
	ctx := context.Background()
	return c.GetCachedUserStats(ctx, userID)
}

func (c *CacheService) FollowUser(followerID, followedID string) error {
	err := c.storage.FollowUser(followerID, followedID)
	if err != nil {
		return err
	}

	// Invalidate relevant caches
	ctx := context.Background()
	c.InvalidateUserCache(ctx, followerID) // Follower's feed will change
	c.InvalidateUserCache(ctx, followedID) // Followed user's follower list changed

	return nil
}

func (c *CacheService) UnfollowUser(followerID, followedID string) error {
	err := c.storage.UnfollowUser(followerID, followedID)
	if err != nil {
		return err
	}

	// Invalidate relevant caches
	ctx := context.Background()
	c.InvalidateUserCache(ctx, followerID) // Follower's feed will change
	c.InvalidateUserCache(ctx, followedID) // Followed user's follower list changed

	return nil
}

func (c *CacheService) IsFollowing(followerID, followedID string) (bool, error) {
	return c.storage.IsFollowing(followerID, followedID)
}

func (c *CacheService) SoftDeleteExpiredStories() (int, error) {
	return c.storage.SoftDeleteExpiredStories()
}
