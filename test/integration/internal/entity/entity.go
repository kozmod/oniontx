package entity

import (
	"fmt"
)

const (
	ConnectionString = "postgresql://test:passwd@localhost:6432/test?sslmode=disable"
)

var (
	ErrExpected = fmt.Errorf("expected fake error")
)
