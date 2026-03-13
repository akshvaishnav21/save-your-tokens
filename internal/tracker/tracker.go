package tracker

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS syt_log (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp     TEXT    NOT NULL DEFAULT (datetime('now','utc')),
    project_path  TEXT,
    original_cmd  TEXT    NOT NULL,
    syt_cmd       TEXT    NOT NULL,
    input_tokens  INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    savings_pct   REAL    NOT NULL DEFAULT 0.0,
    execution_ms  INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_timestamp ON syt_log(timestamp);

CREATE TABLE IF NOT EXISTS syt_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`

// Tracker manages the SQLite database for token tracking.
type Tracker struct {
	db *sql.DB
}

// Record holds data for a single tracked command execution.
type Record struct {
	OriginalCmd  string
	SytCmd       string
	ProjectPath  string
	InputTokens  int
	OutputTokens int
	ExecutionMs  int64
}

// CommandStat holds aggregated stats for a command type.
type CommandStat struct {
	Command      string
	Runs         int
	TokensSaved  int
	AvgSavingsPct float64
}

// DayStat holds per-day stats.
type DayStat struct {
	Date        string
	Commands    int
	TokensSaved int
}

// Summary holds aggregated summary statistics.
type Summary struct {
	TotalCommands int
	TotalSaved    int
	AvgSavingsPct float64
	ByCommand     []CommandStat
	ByDay         []DayStat
	PeriodDays    int
}

// NewTracker opens (or creates) the SQLite database at dbPath.
func NewTracker(dbPath string) (*Tracker, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("creating tracker dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening tracker db: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	// Create schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	t := &Tracker{db: db}

	// Auto-cleanup if last cleanup was >24h ago
	go func() {
		_ = t.maybeCleanup()
	}()

	return t, nil
}

func (t *Tracker) maybeCleanup() error {
	var lastCleanup string
	row := t.db.QueryRow("SELECT value FROM syt_meta WHERE key='last_cleanup'")
	if err := row.Scan(&lastCleanup); err == nil {
		ts, err := time.Parse(time.RFC3339, lastCleanup)
		if err == nil && time.Since(ts) < 24*time.Hour {
			return nil
		}
	}
	if err := t.Cleanup(90); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := t.db.Exec("INSERT OR REPLACE INTO syt_meta (key, value) VALUES ('last_cleanup', ?)", now)
	return err
}

// Track inserts a record into the database.
func (t *Tracker) Track(r Record) error {
	var savingsPct float64
	if r.InputTokens > 0 {
		savingsPct = 100.0 - float64(r.OutputTokens)/float64(r.InputTokens)*100.0
		if savingsPct < 0 {
			savingsPct = 0
		}
	}

	_, err := t.db.Exec(`
		INSERT INTO syt_log (project_path, original_cmd, syt_cmd, input_tokens, output_tokens, savings_pct, execution_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ProjectPath, r.OriginalCmd, r.SytCmd,
		r.InputTokens, r.OutputTokens, savingsPct, r.ExecutionMs,
	)
	if err != nil {
		return fmt.Errorf("inserting track record: %w", err)
	}
	return nil
}

// GetSummary returns aggregated statistics since the given time.
func (t *Tracker) GetSummary(since time.Time) (Summary, error) {
	sinceStr := since.UTC().Format("2006-01-02T15:04:05")

	var s Summary
	s.PeriodDays = int(time.Since(since).Hours() / 24)

	// Total stats
	row := t.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(input_tokens - output_tokens), 0), COALESCE(AVG(savings_pct), 0)
		FROM syt_log WHERE timestamp >= ?`, sinceStr)
	if err := row.Scan(&s.TotalCommands, &s.TotalSaved, &s.AvgSavingsPct); err != nil {
		return s, fmt.Errorf("querying summary: %w", err)
	}

	// By command
	rows, err := t.db.Query(`
		SELECT syt_cmd,
		       COUNT(*) as runs,
		       COALESCE(SUM(input_tokens - output_tokens), 0) as tokens_saved,
		       COALESCE(AVG(savings_pct), 0) as avg_pct
		FROM syt_log
		WHERE timestamp >= ?
		GROUP BY syt_cmd
		ORDER BY tokens_saved DESC
		LIMIT 20`, sinceStr)
	if err != nil {
		return s, fmt.Errorf("querying by command: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cs CommandStat
		if err := rows.Scan(&cs.Command, &cs.Runs, &cs.TokensSaved, &cs.AvgSavingsPct); err != nil {
			continue
		}
		s.ByCommand = append(s.ByCommand, cs)
	}

	// By day
	dayRows, err := t.db.Query(`
		SELECT date(timestamp) as day, COUNT(*), COALESCE(SUM(input_tokens - output_tokens), 0)
		FROM syt_log
		WHERE timestamp >= ?
		GROUP BY day
		ORDER BY day DESC
		LIMIT 30`, sinceStr)
	if err != nil {
		return s, fmt.Errorf("querying by day: %w", err)
	}
	defer dayRows.Close()
	for dayRows.Next() {
		var ds DayStat
		if err := dayRows.Scan(&ds.Date, &ds.Commands, &ds.TokensSaved); err != nil {
			continue
		}
		s.ByDay = append(s.ByDay, ds)
	}

	return s, nil
}

// GetHistory returns the most recent tracked records.
func (t *Tracker) GetHistory(limit int) ([]Record, error) {
	rows, err := t.db.Query(`
		SELECT original_cmd, syt_cmd, COALESCE(project_path,''), input_tokens, output_tokens, execution_ms
		FROM syt_log
		ORDER BY id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("querying history: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.OriginalCmd, &r.SytCmd, &r.ProjectPath, &r.InputTokens, &r.OutputTokens, &r.ExecutionMs); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records, nil
}

// GetDailyStats returns per-day statistics for the last N days.
func (t *Tracker) GetDailyStats(days int) ([]DayStat, error) {
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02T15:04:05")
	rows, err := t.db.Query(`
		SELECT date(timestamp) as day, COUNT(*), COALESCE(SUM(input_tokens - output_tokens), 0)
		FROM syt_log
		WHERE timestamp >= ?
		GROUP BY day
		ORDER BY day DESC`, since)
	if err != nil {
		return nil, fmt.Errorf("querying daily stats: %w", err)
	}
	defer rows.Close()

	var stats []DayStat
	for rows.Next() {
		var ds DayStat
		if err := rows.Scan(&ds.Date, &ds.Commands, &ds.TokensSaved); err != nil {
			continue
		}
		stats = append(stats, ds)
	}
	return stats, nil
}

// Cleanup removes records older than retentionDays.
func (t *Tracker) Cleanup(retentionDays int) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays).Format("2006-01-02T15:04:05")
	_, err := t.db.Exec("DELETE FROM syt_log WHERE timestamp < ?", cutoff)
	if err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (t *Tracker) Close() error {
	return t.db.Close()
}
