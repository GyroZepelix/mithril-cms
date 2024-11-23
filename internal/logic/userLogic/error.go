package userLogic

import "errors"

var (
	ErrNotFound       = errors.New("User not found!")
	ErrInternalServer = errors.New("Internal server error!")
)
