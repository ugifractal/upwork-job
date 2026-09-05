package store

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type Job struct {
	ID          int64  `json:"id"`
	UpworkJobID string `json:"upwork_job_id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Link        string `json:"link"`
	CreatedAt   string `json:"created_at"`
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	for _, pragma := range []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

// CreateJob inserts a job, skipping it when upwork_job_id already exists.
// Returns the job (with its DB row) and created=true when newly inserted,
// or created=false when the job was a duplicate.
func (s *Store) CreateJob(job Job) (Job, bool, error) {
	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO upwork_jobs (upwork_job_id, title, summary, link) VALUES (?, ?, ?, ?)`,
		job.UpworkJobID, job.Title, job.Summary, job.Link,
	)
	if err != nil {
		return Job{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return Job{}, false, err
	}
	if affected == 0 {
		return Job{}, false, nil
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Job{}, false, err
	}
	inserted, err := s.GetJob(id)
	if err != nil {
		return Job{}, false, err
	}
	return inserted, true, nil
}

func (s *Store) GetJob(id int64) (Job, error) {
	row := s.db.QueryRow(
		`SELECT id, upwork_job_id, title, summary, link, created_at FROM upwork_jobs WHERE id = ?`, id)
	return scanJob(row)
}

func (s *Store) ListJobs() ([]Job, error) {
	rows, err := s.db.Query(`SELECT id, upwork_job_id, title, summary, link, created_at FROM upwork_jobs ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.UpworkJobID, &j.Title, &j.Summary, &j.Link, &j.CreatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(row rowScanner) (Job, error) {
	var j Job
	err := row.Scan(&j.ID, &j.UpworkJobID, &j.Title, &j.Summary, &j.Link, &j.CreatedAt)
	return j, err
}