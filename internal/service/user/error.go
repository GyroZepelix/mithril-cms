package user

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound       = errors.New("User not found!")
	ErrInternalServer = errors.New("Internal server error!")
)

type ErrUniqueValueViolation struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

func (e *ErrUniqueValueViolation) Error() string {
	return fmt.Sprintf("Duplicate value for %s: %v", e.Field, e.Value)
}
