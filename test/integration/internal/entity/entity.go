package entity

import (
	"fmt"
)

// connection credentials (oniontx/test/.env)
const (
	PostgresConnectionString = "postgresql://test:passwd@localhost:6432/test?sslmode=disable"

	RedisHostPort     = "localhost:6379"
	RedisUser         = "test"
	RedisUserPassword = "passwd"
)

var (
	ErrExpected = fmt.Errorf("expected fake error")
)
