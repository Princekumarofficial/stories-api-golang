package users

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/princekumarofficial/stories-service/internal/storage"
	"github.com/princekumarofficial/stories-service/internal/types/users"
	"github.com/princekumarofficial/stories-service/internal/utils/password"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
)

// SignUp handles user registration
// @Summary Register a new user
// @Tags users
// @Accept json
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
// @Tags users
// @Accept json
// @Param user body users.SignInRequest true "User login details"
// @Success 200 {object} map[string]string "User authenticated successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Router /login [post]
func Login(storage storage.Storage) http.HandlerFunc {
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

		response.WriteJSON(w, http.StatusOK, map[string]string{
			"user_id": userID,
		})
	}
}
