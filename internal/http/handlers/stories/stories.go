package stories

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/princekumarofficial/stories-service/internal/storage"
	"github.com/princekumarofficial/stories-service/internal/types"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
)

func Feed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is the feed endpoint"))
	}
}

func PostStory(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		storyID, err := storage.CreateStory("1", story.Text, story.MediaKey, story.Visibility, story.AudienceUserIDs)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}
		slog.Info("Story created with ID:", slog.String("story_id", storyID))

		response.WriteJSON(w, http.StatusCreated, map[string]string{"id": storyID})
	}
}
