package test

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlonMell/grovelog"
	"github.com/AlonMell/migrator"
	"github.com/fatih/color"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Database connection parameters
	dbHost     = "localhost"
	dbPort     = 15432
	dbUser     = "test"
	dbPassword = "test"
	dbName     = "testdb"

	// Migration configuration
	migrationsTable = "migrations_test"

	// Docker configuration
	imageName     = "migrator-img"
	containerName = "migrator"
)

func connectToDB(t *testing.T) *sql.DB {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName,
	)

	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err, "Failed to open database connection")

	err = db.Ping()
	require.NoError(t, err, "Failed to connect to the database. Make sure PostgreSQL is running in Docker on port 15432")

	return db
}

func prepareDocker(t *testing.T) *sql.DB {
	// Stop and remove any existing container
	exec.Command("docker", "stop", containerName).Run()
	exec.Command("docker", "rm", containerName).Run()

	// Build the Docker image
	cmd := exec.Command("docker", "build", "-t", imageName, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Docker image build error: %v\nOutput: %s", err, output)
	}

	// Run the container
	dbPortMapping := fmt.Sprintf("%d:5432", dbPort)
	cmd = exec.Command("docker", "run", "--name", containerName, "-p", dbPortMapping, "-d", imageName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Docker container start error: %v\nOutput: %s", err, output)
	}

	// Wait for the database to be ready
	var db *sql.DB
	maxAttempts := 10
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName,
	)

	for range maxAttempts {
		time.Sleep(time.Second)
		db, err = sql.Open("postgres", connString)
		if err != nil {
			continue
		}

		if err = db.Ping(); err == nil {
			return db
		}
		db.Close()
	}

	t.Fatalf("Failed to connect to database after %d attempts", maxAttempts)
	return nil
}

// TestMigrator runs a series of tests for the migrator functionality
func TestMigrator(t *testing.T) {
	migrationsDir, err := filepath.Abs("migrations")
	require.NoError(t, err, "Failed to get absolute path to migrations directory")

	// Set up the test environment
	db := prepareDocker(t)
	defer func() {
		db.Close()
		exec.Command("docker", "stop", containerName).Run()
		exec.Command("docker", "rm", containerName).Run()
	}()

	color.NoColor = false
	opts := grovelog.NewOptions(slog.LevelDebug, "", grovelog.Color)
	logger := grovelog.NewLogger(os.Stdout, opts)
	ctx := context.Background()

	t.Run("Initialize migration table", func(t *testing.T) {
		m := migrator.New(db, logger, 1, 0, migrationsTable, migrationsDir)

		err := m.Migrate(ctx)
		assert.NoError(t, err, "Should initialize migration table without errors")

		assertTableExists(t, db, migrationsTable, true)
	})

	t.Run("Apply Up migrations", func(t *testing.T) {
		m := migrator.New(db, logger, 10, 0, migrationsTable, migrationsDir)

		err := m.Migrate(ctx)
		assert.NoError(t, err, "Migrations should execute without errors")

		assertTableExists(t, db, "logs", true)
		assertTableExists(t, db, "users", true)
		assertTableExists(t, db, "posts", true)
		assertTableExists(t, db, "categories", true)
	})

	t.Run("Rollback with Down migrations", func(t *testing.T) {
		m := migrator.New(db, logger, 0, 1, migrationsTable, migrationsDir)

		err := m.Migrate(ctx)
		assert.NoError(t, err, "Migration rollback should execute without errors")

		assertTableExists(t, db, "logs", true)
		assertTableExists(t, db, "users", true)
		assertTableExists(t, db, "posts", false)
		assertTableExists(t, db, "categories", false)
	})

	t.Run("Apply migrations to specific version", func(t *testing.T) {
		m := migrator.New(db, logger, 0, 2, migrationsTable, migrationsDir)

		err := m.Migrate(ctx)
		assert.NoError(t, err, "Migrations should execute without errors")

		assertTableExists(t, db, "logs", true)
		assertTableExists(t, db, "users", true)
		assertTableExists(t, db, "posts", true)
		assertTableExists(t, db, "categories", false)
	})

	t.Run("Handle errors during migration", func(t *testing.T) {
		invalidDir := filepath.Join(t.TempDir(), "invalid_migrations")
		require.NoError(t, os.MkdirAll(invalidDir, 0755))

		invalidSQL := "CREATE TABLE invalid_syntax (id INTEGER, );"
		invalidFile := filepath.Join(invalidDir, "0001.01.01.invalid.up.sql")
		require.NoError(t, os.WriteFile(invalidFile, []byte(invalidSQL), 0644))

		m := migrator.New(db, logger, 1, 1, migrationsTable, invalidDir)

		err := m.Migrate(ctx)
		assert.Error(t, err, "Should return an error when executing invalid migration")
	})
}

func assertTableExists(t *testing.T, db *sql.DB, tableName string, shouldExist bool) {
	var exists bool
	query := "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)"

	err := db.QueryRowContext(context.Background(), query, tableName).Scan(&exists)
	if err != nil {
		t.Fatalf("Error checking table existence %s: %v", tableName, err)
	}

	if shouldExist {
		assert.True(t, exists, "Table %s should exist", tableName)
	} else {
		assert.False(t, exists, "Table %s should not exist", tableName)
	}
}
