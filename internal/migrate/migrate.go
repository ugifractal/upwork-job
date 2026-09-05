package migrate

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const schemaMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version    TEXT PRIMARY KEY,
	applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

type migration struct {
	version string
	upFile  string
	downFile string
}

type Runner struct {
	db  *sql.DB
	dir string
}

func New(db *sql.DB, dir string) *Runner {
	return &Runner{db: db, dir: dir}
}

func (r *Runner) Up() error {
	if err := r.ensureTable(); err != nil {
		return err
	}
	applied, err := r.appliedVersions()
	if err != nil {
		return err
	}
	migs, err := r.loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migs {
		if applied[m.version] {
			continue
		}
		if err := r.applyUp(m); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.version, err)
		}
		fmt.Printf("applied    %s\n", m.version)
	}
	if migrated := len(migs); migrated == len(applied) {
		fmt.Println("no pending migrations")
	}
	return nil
}

func (r *Runner) Down() error {
	if err := r.ensureTable(); err != nil {
		return err
	}
	applied, err := r.appliedVersions()
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		fmt.Println("nothing to roll back")
		return nil
	}

	migs, err := r.loadMigrations()
	if err != nil {
		return err
	}

	var versions []string
	for v := range applied {
		versions = append(versions, v)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))
	target := versions[0]

	for _, m := range migs {
		if m.version == target {
			if err := r.applyDown(m); err != nil {
				return fmt.Errorf("rollback %s failed: %w", m.version, err)
			}
			fmt.Printf("rolled back %s\n", m.version)
			return nil
		}
	}
	return fmt.Errorf("down migration file for version %s not found", target)
}

func (r *Runner) Status() error {
	if err := r.ensureTable(); err != nil {
		return err
	}
	applied, err := r.appliedVersions()
	if err != nil {
		return err
	}
	migs, err := r.loadMigrations()
	if err != nil {
		return err
	}

	fmt.Printf("%-12s  %-10s  %s\n", "VERSION", "STATE", "FILE")
	for _, m := range migs {
		state := "pending"
		if applied[m.version] {
			state = "applied"
		}
		fmt.Printf("%-12s  %-10s  %s\n", m.version, state, filepath.Base(m.downFile))
	}
	return nil
}

func (r *Runner) applyUp(m migration) error {
	sqlBytes, err := os.ReadFile(m.upFile)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range splitStatements(string(sqlBytes)) {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, m.version); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Runner) applyDown(m migration) error {
	sqlBytes, err := os.ReadFile(m.downFile)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range splitStatements(string(sqlBytes)) {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM schema_migrations WHERE version = ?`, m.version); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Runner) ensureTable() error {
	_, err := r.db.Exec(schemaMigrationsTable)
	return err
}

func (r *Runner) appliedVersions() (map[string]bool, error) {
	rows, err := r.db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func (r *Runner) loadMigrations() ([]migration, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}

	byVersion := map[string]*migration{}
	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasSuffix(name, ".up.sql"):
			version := strings.TrimSuffix(name, ".up.sql")
			m, ok := byVersion[version]
			if !ok {
				m = &migration{version: version}
				byVersion[version] = m
			}
			m.upFile = filepath.Join(r.dir, name)
		case strings.HasSuffix(name, ".down.sql"):
			version := strings.TrimSuffix(name, ".down.sql")
			m, ok := byVersion[version]
			if !ok {
				m = &migration{version: version}
				byVersion[version] = m
			}
			m.downFile = filepath.Join(r.dir, name)
		}
	}

	var versions []string
	for v, m := range byVersion {
		if m.upFile == "" || m.downFile == "" {
			return nil, fmt.Errorf("migration %s is missing its .up.sql or .down.sql file", v)
		}
		versions = append(versions, v)
	}
	sort.Strings(versions)

	migs := make([]migration, 0, len(versions))
	for _, v := range versions {
		migs = append(migs, *byVersion[v])
	}
	return migs, nil
}

func splitStatements(sqlText string) []string {
	var stmts []string
	var cur strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(sqlText); i++ {
		ch := sqlText[i]
		prev := byte(0)
		if i > 0 {
			prev = sqlText[i-1]
		}
		switch ch {
		case '\'':
			if !inDouble && (prev != '\\') {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle && (prev != '\\') {
				inDouble = !inDouble
			}
		case ';':
			if !inSingle && !inDouble {
				if s := strings.TrimSpace(cur.String()); s != "" {
					stmts = append(stmts, s)
				}
				cur.Reset()
				continue
			}
		}
		cur.WriteByte(ch)
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}