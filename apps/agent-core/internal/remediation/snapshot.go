package remediation

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// SnapshotManager captures and restores state before/after remediation steps.
// Snapshots are stored in the local SQLite database.
type SnapshotManager struct {
	db *sql.DB
}

// Snapshot represents a captured state before a step executes.
type Snapshot struct {
	ID        int64                  `json:"id"`
	PlanID    string                 `json:"plan_id"`
	StepID    int                    `json:"step_id"`
	ToolName  string                 `json:"tool_name"`
	State     map[string]interface{} `json:"state"`
	CreatedAt time.Time              `json:"created_at"`
}

func NewSnapshotManager(db *sql.DB) *SnapshotManager {
	return &SnapshotManager{db: db}
}

// Migrate creates the snapshots table if it doesn't exist.
func (sm *SnapshotManager) Migrate() error {
	_, err := sm.db.Exec(`CREATE TABLE IF NOT EXISTS remediation_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		plan_id TEXT NOT NULL,
		step_id INTEGER NOT NULL,
		tool_name TEXT NOT NULL,
		state_json TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`)
	return err
}

// Capture records the current state relevant to a step before it executes.
// The state is gathered by running the step's verification tool (or a read-only equivalent).
func (sm *SnapshotManager) Capture(planID string, step *RemediationStep, state map[string]interface{}) (*Snapshot, error) {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot state: %w", err)
	}

	now := time.Now().UTC()
	result, err := sm.db.Exec(
		`INSERT INTO remediation_snapshots (plan_id, step_id, tool_name, state_json, created_at) VALUES (?, ?, ?, ?, ?)`,
		planID, step.StepID, step.ToolName, string(stateJSON), now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("save snapshot: %w", err)
	}

	id, _ := result.LastInsertId()
	slog.Info("[remediation] Snapshot captured", "plan_id", planID, "step_id", step.StepID, "tool", step.ToolName)

	return &Snapshot{
		ID:        id,
		PlanID:    planID,
		StepID:    step.StepID,
		ToolName:  step.ToolName,
		State:     state,
		CreatedAt: now,
	}, nil
}

// GetSnapshot retrieves the snapshot for a specific plan+step.
func (sm *SnapshotManager) GetSnapshot(planID string, stepID int) (*Snapshot, error) {
	var s Snapshot
	var stateJSON string
	var createdAt string

	err := sm.db.QueryRow(
		`SELECT id, plan_id, step_id, tool_name, state_json, created_at FROM remediation_snapshots WHERE plan_id = ? AND step_id = ? ORDER BY id DESC LIMIT 1`,
		planID, stepID,
	).Scan(&s.ID, &s.PlanID, &s.StepID, &s.ToolName, &stateJSON, &createdAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(stateJSON), &s.State)
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &s, nil
}

// CleanupPlan removes all snapshots for a given plan.
func (sm *SnapshotManager) CleanupPlan(planID string) error {
	_, err := sm.db.Exec(`DELETE FROM remediation_snapshots WHERE plan_id = ?`, planID)
	return err
}
