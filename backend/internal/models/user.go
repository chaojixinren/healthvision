package models

import "time"

const RoleUser = "user"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"size:255;not null;uniqueIndex" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Name         string    `gorm:"size:100" json:"name"`
	Role         string    `gorm:"size:32;not null;default:user" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
