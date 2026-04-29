package models

import "time"

type Medicine struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	ImageURL    string    `gorm:"size:500" json:"image_url"`
	Description string    `gorm:"type:text" json:"description"`
	Notes       string    `gorm:"type:text" json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
