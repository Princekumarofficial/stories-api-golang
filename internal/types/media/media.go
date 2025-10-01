package media

import "time"

// MediaUpload represents a media file upload record in the database
type MediaUpload struct {
	ID          uint64    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	ObjectKey   string    `json:"object_key" db:"object_key"`
	FileName    string    `json:"file_name" db:"file_name"`
	ContentType string    `json:"content_type" db:"content_type"`
	Size        int64     `json:"size" db:"size"`
	UploadedAt  time.Time `json:"uploaded_at" db:"uploaded_at"`
	URL         string    `json:"url" db:"url"`
	IsDeleted   bool      `json:"is_deleted" db:"is_deleted"`
}

// MediaUploadRequest represents a request to create a media upload record
type MediaUploadRequest struct {
	ObjectKey   string `json:"object_key" validate:"required"`
	FileName    string `json:"file_name" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
	Size        int64  `json:"size" validate:"required,min=1"`
}

// ConfirmUploadRequest represents a request to confirm a successful upload
type ConfirmUploadRequest struct {
	ObjectKey string `json:"object_key" validate:"required"`
}
