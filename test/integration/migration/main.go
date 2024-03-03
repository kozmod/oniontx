package main

import (
	"database/sql"
	"embed"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
)

const (
	statusCmd = "status"
	upCmd     = "up"
	downCmd   = "down"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	log.Printf("migration start")

	cmd := flag.String("cmd", "status", "goose command")
	connString := flag.String("url", "", "connection postgres URL")
	flag.Parse()

	*connString = prepareConnString(*connString)

	connConfig, err := pgx.ParseConfig(*connString)
	if err != nil {
		log.Fatal(errors.WithMessage(err, "parse config"))
	}

	connStr := stdlib.RegisterConnConfig(connConfig)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(errors.WithMessage(err, "open db"))
	}

	if err := retryPing(db, 3, 30*time.Second); err != nil {
		log.Fatal(errors.WithMessage(err, "ping db"))
	}

	goose.SetBaseFS(embedMigrations)

	log.Printf("migration start with command '%s' with connection string '%s'", *cmd, *connString)

	switch {
	case strings.EqualFold(*cmd, statusCmd):
		err = goose.Status(db, "migrations")
	case strings.EqualFold(*cmd, upCmd):
		err = goose.Up(db, "migrations")
	case strings.EqualFold(*cmd, downCmd):
		err = goose.Down(db, "migrations")
	default:
		log.Fatal("cmd is not fit for 'goose'")
	}
	if err != nil {
		log.Fatal(err)
	}

	log.Println("migration finished")
}

//goland:noinspection SpellCheckingInspection
func retryPing(db *sql.DB, retries int, timeout time.Duration) error {
	for i := 1; i <= retries; i++ {
		if err := db.Ping(); err != nil {
			log.Printf("%d ping DB error: %v", i, err)
			time.Sleep(timeout)
			continue
		}
		return nil
	}
	return errors.New("DB unreacheble")
}

func prepareConnString(connString string) string {
	const prefix = "postgresql://"
	if strings.Index(connString, prefix) == 0 {
		return connString
	}
	connString = prefix + connString
	return connString
}
