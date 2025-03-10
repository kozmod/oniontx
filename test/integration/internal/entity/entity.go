package entity

import (
	"fmt"
)

// connection credentials.
const (
	PostgresConnectionString = "postgresql://test:passwd@localhost:6432/test?sslmode=disable"

	MongoConnectionString = "mongodb://localhost:27017"

	RedisAddr = "localhost:6379"
)

var (
	// ErrExpected - error for tests.
	ErrExpected = fmt.Errorf("expected fake error")
)
