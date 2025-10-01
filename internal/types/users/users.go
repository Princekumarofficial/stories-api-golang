package users

type SignUpRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type SignInRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	CreatedAt string `json:"created_at"`
}

type UserStats struct {
	Posted         int                    `json:"posted"`
	Views          int                    `json:"views"`
	UniqueViewers  int                    `json:"unique_viewers"`
	ReactionCounts map[string]int         `json:"reaction_counts"`
}
