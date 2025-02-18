package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrInterfaceCast = errors.New("cast error")
)
