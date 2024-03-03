package pgx

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

func ConnectDB(ctx context.Context, t *testing.T) *pgx.Conn {
	conn, err := pgx.Connect(ctx, entity.ConnectionString)
	assert.NoError(t, err)

	err = conn.Ping(ctx)
	assert.NoError(t, err)
	return conn
}

func ClearDB(ctx context.Context, db *pgx.Conn) error {
	_, err := db.Exec(ctx, `TRUNCATE TABLE pgx;`)
	if err != nil {
		return fmt.Errorf("clear DB: %w", err)
	}
	return nil
}

func GetTextRecords(ctx context.Context, db *pgx.Conn) ([]string, error) {
	row, err := db.Query(ctx, "SELECT val FROM pgx;")
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
