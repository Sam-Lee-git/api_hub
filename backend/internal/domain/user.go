package domain

import "time"

type User struct {
	ID           int64      `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	DisplayName  string     `json:"display_name"`
	Role         string     `json:"role"`   // "user" | "admin"
	Status       string     `json:"status"` // "active" | "suspended"
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"-"`
}

func (u *User) IsAdmin() bool  { return u.Role == "admin" }
func (u *User) IsActive() bool { return u.Status == "active" }
