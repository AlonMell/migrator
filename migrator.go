package migrator

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type Migrator struct {
	db              *sql.DB
	Target, Current *Migration
	Path            string
	Table           string
}

type Migration struct {
	Major, Minor int
	FileNumber   int
}

func New(db *sql.DB, path, table string, major, minor int) *Migrator {
	return &Migrator{
		db:      db,
		Table:   table,
		Path:    path,
		Target:  &Migration{major, minor, 0},
		Current: &Migration{},
	}
}

// TODO: Сделать автодобавление новой записи миграций
// TODO: Сделать парсинг файлов в горутинах (И последовательное выполнение скриптов)
// TODO: Сделать возможность отката к предыдущим версиям (добавить up/down)
// TODO: Возможно, ускорить поиск подходящего номера файла

func (m *Migrator) Migrate() error {
	if exists, err := m.tableExists(); err != nil {
		return err
	} else if !exists {
		m.Target.findFileNumber(m.Path)
		err = m.execBoundsFiles()
		return err
	}

	if err := m.setCurrentVersion(); err != nil {
		return err
	}

	if m.Target.Major < m.Current.Major ||
		m.Target.Major == m.Current.Major &&
			m.Target.Minor <= m.Current.Minor {

		return nil
	}

	m.Target.findFileNumber(m.Path)
	m.Current.FileNumber++

	err := m.execBoundsFiles()

	return err
}

func (m *Migrator) setCurrentVersion() error {
	const op = "migrator.setCurrentVersion"

	query := fmt.Sprintf(`
    SELECT major_version, minor_version, file_number
    FROM %s 
    ORDER BY date_applied DESC 
    LIMIT 1;`, m.Table)

	stmt, err := m.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	err = stmt.QueryRow().Scan(&m.Current.Major, &m.Current.Minor, &m.Current.FileNumber)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// execBoundsFiles execute all sql scripts in directory where first, second - it's bounds
func (m *Migrator) execBoundsFiles() error {
	const op = "migrator.execBoundsFiles"

	first, second := m.Current.FileNumber, m.Target.FileNumber

	files, err := os.ReadDir(m.Path)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	scripts := make(map[int]string)
	errCh := make(chan error, second-first+1)
	var wg sync.WaitGroup

	for i := first; i <= second; i++ {
		wg.Add(1)
		go func(i int, fileName string) {
			defer wg.Done()
			path := filepath.Join(m.Path, fileName)
			file, err := os.Open(path)
			if err != nil {
				errCh <- fmt.Errorf("%s: %w", op, err)
				return
			}
			defer file.Close()

			script, err := io.ReadAll(file)
			if err != nil {
				errCh <- fmt.Errorf("%s: %w", op, err)
				return
			}

			scripts[i] = string(script)
		}(i, files[i].Name())
	}

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		return <-errCh
	}

	for i := first; i <= second; i++ {
		if err := m.execScript(scripts[i]); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

// exec execute sql script
func (m *Migrator) execScript(script string) error {
	const op = "migrator.execScript"
	_, err := m.db.Exec(script)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (m *Migrator) tableExists() (bool, error) {
	const op = "migrator.tableExists"

	query := `
    SELECT EXISTS (
        SELECT FROM information_schema.tables 
        WHERE table_schema = 'public' 
        AND table_name = $1
    );`

	stmt, err := m.db.Prepare(query)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var exists bool
	err = stmt.QueryRow(m.Table).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

// findFileNumber find last version file number
// in: 00 00
// files: 0001.00.00 0002.00.00
// out: 2
func (m *Migration) findFileNumber(path string) {
	var first int

	for i := 1; i < 10000; i++ {
		fileName := fmt.Sprintf("%04d.%02d.%02d", i, m.Major, m.Minor)
		path = filepath.Join(path, fileName)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}

		first = i
	}

	m.FileNumber = first
}
