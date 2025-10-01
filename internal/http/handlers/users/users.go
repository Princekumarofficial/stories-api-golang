package users

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/princekumarofficial/stories-service/internal/http/middleware"
	"github.com/princekumarofficial/stories-service/internal/storage"
	"github.com/princekumarofficial/stories-service/internal/types/users"
	"github.com/princekumarofficial/stories-service/internal/utils/jwt"
	"github.com/princekumarofficial/stories-service/internal/utils/password"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
)

// SignUp handles user registration
// @Summary Register a new user
// @Description Register a new user account
// @Tags users
// @Accept json
// @Produce json
// @Param user body users.SignUpRequest true "User registration details"
// @Success 201 {object} map[string]string "User created successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /signup [post]
func SignUp(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var signupReq users.SignUpRequest

		err := json.NewDecoder(r.Body).Decode(&signupReq)
		if err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		// Validate request
		validate := validator.New()
		err = validate.Struct(signupReq)
		if err != nil {
			if ve, ok := err.(validator.ValidationErrors); ok {
				response.WriteJSON(w, http.StatusBadRequest, response.ValidationError(ve))
				return
			}
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		hashedPassword, err := password.HashPassword(signupReq.Password)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(errors.New("failed to hash password")))
			return
		}

		userID, err := storage.CreateUser(signupReq.Email, hashedPassword)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}
		slog.Info("User created with ID:", slog.String("user_id", userID))

		response.WriteJSON(w, http.StatusCreated, map[string]string{
			"id": userID,
		})
	}
}

// Login handles user authentication
// @Summary Authenticate a user
// @Description Authenticate a user and return JWT token
// @Tags users
// @Accept json
// @Produce json
// @Param user body users.SignInRequest true "User login details"
// @Success 200 {object} map[string]string "User authenticated successfully with token"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Router /login [post]
func Login(storage storage.Storage, JWTSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var signinReq users.SignInRequest

		err := json.NewDecoder(r.Body).Decode(&signinReq)
		if err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		// Validate request
		validate := validator.New()
		err = validate.Struct(signinReq)
		if err != nil {
			if ve, ok := err.(validator.ValidationErrors); ok {
				response.WriteJSON(w, http.StatusBadRequest, response.ValidationError(ve))
				return
			}
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		// Authentication logic
		userID, hashedPassword, err := storage.GetUserByEmail(signinReq.Email)
		if err != nil {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("invalid email or password")))
			return
		}

		correctPassword := password.CheckPasswordHash(signinReq.Password, hashedPassword)
		if !correctPassword {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("invalid email or password")))
			return
		}
		token, err := jwt.CreateToken(userID, JWTSecret)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(errors.New("failed to generate token")))
			return
		}

		response.WriteJSON(w, http.StatusOK, map[string]string{
			"user_id": userID,
			"token":   token,
		})
	}
}

// GetStats returns user statistics for the last 7 days
// @Summary Get user statistics
// @Description Get user statistics including posts, views, unique viewers, and reaction breakdown for the last 7 days
// @Tags users
// @Produce json
// @Success 200 {object} users.UserStats "User statistics"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /me/stats [get]
func GetStats(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context (set by auth middleware)
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("unauthorized")))
			return
		}

		// Get user stats from storage
		posted, views, uniqueViewers, reactionCounts, err := storage.GetUserStats(userID)
		if err != nil {
			slog.Error("Failed to get user stats", slog.String("error", err.Error()), slog.String("user_id", userID))
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(errors.New("failed to get user stats")))
			return
		}

		// Create response
		stats := users.UserStats{
			Posted:         posted,
			Views:          views,
			UniqueViewers:  uniqueViewers,
			ReactionCounts: reactionCounts,
		}

		response.WriteJSON(w, http.StatusOK, stats)
	}
}
