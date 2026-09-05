-- 0001_create_upwork_jobs.up.sql
CREATE TABLE IF NOT EXISTS upwork_jobs (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    upwork_job_id TEXT NOT NULL UNIQUE,
    title         TEXT NOT NULL,
    summary       TEXT NOT NULL DEFAULT '',
    link          TEXT NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);