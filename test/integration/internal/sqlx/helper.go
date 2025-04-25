package sqlx

import (
	"context"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

func ConnectDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("postgres", entity.PostgresConnectionString)
	assert.NoError(t, err)

	err = db.Ping()
	assert.NoError(t, err)
	return db
}

func ClearDB(ctx context.Context, db *sqlx.DB) error {
	_, err := db.ExecContext(ctx, `TRUNCATE TABLE sqlx;`)
	if err != nil {
		return fmt.Errorf("clear DB: %w", err)
	}
	return nil
}

func GetTextRecords(ctx context.Context, db *sqlx.DB) ([]string, error) {
	row, err := db.QueryContext(ctx, "SELECT val FROM sqlx;")
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
