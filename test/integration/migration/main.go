package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	statusCmd = "status"
	upCmd     = "up"
	downCmd   = "down"

	migrationsDir = "migrations"
	driverName    = "pgx"

	execTimeout = 30 * time.Second
)

var (
	commands = []string{
		statusCmd, upCmd, downCmd,
	}
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	log.Printf("migration start")

	var (
		cmd       = flag.String("cmd", statusCmd, "goose command")
		urlString = flag.String("url", "", "connection postgres URL")
	)
	flag.Parse()

	var (
		connString = prepareConnString(*urlString)
		command    = strings.ToLower(strings.TrimSpace(*cmd))
	)

	connConfig, err := pgx.ParseConfig(connString)
	if err != nil {
		log.Fatal(fmt.Errorf("parse config: %w", err))
	}

	connStr := stdlib.RegisterConnConfig(connConfig)
	db, err := sql.Open(driverName, connStr)
	if err != nil {
		log.Fatal(fmt.Errorf("open [%s]: %w", connString, err))
	}

	if err = retryPing(db, 3, 30*time.Second); err != nil {
		log.Fatal(fmt.Errorf("ping [%s]: %w", connString, err))
	}

	goose.SetBaseFS(embedMigrations)

	log.Printf("migration start with command [%s] - connection string [%s]", command, connString)

	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()

	switch command {
	case statusCmd:
		err = goose.StatusContext(ctx, db, migrationsDir)
	case upCmd:
		err = goose.UpContext(ctx, db, migrationsDir)
	case downCmd:
		err = goose.DownContext(ctx, db, migrationsDir)
	default:
		log.Printf(
			"cmd [%s] is not match with [%s]",
			command,
			strings.Join(commands, ","),
		)
		return
	}
	if err != nil {
		log.Printf("execute migration failed: %v", err)
		return
	}

	log.Println("migration finished")
}

func retryPing(db *sql.DB, tries int, timeout time.Duration) error {
	for i := 1; i <= tries; i++ {
		if err := db.Ping(); err != nil {
			log.Printf("[%d] ping failed: %v", i, err)
			time.Sleep(timeout)
			continue
		}
		return nil
	}
	return fmt.Errorf("database is unreachable")
}

func prepareConnString(connString string) string {
	const prefix = "postgresql://"

	connString = strings.TrimSpace(connString)
	if strings.Index(connString, prefix) == 0 {
		return connString
	}
	connString = fmt.Sprintf("%s%s", prefix, connString)
	return connString
}
