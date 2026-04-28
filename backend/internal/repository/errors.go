package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrDuplicateKey = errors.New("duplicate key")
)

func wrapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrUserNotFound
	}
	return err
}

func isUniqueConstraintError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}
