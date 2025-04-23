package stdlib

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

func ConnectDB(t *testing.T) *sql.DB {
	t.Helper()

	connConfig, err := pgx.ParseConfig(entity.PostgresConnectionString)
	assert.NoError(t, err)

	connStr := stdlib.RegisterConnConfig(connConfig)
	db, err := sql.Open("pgx", connStr)
	assert.NoError(t, err)

	err = db.Ping()
	assert.NoError(t, err)

	return db
}

func ClearDB(db *sql.DB) error {
	_, err := db.Exec("TRUNCATE TABLE stdlib;")
	if err != nil {
		return fmt.Errorf("clear DB: %w", err)
	}
	return nil
}

func GetTextRecords(db *sql.DB) ([]string, error) {
	row, err := db.Query("SELECT val FROM stdlib;")
	if err != nil {
		return nil, fmt.Errorf("get `text` records: %w", err)
	}

	var texts []string
	for row.Next() {
		var text string
		err = row.Scan(&text)
		if err != nil {
			return nil, fmt.Errorf("scan `text` records: %w", err)
		}
		texts = append(texts, text)
	}
	return texts, nil
}
