package models

import "time"

const (
	BindingStatusPending  = "pending"
	BindingStatusAccepted = "accepted"
	BindingStatusRejected = "rejected"
)

type Binding struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ElderID   uint      `gorm:"not null;index" json:"elder_id"`
	ChildID   uint      `gorm:"not null;index" json:"child_id"`
	Status    string    `gorm:"size:16;not null;default:pending" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Elder *User `gorm:"foreignKey:ElderID" json:"elder,omitempty"`
	Child *User `gorm:"foreignKey:ChildID" json:"child,omitempty"`
}
