package stories

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/princekumarofficial/stories-service/internal/events"
	"github.com/princekumarofficial/stories-service/internal/http/middleware"
	"github.com/princekumarofficial/stories-service/internal/storage"
	"github.com/princekumarofficial/stories-service/internal/types"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
)

// Feed handles the stories feed endpoint
// @Summary Get stories feed
// @Tags stories
// @Security BearerAuth
// @Param limit query int false "Page size (1-100, default 50)"
// @Param cursor query string false "next_cursor from the previous page"
// @Router /feed [get]
func Feed(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		query := r.URL.Query()
		limit := types.DefaultFeedLimit
		if query.Has("limit") {
			n, err := strconv.Atoi(query.Get("limit"))
			if err != nil || n < 1 || n > 100 {
				response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("limit must be between 1 and 100")))
				return
			}
			limit = n
		}
		var before *types.FeedCursor
		if query.Has("cursor") {
			var err error
			before, err = decodeFeedCursor(query.Get("cursor"))
			if err != nil {
				response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(err))
				return
			}
		}
		stories, err := storage.GetFeedPage(r.Context(), userID, before, limit)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		var next string
		if len(stories) > limit {
			stories = stories[:limit]
			last := stories[limit-1]
			next = base64.RawURLEncoding.EncodeToString([]byte(last.CreatedAt + "|" + last.ID))
		}
		response.WriteJSON(w, http.StatusOK, struct {
			response.Response
			NextCursor string `json:"next_cursor,omitempty"`
		}{response.RequestOK("Cached feed retrieved successfully", stories), next})
	}
}

func decodeFeedCursor(token string) (*types.FeedCursor, error) {
	invalid := errors.New("invalid feed cursor")
	if len(token) > 96 {
		return nil, invalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, invalid
	}
	stamp, idText, ok := strings.Cut(string(raw), "|")
	id, err := strconv.ParseInt(idText, 10, 32)
	if !ok || err != nil || id < 1 {
		return nil, invalid
	}
	if _, err := time.Parse(time.RFC3339Nano, stamp); err != nil {
		return nil, invalid
	}
	return &types.FeedCursor{CreatedAt: stamp, ID: int(id)}, nil
}

// PostStory handles creating a new story
// @Summary Create a new story
// @Description Create a new story with authentication required
// @Tags stories
// @Accept json
// @Produce json
// @Param story body types.StoryPostRequest true "Story content"
// @Success 201 {object} map[string]string "Story created successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /stories [post]
func PostStory(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		var story types.StoryPostRequest

		err := json.NewDecoder(r.Body).Decode(&story)
		if errors.Is(err, io.EOF) {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("request body cannot be empty")))
			return
		} else if err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		// Validate request
		validate := validator.New()
		err = validate.Struct(story)
		if err != nil {
			if ve, ok := err.(validator.ValidationErrors); ok {
				response.WriteJSON(w, http.StatusBadRequest, response.ValidationError(ve))
				return
			}
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		storyID, err := storage.CreateStory(userID, story.Text, story.MediaKey, story.Visibility, story.AudienceUserIDs)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}
		slog.Info("Story created with ID:", slog.String("story_id", storyID))

		response.WriteJSON(w, http.StatusCreated, map[string]string{"id": storyID})
	}
}

// ViewStory handles recording a story view
// @Summary Record a story view
// @Description Record that a user has viewed a story (idempotent - one view per user)
// @Tags stories
// @Param id path string true "Story ID"
// @Success 200 {object} response.Response "View recorded successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Story not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /stories/{id}/view [post]
func ViewStory(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		storyID := r.PathValue("id")
		if storyID == "" {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("story ID is required")))
			return
		}

		// Verify story exists before recording view
		_, err := storage.GetStoryByID(storyID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.WriteJSON(w, http.StatusNotFound, response.GeneralError(errors.New("story not found")))
				return
			}
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		err = storage.RecordStoryView(storyID, userID)
		if err != nil {
			slog.Error("Failed to record story view", slog.String("error", err.Error()))
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("View recorded successfully", nil))
	}
}

func isValidReactionEmoji(emoji types.ReactionType) bool {
	switch emoji {
	case types.ReactionThumbsUp, types.ReactionHeart, types.ReactionLaugh,
		types.ReactionSurprised, types.ReactionSad, types.ReactionFire:
		return true
	default:
		return false
	}
}

// AddReaction handles adding a reaction to a story
// @Summary Add a reaction to a story
// @Description Add an emoji reaction to a story (replaces existing reaction if any)
// @Tags stories
// @Accept json
// @Produce json
// @Param id path string true "Story ID"
// @Param reaction body types.ReactionRequest true "Reaction emoji"
// @Success 200 {object} response.Response "Reaction added successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Story not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /stories/{id}/reactions [post]
func AddReaction(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		storyID := r.PathValue("id")
		if storyID == "" {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("story ID is required")))
			return
		}

		var reactionReq types.ReactionRequest
		err := json.NewDecoder(r.Body).Decode(&reactionReq)
		if errors.Is(err, io.EOF) {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("request body cannot be empty")))
			return
		} else if err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		// Validate the emoji
		if !isValidReactionEmoji(reactionReq.Emoji) {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("invalid emoji: must be one of 👍 ❤️ 😂 😮 😢 🔥")))
			return
		}

		// Verify story exists before adding reaction
		_, err = storage.GetStoryByID(storyID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.WriteJSON(w, http.StatusNotFound, response.GeneralError(errors.New("story not found")))
				return
			}
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		err = storage.AddReaction(storyID, userID, reactionReq.Emoji)
		if err != nil {
			slog.Error("Failed to add reaction", slog.String("error", err.Error()))
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Reaction added successfully", nil))
	}
}

// GetStory handles retrieving a specific story by ID
// @Summary Get a story by ID
// @Description Get a specific story by its ID with permission checks based on visibility and graph
// @Tags stories
// @Param id path string true "Story ID"
// @Success 200 {object} response.Response "Story retrieved successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 403 {object} response.Response "Forbidden - no permission to view this story"
// @Failure 404 {object} response.Response "Story not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /stories/{id} [get]
func GetStory(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		storyID := r.PathValue("id")
		if storyID == "" {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("story ID is required")))
			return
		}

		// Check if user can view this story
		canView, err := storage.CanUserViewStory(storyID, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.WriteJSON(w, http.StatusNotFound, response.GeneralError(errors.New("story not found")))
				return
			}
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		if !canView {
			response.WriteJSON(w, http.StatusForbidden, response.GeneralError(errors.New("you don't have permission to view this story")))
			return
		}

		// Get the story
		story, err := storage.GetStoryByID(storyID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.WriteJSON(w, http.StatusNotFound, response.GeneralError(errors.New("story not found")))
				return
			}
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Story retrieved successfully", story))
	}
}

// ViewStoryWithEvents handles recording a story view with real-time events
// @Summary Record a story view with real-time notifications
// @Description Record that a user has viewed a story (idempotent - one view per user) and send real-time notification to author
// @Tags stories
// @Param id path string true "Story ID"
// @Success 200 {object} response.Response "View recorded successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Story not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /stories/{id}/view [post]
func ViewStoryWithEvents(storage storage.Storage, eventPublisher *events.EventPublisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		storyID := r.PathValue("id")
		if storyID == "" {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("story ID is required")))
			return
		}

		// Get story to find the author ID
		story, err := storage.GetStoryByID(storyID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.WriteJSON(w, http.StatusNotFound, response.GeneralError(errors.New("story not found")))
				return
			}
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		// Record the view in database
		err = storage.RecordStoryView(storyID, userID)
		if err != nil {
			slog.Error("Failed to record story view", slog.String("error", err.Error()))
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		// Publish real-time event (fire and forget)
		go func() {
			err := eventPublisher.PublishStoryViewed(storyID, userID, story.AuthorID)
			if err != nil {
				slog.Error("Failed to publish story viewed event", slog.String("error", err.Error()))
			}
		}()

		response.WriteJSON(w, http.StatusOK, response.RequestOK("View recorded successfully", nil))
	}
}

// AddReactionWithEvents handles adding a reaction to a story with real-time events
// @Summary Add a reaction to a story with real-time notifications
// @Description Add an emoji reaction to a story and send real-time notification to author
// @Tags stories
// @Accept json
// @Produce json
// @Param id path string true "Story ID"
// @Param reaction body types.ReactionRequest true "Reaction details"
// @Success 200 {object} response.Response "Reaction added successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Story not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /stories/{id}/reactions [post]
func AddReactionWithEvents(storage storage.Storage, eventPublisher *events.EventPublisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		storyID := r.PathValue("id")
		if storyID == "" {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("story ID is required")))
			return
		}

		var reactionReq types.ReactionRequest
		err := json.NewDecoder(r.Body).Decode(&reactionReq)
		if errors.Is(err, io.EOF) {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("request body cannot be empty")))
			return
		} else if err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		// Validate the emoji
		if !isValidReactionEmoji(reactionReq.Emoji) {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("invalid emoji: must be one of 👍 ❤️ 😂 😮 😢 🔥")))
			return
		}

		// Get story to find the author ID
		story, err := storage.GetStoryByID(storyID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.WriteJSON(w, http.StatusNotFound, response.GeneralError(errors.New("story not found")))
				return
			}
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		// Add reaction to database
		err = storage.AddReaction(storyID, userID, reactionReq.Emoji)
		if err != nil {
			slog.Error("Failed to add reaction", slog.String("error", err.Error()))
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		// Publish real-time event (fire and forget)
		go func() {
			err := eventPublisher.PublishStoryReacted(storyID, userID, story.AuthorID, reactionReq.Emoji)
			if err != nil {
				slog.Error("Failed to publish story reacted event", slog.String("error", err.Error()))
			}
		}()

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Reaction added successfully", nil))
	}
}
