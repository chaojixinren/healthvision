package models

import "time"

type Reminder struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	MedicineID uint      `gorm:"not null;index" json:"medicine_id"`
	Time       string    `gorm:"size:5;not null" json:"time"`
	Enabled    bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
