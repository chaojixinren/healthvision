package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound = errors.New("用户不存在")
	ErrNotFound     = errors.New("记录不存在")
	ErrDuplicateKey = errors.New("数据重复")
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
