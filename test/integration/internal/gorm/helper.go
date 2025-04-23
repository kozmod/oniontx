package gorm

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

func ConnectDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(postgres.Open(entity.PostgresConnectionString), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)
	return db
}

func ClearDB(db *gorm.DB) error {
	ex := db.Exec(`TRUNCATE TABLE gorm;`)
	if ex.Error != nil {
		return fmt.Errorf("clear DB: %w", ex.Error)
	}
	return nil
}

func GetTextRecords(db *gorm.DB) ([]Text, error) {
	var texts []Text
	db = db.Find(&texts)
	if err := db.Error; err != nil {
		return nil, fmt.Errorf("get `text` records: %w", err)
	}
	return texts, nil
}
