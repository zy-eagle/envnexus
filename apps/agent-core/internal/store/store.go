package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db      *sql.DB
	dataDir string
}

func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "agent.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	s := &Store{db: db, dataDir: dataDir}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	slog.Info("[store] SQLite initialized", "path", dbPath)
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'created',
			intent TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT,
			event_type TEXT NOT NULL,
			payload TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS config_cache (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS governance_baselines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scope TEXT NOT NULL,
			baseline_json TEXT NOT NULL,
			captured_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS governance_drifts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			baseline_id INTEGER NOT NULL,
			drift_json TEXT NOT NULL,
			severity TEXT NOT NULL DEFAULT 'info',
			detected_at TEXT NOT NULL
		)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}
	return nil
}

func (s *Store) SaveSession(id, tenantID, deviceID, status, intent string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO sessions (id, tenant_id, device_id, status, intent, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM sessions WHERE id = ?), ?), ?)`,
		id, tenantID, deviceID, status, intent, id, now, now,
	)
	return err
}

func (s *Store) SaveAuditEvent(sessionID, eventType, payload string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO audit_events (session_id, event_type, payload, created_at) VALUES (?, ?, ?, ?)`,
		sessionID, eventType, payload, now,
	)
	return err
}

func (s *Store) SetConfig(key, value string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO config_cache (key, value, updated_at) VALUES (?, ?, ?)`,
		key, value, now,
	)
	return err
}

func (s *Store) GetConfig(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM config_cache WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

type GovernanceBaseline struct {
	ID           int64
	Scope        string
	BaselineJSON string
	CapturedAt   string
}

func (s *Store) SaveBaseline(scope, baselineJSON string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		`INSERT INTO governance_baselines (scope, baseline_json, captured_at) VALUES (?, ?, ?)`,
		scope, baselineJSON, now,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) GetLatestBaseline(scope string) (*GovernanceBaseline, error) {
	var b GovernanceBaseline
	err := s.db.QueryRow(
		`SELECT id, scope, baseline_json, captured_at FROM governance_baselines WHERE scope = ? ORDER BY captured_at DESC LIMIT 1`,
		scope,
	).Scan(&b.ID, &b.Scope, &b.BaselineJSON, &b.CapturedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) SaveDrift(baselineID int64, driftJSON, severity string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO governance_drifts (baseline_id, drift_json, severity, detected_at) VALUES (?, ?, ?, ?)`,
		baselineID, driftJSON, severity, now,
	)
	return err
}

type RecentSession struct {
	ID        string
	TenantID  string
	DeviceID  string
	Status    string
	Intent    string
	CreatedAt string
}

// Vacuum runs SQLite VACUUM to reclaim disk space and optimize the database.
func (s *Store) Vacuum() error {
	_, err := s.db.Exec("VACUUM")
	if err != nil {
		slog.Warn("[store] VACUUM failed", "error", err)
	}
	return err
}

func (s *Store) ListRecentSessions(limit int) ([]RecentSession, error) {
	rows, err := s.db.Query(
		`SELECT id, tenant_id, device_id, status, COALESCE(intent, ''), created_at FROM sessions ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []RecentSession
	for rows.Next() {
		var s RecentSession
		if err := rows.Scan(&s.ID, &s.TenantID, &s.DeviceID, &s.Status, &s.Intent, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}
