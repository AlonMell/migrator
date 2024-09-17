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
		Table:   "migrations_history",
		Path:    path,
		Target:  &Migration{major, minor, 2},
		Current: &Migration{major, minor, 0},
	}
}

// TODO: Сделать автодобавление новой записи миграций
// Готово TODO: Сделать парсинг файлов в горутинах (И последовательное выполнение скриптов)
// Готово TODO: Сделать возможность отката к предыдущим версиям (добавить up/down)
// Сделано по другомуTODO: Возможно, ускорить поиск подходящего номера файла

func (m *Migrator) Migrate() error {
	if exists, err := m.tableExists(); err != nil {
		return err
	} else if !exists {
		err = m.ExecBoundsFiles("up")
		return err
	}

	if err := m.setCurrentVersion(); err != nil {
		return err
	}

	if m.Target.FileNumber > m.Current.FileNumber {
		// Apply up migrations
		err := m.ExecBoundsFiles("up")
		if err != nil {
			return err
		}
	} else if m.Target.FileNumber < m.Current.FileNumber {
		// Apply down migrations
		err := m.ExecBoundsFiles("down")
		if err != nil {
			return err
		}
	}
	return nil
}

//func (m *Migrator) Migrate() error {
//	if exists, err := m.tableExists(); err != nil {
//		return err
//	} else if !exists {
//		m.Target.findFileNumber(m.Path)
//		err = m.ExecBoundsFiles()
//		return err
//	}
//
//	if err := m.setCurrentVersion(); err != nil {
//		return err
//	}
//
//	if m.Target.Major < m.Current.Major ||
//		m.Target.Major == m.Current.Major &&
//			m.Target.Minor <= m.Current.Minor {
//
//		return nil
//	}
//
//	m.Target.findFileNumber(m.Path)
//	m.Current.FileNumber++
//
//	err := m.ExecBoundsFiles()
//
//	return err
//}

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

// execBoundsFiles execute all sql scripts in directory where first, last - it's bounds
func (m *Migrator) ExecBoundsFiles(direction string) error {
	const op = "migrator.execBoundsFiles"

	var first, last int
	if direction == "up" {
		first, last = m.Current.FileNumber, m.Target.FileNumber
	} else if direction == "down" {
		first, last = m.Target.FileNumber, m.Current.FileNumber
	}

	files, err := os.ReadDir(m.Path)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	var wg sync.WaitGroup
	scripts := make([]string, last-first)
	errCh := make(chan error, last-first)

	if direction == "up" {
		for i := first; i < last; i++ {
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

				scripts[i/2-first] = string(script)
			}(2*i+1, files[2*i+1].Name())
		}
	} else if direction == "down" {
		for i := first; i < last; i++ {
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

				scripts[last-i/2-1] = string(script) // 4 3 -> 5
			}(2*i, files[2*i].Name())
		}
	}
	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		return <-errCh
	}

	for _, script := range scripts {
		if err := m.execScript(script); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil

}

func readScript(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	script, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(script), nil
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
