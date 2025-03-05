package entity

import (
	"fmt"
)

// connection credentials (oniontx/test/.env)
const (
	PostgresConnectionString = "postgresql://test:passwd@localhost:6432/test?sslmode=disable"

	RedisAddr = "localhost:6379"
)

var (
	ErrExpected = fmt.Errorf("expected fake error")
)
