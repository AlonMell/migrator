# Migrator

[![Go Reference](https://pkg.go.dev/badge/github.com/AlonMell/migrator.svg)](https://pkg.go.dev/github.com/AlonMell/migrator)
[![Go Version](https://img.shields.io/github/go-mod/go-version/AlonMell/migrator)](https://github.com/AlonMell/migrator)
[![License](https://img.shields.io/github/license/AlonMell/migrator)](https://github.com/AlonMell/migrator/blob/main/LICENSE)

A lightweight, SQL-based database migration tool for Go applications.

## Features

- Simple and efficient SQL migration management
- Supports both UP and DOWN migrations
- Version-based migration tracking
- Transaction-based migrations for safety
- Detailed logging with customizable log levels
- Automatic migration table creation
- Compatible with PostgreSQL (extensible for other databases)

## Installation

```bash
go get github.com/AlonMell/migrator
```

## Usage

### Basic Usage

```go
package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/AlonMell/migrator"
	_ "github.com/lib/pq"
)

func main() {
	// Connect to the database
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/dbname?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create a new migrator
	// Parameters: database connection, logger, major version, minor version,
	// migrations table name, migrations directory path
	m := migrator.New(db, nil, 1, 0, "migrations", "./migrations")

	// Run migrations
	ctx := context.Background()
	if err := m.Migrate(ctx); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
}
```

### With Custom Logger

```go
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"github.com/AlonMell/grovelog"
	"github.com/AlonMell/migrator"
	_ "github.com/lib/pq"
)

func main() {
	// Setup logger
	opts := grovelog.NewOptions(slog.LevelDebug, "", grovelog.Color)
	logger := grovelog.NewLogger(os.Stdout, opts)

	// Connect to database
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/dbname?sslmode=disable")
	if err != nil {
		logger.Error("Failed to connect to database", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create migrator with custom logger
	m := migrator.New(db, logger, 1, 0, "migrations", "./migrations")

	// Run migrations
	ctx := context.Background()
	if err := m.Migrate(ctx); err != nil {
		logger.Error("Migration failed", err)
		os.Exit(1)
	}
}
```

## Migration File Format

Migration files must follow this naming convention:

```
nnnn.mm.nn[.comment].(up|down).sql
```

Where:
- `nnnn` - File number (4 digits)
- `mm` - Major version (2 digits)
- `nn` - Minor version (2 digits)
- `[.comment]` - Optional comment/description
- `(up|down)` - Migration direction (up = apply, down = rollback)

Examples:
- `0001.00.00.baseline.up.sql` - Initial schema creation
- `0001.00.00.baseline.down.sql` - Rollback for initial schema
- `0002.00.01.create_users.up.sql` - Create users table
- `0002.00.01.create_users.down.sql` - Drop users table

## Version Control

The migrator uses a version system with major and minor components:
- Major version (mm) - For significant schema changes
- Minor version (nn) - For smaller, incremental changes

When running the migrator, specify the target version (major.minor) you want to migrate to:
- To upgrade: specify a higher version than current
- To downgrade: specify a lower version than current

## Development

### Requirements

- Go 1.24+
- PostgreSQL
- Docker (for integration tests)

### Testing

Run integration tests with Docker:
```bash
make test
```

### Building

```bash
make build
```

### Other Commands

```bash
make help  # Show available commands
```

## License

MIT License - See LICENSE file for details.