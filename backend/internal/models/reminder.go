package models

import "time"

const (
	RepeatTypeDaily    = "daily"
	RepeatTypeInterval = "interval"
	RepeatTypeWeekly   = "weekly"
)

type Reminder struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null;index" json:"user_id"`
	MedicineID   uint      `gorm:"not null;index" json:"medicine_id"`
	Time         string    `gorm:"size:5;not null" json:"time"`
	RepeatType   string    `gorm:"size:10;not null;default:daily" json:"repeat_type"`
	IntervalDays int       `gorm:"not null;default:1" json:"interval_days"`
	Weekdays     string    `gorm:"size:20" json:"weekdays"`
	Enabled      bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedBy    uint      `gorm:"not null;default:0" json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
