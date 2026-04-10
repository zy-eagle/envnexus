package watchlist

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS watch_items (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			source TEXT NOT NULL DEFAULT 'user',
			tool_name TEXT NOT NULL,
			tool_params TEXT,
			condition_json TEXT NOT NULL,
			interval_ms INTEGER NOT NULL DEFAULT 300000,
			enabled INTEGER NOT NULL DEFAULT 1,
			last_check_at TEXT,
			last_status TEXT,
			consecutive_fail INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS watch_alerts (
			id TEXT PRIMARY KEY,
			watch_item_id TEXT NOT NULL,
			item_name TEXT NOT NULL,
			severity TEXT NOT NULL DEFAULT 'info',
			message TEXT NOT NULL,
			value TEXT,
			created_at TEXT NOT NULL,
			resolved INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_watch_items_source ON watch_items(source)`,
		`CREATE INDEX IF NOT EXISTS idx_watch_items_enabled ON watch_items(enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_watch_alerts_item ON watch_alerts(watch_item_id)`,
		`CREATE INDEX IF NOT EXISTS idx_watch_alerts_resolved ON watch_alerts(resolved)`,
	}
	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SaveWatchItem(item *WatchItem) error {
	condJSON, err := json.Marshal(item.Condition)
	if err != nil {
		return err
	}
	paramsJSON, err := json.Marshal(item.ToolParams)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO watch_items
		 (id, name, description, source, tool_name, tool_params, condition_json, interval_ms, enabled, last_check_at, last_status, consecutive_fail, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM watch_items WHERE id = ?), ?))`,
		item.ID, item.Name, item.Description, string(item.Source),
		item.ToolName, string(paramsJSON), string(condJSON),
		item.Interval.Milliseconds(), boolToInt(item.Enabled),
		nullableTime(item.LastCheckAt), item.LastStatus, item.ConsecutiveFail,
		item.ID, now,
	)
	return err
}

func (s *Store) GetWatchItem(id string) (*WatchItem, error) {
	var (
		item        WatchItem
		source      string
		paramsJSON  string
		condJSON    string
		intervalMs  int64
		enabled     int
		lastCheckAt sql.NullString
		createdAt   string
	)
	err := s.db.QueryRow(
		`SELECT id, name, description, source, tool_name, tool_params, condition_json,
		        interval_ms, enabled, last_check_at, last_status, consecutive_fail, created_at
		 FROM watch_items WHERE id = ?`, id,
	).Scan(&item.ID, &item.Name, &item.Description, &source,
		&item.ToolName, &paramsJSON, &condJSON,
		&intervalMs, &enabled, &lastCheckAt, &item.LastStatus, &item.ConsecutiveFail, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.Source = WatchItemSource(source)
	item.Interval = time.Duration(intervalMs) * time.Millisecond
	item.Enabled = enabled == 1
	if lastCheckAt.Valid {
		item.LastCheckAt, _ = time.Parse(time.RFC3339, lastCheckAt.String)
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if err := json.Unmarshal([]byte(condJSON), &item.Condition); err != nil {
		return nil, err
	}
	if paramsJSON != "" {
		_ = json.Unmarshal([]byte(paramsJSON), &item.ToolParams)
	}
	return &item, nil
}

func (s *Store) ListWatchItems(source string) ([]*WatchItem, error) {
	query := `SELECT id, name, description, source, tool_name, tool_params, condition_json,
	                 interval_ms, enabled, last_check_at, last_status, consecutive_fail, created_at
	          FROM watch_items`
	var args []interface{}
	if source != "" {
		query += ` WHERE source = ?`
		args = append(args, source)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*WatchItem
	for rows.Next() {
		var (
			item        WatchItem
			src         string
			paramsJSON  string
			condJSON    string
			intervalMs  int64
			enabled     int
			lastCheckAt sql.NullString
			createdAt   string
		)
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &src,
			&item.ToolName, &paramsJSON, &condJSON,
			&intervalMs, &enabled, &lastCheckAt, &item.LastStatus, &item.ConsecutiveFail, &createdAt); err != nil {
			return nil, err
		}
		item.Source = WatchItemSource(src)
		item.Interval = time.Duration(intervalMs) * time.Millisecond
		item.Enabled = enabled == 1
		if lastCheckAt.Valid {
			item.LastCheckAt, _ = time.Parse(time.RFC3339, lastCheckAt.String)
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if err := json.Unmarshal([]byte(condJSON), &item.Condition); err != nil {
			return nil, err
		}
		if paramsJSON != "" {
			_ = json.Unmarshal([]byte(paramsJSON), &item.ToolParams)
		}
		items = append(items, &item)
	}
	return items, nil
}

func (s *Store) ListEnabledWatchItems() ([]*WatchItem, error) {
	rows, err := s.db.Query(
		`SELECT id, name, description, source, tool_name, tool_params, condition_json,
		        interval_ms, enabled, last_check_at, last_status, consecutive_fail, created_at
		 FROM watch_items WHERE enabled = 1 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*WatchItem
	for rows.Next() {
		var (
			item        WatchItem
			src         string
			paramsJSON  string
			condJSON    string
			intervalMs  int64
			enabled     int
			lastCheckAt sql.NullString
			createdAt   string
		)
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &src,
			&item.ToolName, &paramsJSON, &condJSON,
			&intervalMs, &enabled, &lastCheckAt, &item.LastStatus, &item.ConsecutiveFail, &createdAt); err != nil {
			return nil, err
		}
		item.Source = WatchItemSource(src)
		item.Interval = time.Duration(intervalMs) * time.Millisecond
		item.Enabled = enabled == 1
		if lastCheckAt.Valid {
			item.LastCheckAt, _ = time.Parse(time.RFC3339, lastCheckAt.String)
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if err := json.Unmarshal([]byte(condJSON), &item.Condition); err != nil {
			return nil, err
		}
		if paramsJSON != "" {
			_ = json.Unmarshal([]byte(paramsJSON), &item.ToolParams)
		}
		items = append(items, &item)
	}
	return items, nil
}

func (s *Store) DeleteWatchItem(id string) error {
	_, err := s.db.Exec(`DELETE FROM watch_items WHERE id = ?`, id)
	return err
}

func (s *Store) UpdateWatchItemStatus(id, status string, consecutiveFail int) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`UPDATE watch_items SET last_check_at = ?, last_status = ?, consecutive_fail = ? WHERE id = ?`,
		now, status, consecutiveFail, id,
	)
	return err
}

func (s *Store) SetWatchItemEnabled(id string, enabled bool) error {
	_, err := s.db.Exec(
		`UPDATE watch_items SET enabled = ? WHERE id = ?`,
		boolToInt(enabled), id,
	)
	return err
}

func (s *Store) SaveAlert(alert *WatchAlert) error {
	now := time.Now().UTC().Format(time.RFC3339)
	valueJSON := ""
	if alert.Value != nil {
		b, _ := json.Marshal(alert.Value)
		valueJSON = string(b)
	}
	_, err := s.db.Exec(
		`INSERT INTO watch_alerts (id, watch_item_id, item_name, severity, message, value, created_at, resolved)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		alert.ID, alert.WatchItemID, alert.ItemName, string(alert.Severity),
		alert.Message, valueJSON, now, boolToInt(alert.Resolved),
	)
	return err
}

func (s *Store) ListAlerts(resolved *bool, limit int) ([]*WatchAlert, error) {
	query := `SELECT id, watch_item_id, item_name, severity, message, value, created_at, resolved
	          FROM watch_alerts`
	var args []interface{}
	if resolved != nil {
		query += ` WHERE resolved = ?`
		args = append(args, boolToInt(*resolved))
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*WatchAlert
	for rows.Next() {
		var (
			a         WatchAlert
			severity  string
			valueJSON sql.NullString
			createdAt string
			resolved  int
		)
		if err := rows.Scan(&a.ID, &a.WatchItemID, &a.ItemName, &severity,
			&a.Message, &valueJSON, &createdAt, &resolved); err != nil {
			return nil, err
		}
		a.Severity = AlertSeverity(severity)
		a.Resolved = resolved == 1
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if valueJSON.Valid && valueJSON.String != "" {
			_ = json.Unmarshal([]byte(valueJSON.String), &a.Value)
		}
		alerts = append(alerts, &a)
	}
	return alerts, nil
}

func (s *Store) ResolveAlert(id string) error {
	_, err := s.db.Exec(`UPDATE watch_alerts SET resolved = 1 WHERE id = ?`, id)
	return err
}

func (s *Store) CountAlertsBySeverity() (info, warning, critical int, err error) {
	rows, err := s.db.Query(
		`SELECT severity, COUNT(*) FROM watch_alerts WHERE resolved = 0 GROUP BY severity`)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var sev string
		var cnt int
		if err := rows.Scan(&sev, &cnt); err != nil {
			return 0, 0, 0, err
		}
		switch AlertSeverity(sev) {
		case SeverityInfo:
			info = cnt
		case SeverityWarning:
			warning = cnt
		case SeverityCritical:
			critical = cnt
		}
	}
	return
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
