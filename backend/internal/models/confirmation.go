package models

import "time"

type Confirmation struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ReminderID    uint       `gorm:"not null;uniqueIndex:idx_reminder_date" json:"reminder_id"`
	MedicineID    uint       `gorm:"not null" json:"medicine_id"`
	ScheduledDate string     `gorm:"size:10;not null;uniqueIndex:idx_reminder_date" json:"scheduled_date"`
	ScheduledTime string     `gorm:"size:5;not null" json:"scheduled_time"`
	UserID        uint       `gorm:"not null;index" json:"user_id"`
	ConfirmedAt   *time.Time `json:"confirmed_at"`
	ConfirmedBy   uint       `gorm:"default:0" json:"confirmed_by"`
	CreatedAt     time.Time  `json:"created_at"`
}
