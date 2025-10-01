package stories

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/princekumarofficial/stories-service/internal/http/middleware"
	"github.com/princekumarofficial/stories-service/internal/storage"
	"github.com/princekumarofficial/stories-service/internal/types"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
)

// Feed handles the stories feed endpoint
// @Summary Get stories feed
// @Tags stories
// @Security BearerAuth
// @Router /feed [get]
func Feed(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		stories, err := storage.GetStoriesForUser(userID)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Stories fetched successfully", stories))
	}
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
