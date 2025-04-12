package test

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/AlonMell/migrator"
	ver "github.com/AlonMell/migrator/internal/version"
	"github.com/fatih/color"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dbHost          = "localhost"
	dbPort          = 15432
	dbUser          = "test"
	dbPassword      = "test"
	dbName          = "testdb"
	migrationsTable = "migrations_test"
	migrationsDir   = "/home/alonmell/dev/golang/migrator/test/migrations"
	imageName       = "migrator-img"
	containerName   = "migrator"
)

// Подключение к тестовой БД
func connectToDB(t *testing.T) *sql.DB {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName,
	)

	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err, "Не удалось открыть соединение с БД")

	err = db.Ping()
	require.NoError(t, err, "Не удалось подключиться к БД. Убедитесь, что PostgreSQL запущен в Docker на порту 15432")

	return db
}

func prepareDocker(t *testing.T) *sql.DB {
	exec.Command("docker", "stop", containerName).Run()
	exec.Command("docker", "rm", containerName).Run()
	cmd := exec.Command("docker", "build", "-t", imageName, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Ошибка сборки образа Docker: %v\nВывод: %s", err, output)
	}

	dbPortString := strconv.Itoa(dbPort) + ":5432" //+ strconv.Itoa(dbPort)
	cmd = exec.Command("docker", "run", "--name", containerName, "-p", dbPortString, "-d", imageName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Ошибка запуска контейнера Docker: %v\nВывод: %s", err, output)
	}

	var db *sql.DB
	maxAttempts := 10
	for range maxAttempts {
		time.Sleep(time.Second)
		db, err = sql.Open("postgres", fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName,
		))
		if err != nil {
			continue
		}

		if err = db.Ping(); err == nil {
			return db
		}
		db.Close()
	}

	t.Fatalf("Не удалось подключиться к БД после %d попыток", maxAttempts)
	return nil
}

func TestMigrator(t *testing.T) {
	db := prepareDocker(t)
	color.NoColor = false
	logger := migrator.NewDefaultLogger()

	ctx := context.Background()

	t.Run("Инициализация таблицы миграций", func(t *testing.T) {
		m := migrator.New(db, logger, 1, 0, migrationsTable, migrationsDir)

		err := m.InitMigrationTable(ctx)
		assert.NoError(t, err, "Должен инициализировать таблицу миграций без ошибок")

		// Проверяем, что таблица создана
		var exists bool
		err = db.QueryRowContext(ctx,
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)",
			migrationsTable,
		).Scan(&exists)
		assert.NoError(t, err)
		assert.True(t, exists, "Таблица миграций должна существовать")

		// Проверяем начальную версию
		version, err := m.FetchCurrentVersion(ctx)
		assert.NoError(t, err)
		assert.Equal(t, ver.Zero, version, "Начальная версия должна быть нулевой")
	})

	t.Run("Применение Up-миграций", func(t *testing.T) {
		m := migrator.New(db, logger, 10, 0, migrationsTable, migrationsDir)

		err := m.Migrate(ctx)
		assert.NoError(t, err, "Миграции должны выполниться без ошибок")

		tableExists(t, db, "logs", true)
		tableExists(t, db, "users", true)
		tableExists(t, db, "posts", true)
		tableExists(t, db, "categories", true)
	})

	t.Run("Откат с помощью Down-миграций", func(t *testing.T) {
		m := migrator.New(db, logger, 0, 1, migrationsTable, migrationsDir)

		err := m.Migrate(ctx)
		assert.NoError(t, err, "Откат миграций должен выполниться без ошибок")

		tableExists(t, db, "logs", true)
		tableExists(t, db, "users", true)
		tableExists(t, db, "posts", false)
		tableExists(t, db, "categories", false)
	})

	// t.Run("Обработка ошибок при миграции", func(t *testing.T) {
	// 	// Создаем директорию с некорректной миграцией
	// 	invalidDir := filepath.Join(t.TempDir(), "invalid_migrations")
	// 	require.NoError(t, os.MkdirAll(invalidDir, 0755))

	// 	// Создаем файл с некорректным SQL
	// 	invalidSQL := "CREATE TABLE invalid_syntax (id INTEGER, );"
	// 	invalidFile := filepath.Join(invalidDir, "0001.01.01.invalid.up.sql")
	// 	require.NoError(t, os.WriteFile(invalidFile, []byte(invalidSQL), 0644))

	// 	// Создаем мигратор с некорректным SQL
	// 	m := migrator.New(db, logger, 1, 1, migrationsTable, invalidDir)

	// 	// Инициализируем таблицу миграций
	// 	err := m.InitMigrationTable(ctx)
	// 	require.NoError(t, err, "Должен инициализировать таблицу миграций")

	// 	// Пытаемся выполнить некорректную миграцию
	// 	err = m.Migrate(ctx)
	// 	assert.Error(t, err, "Должна возникнуть ошибка при выполнении некорректной миграции")
	// })
}

func tableExists(t *testing.T, db *sql.DB, tableName string, shouldExist bool) {
	var exists bool
	query := "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)"

	err := db.QueryRowContext(context.Background(), query, tableName).Scan(&exists)
	if err != nil {
		t.Fatalf("Ошибка при проверке существования таблицы %s: %v", tableName, err)
	}

	if shouldExist {
		assert.True(t, exists, "Таблица %s должна существовать", tableName)
	} else {
		assert.False(t, exists, "Таблицы %s не должна существовать", tableName)
	}
}
