package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/AlonMell/migrator"
	_ "github.com/lib/pq"
)

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

func MustConnect(cfg DBConfig) *sql.DB {
	const op = "migrator.MustConnect"

	sourceInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database)

	db, err := sql.Open("postgres", sourceInfo)
	if err != nil {
		panic(fmt.Errorf("%s: %w", op, err))
	} else if err = db.Ping(); err != nil {
		panic(fmt.Errorf("%s: %w", op, err))
	}

	return db
}

func main() {
	cfg := DBConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "alonmell",
		Password: "qwerty",
		Database: "ProviderHub",
	}
	db := MustConnect(cfg)

	var path, table string
	var major, minor int
	flag.StringVar(&path, "path", path, "path to migrations")
	flag.StringVar(&table, "table", table, "table name")
	flag.IntVar(&major, "major", major, "major version")
	flag.IntVar(&minor, "minor", minor, "minor version")
	flag.Parse()

	m := migrator.New(db, path, table, major, minor)
	if err := m.Migrate(); err != nil {
		panic(err)
	}

	fmt.Println("Migration completed successfully")
}
