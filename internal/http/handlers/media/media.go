package media

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/princekumarofficial/stories-service/internal/http/middleware"
	mediaService "github.com/princekumarofficial/stories-service/internal/services/media"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
)

type MediaHandlers struct {
	mediaService *mediaService.Service
}

type UploadURLRequest struct {
	ContentType string `json:"content_type" validate:"required"`
}

type UploadURLResponse struct {
	ObjectKey   string `json:"object_key"`
	UploadURL   string `json:"upload_url"`
	ExpiresAt   int64  `json:"expires_at"`
	MaxFileSize int64  `json:"max_file_size"`
	ContentType string `json:"content_type"`
}

type MediaInfoResponse struct {
	ObjectKey   string    `json:"object_key"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	UploadedAt  time.Time `json:"uploaded_at"`
	MediaURL    string    `json:"media_url"`
}

// NewMediaHandlers creates a new media handlers instance
func NewMediaHandlers(mediaService *mediaService.Service) *MediaHandlers {
	return &MediaHandlers{
		mediaService: mediaService,
	}
}

// GenerateUploadURL generates a presigned URL for media upload
// @Summary Generate presigned upload URL
// @Description Generate a presigned URL for uploading media files
// @Tags media
// @Accept json
// @Produce json
// @Param request body UploadURLRequest true "Upload URL request"
// @Success 200 {object} UploadURLResponse "Upload URL generated successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /media/upload-url [post]
func (h *MediaHandlers) GenerateUploadURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		var req UploadURLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("invalid request body")))
			return
		}

		// Generate presigned upload URL
		uploadInfo, err := h.mediaService.GeneratePresignedUploadURL(userID, req.ContentType)
		if err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		resp := UploadURLResponse{
			ObjectKey:   uploadInfo.ObjectKey,
			UploadURL:   uploadInfo.UploadURL,
			ExpiresAt:   uploadInfo.ExpiresAt,
			MaxFileSize: uploadInfo.MaxFileSize,
			ContentType: uploadInfo.ContentType,
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Upload URL generated successfully", resp))
	}
}

// GetMediaInfo retrieves information about a media file
// @Summary Get media file information
// @Description Get information about a specific media file
// @Tags media
// @Produce json
// @Param object_key path string true "Object key"
// @Success 200 {object} MediaInfoResponse "Media information retrieved successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Media not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /media/{object_key}/info [get]
func (h *MediaHandlers) GetMediaInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		_, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		// Get object key from URL path
		objectKey := r.URL.Path[len("/media/"):]
		if objectKey == "" {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("object key is required")))
			return
		}

		// Remove "/info" suffix
		if len(objectKey) > 5 && objectKey[len(objectKey)-5:] == "/info" {
			objectKey = objectKey[:len(objectKey)-5]
		}

		// Get object information
		objInfo, err := h.mediaService.GetObjectInfo(objectKey)
		if err != nil {
			response.WriteJSON(w, http.StatusNotFound, response.GeneralError(errors.New("media not found")))
			return
		}

		// Generate media URL
		mediaURL := h.mediaService.GetMediaURL(objectKey)

		resp := MediaInfoResponse{
			ObjectKey:   objectKey,
			Size:        objInfo.Size,
			ContentType: objInfo.ContentType,
			UploadedAt:  objInfo.LastModified,
			MediaURL:    mediaURL,
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Media information retrieved successfully", resp))
	}
}

// GenerateDownloadURL generates a presigned URL for media download
// @Summary Generate presigned download URL
// @Description Generate a presigned URL for downloading media files
// @Tags media
// @Produce json
// @Param object_key path string true "Object key"
// @Param expires query int false "Expiration time in seconds (default: 3600)"
// @Success 200 {object} map[string]interface{} "Download URL generated successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Media not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /media/{object_key}/download-url [get]
func (h *MediaHandlers) GenerateDownloadURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		_, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		// Get object key from URL path
		objectKey := r.URL.Path[len("/media/"):]
		if objectKey == "" {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("object key is required")))
			return
		}

		// Remove "/download-url" suffix
		if len(objectKey) > 13 && objectKey[len(objectKey)-13:] == "/download-url" {
			objectKey = objectKey[:len(objectKey)-13]
		}

		// Parse expiration time
		expiresParam := r.URL.Query().Get("expires")
		expires := 3600 // default 1 hour
		if expiresParam != "" {
			if parsedExpires, err := strconv.Atoi(expiresParam); err == nil && parsedExpires > 0 {
				expires = parsedExpires
			}
		}

		// Generate presigned download URL
		downloadURL, err := h.mediaService.GeneratePresignedDownloadURL(objectKey, time.Duration(expires)*time.Second)
		if err != nil {
			response.WriteJSON(w, http.StatusNotFound, response.GeneralError(errors.New("failed to generate download URL")))
			return
		}

		resp := map[string]interface{}{
			"object_key":   objectKey,
			"download_url": downloadURL.String(),
			"expires_at":   time.Now().Add(time.Duration(expires) * time.Second).Unix(),
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Download URL generated successfully", resp))
	}
}

// ListUserMedia lists all media files for the authenticated user
// @Summary List user media files
// @Description List all media files uploaded by the authenticated user
// @Tags media
// @Produce json
// @Success 200 {array} MediaInfoResponse "Media files retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /media [get]
func (h *MediaHandlers) ListUserMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		// List user media files
		objects, err := h.mediaService.ListUserMedia(userID)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(errors.New("failed to list media files")))
			return
		}

		var mediaFiles []MediaInfoResponse
		for _, obj := range objects {
			mediaURL := h.mediaService.GetMediaURL(obj.Key)
			mediaFiles = append(mediaFiles, MediaInfoResponse{
				ObjectKey:   obj.Key,
				Size:        obj.Size,
				ContentType: obj.ContentType,
				UploadedAt:  obj.LastModified,
				MediaURL:    mediaURL,
			})
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Media files retrieved successfully", mediaFiles))
	}
}

// DeleteMedia deletes a media file
// @Summary Delete media file
// @Description Delete a specific media file
// @Tags media
// @Param object_key path string true "Object key"
// @Success 200 {object} response.Response "Media file deleted successfully"
// @Failure 400 {object} response.Response "Bad request"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 404 {object} response.Response "Media not found"
// @Failure 500 {object} response.Response "Internal server error"
// @Security BearerAuth
// @Router /media/{object_key} [delete]
func (h *MediaHandlers) DeleteMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		// Get object key from URL path
		objectKey := r.URL.Path[len("/media/"):]
		if objectKey == "" {
			response.WriteJSON(w, http.StatusBadRequest, response.GeneralError(errors.New("object key is required")))
			return
		}

		// Verify that the object belongs to the user (basic security check)
		expectedPrefix := "users/" + userID + "/media/"
		if len(objectKey) < len(expectedPrefix) || objectKey[:len(expectedPrefix)] != expectedPrefix {
			response.WriteJSON(w, http.StatusForbidden, response.GeneralError(errors.New("access denied")))
			return
		}

		// Delete the object
		err := h.mediaService.DeleteObject(objectKey)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(errors.New("failed to delete media file")))
			return
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Media file deleted successfully", nil))
	}
}
