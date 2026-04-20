package domain

import "time"

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	DisplayName  string
	Role         string // "user" | "admin"
	Status       string // "active" | "suspended"
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func (u *User) IsAdmin() bool  { return u.Role == "admin" }
func (u *User) IsActive() bool { return u.Status == "active" }
