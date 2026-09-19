package main

import (
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/tecisage/resell-apis/internal/config"
)

func main() {

	cfg := config.MustLoad()

	if len(os.Args) < 2 {
		log.Fatal("usage: make migrate <up | down>")
	}

	m, err := migrate.New("file://migrations", cfg.DB_URL)
	if err != nil {
		log.Fatal(err)
	}

	switch os.Args[1] {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		fmt.Println("database migration success")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		fmt.Println("database rollback success")
	default:
		log.Fatalf("Invalid argument: %s", os.Args[1])
	}
}
