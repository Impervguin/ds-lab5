package domain

import "errors"

var (
	ErrLibraryNotFound   = errors.New("library not found")
	ErrBookNotFound      = errors.New("book not found")
	ErrNoAvailableCopies = errors.New("no available copies")
	ErrInvalidCondition  = errors.New("invalid book condition")
)
