package watchlist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

// DefaultPlatformSyncInterval controls how often the agent pulls governance
// rules from the platform when no interval is configured.
const DefaultPlatformSyncInterval = 5 * time.Minute

// PlatformRule mirrors services/platform-api domain.GovernanceRule but is
// declared locally so the agent never imports platform-api.
type PlatformRule struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	RuleType      string `json:"rule_type"`
	ConditionJSON string `json:"condition"`
	ActionJSON    string `json:"action"`
	Severity      string `json:"severity"`
	Enabled       bool   `json:"enabled"`
}

// PlatformToolPermission mirrors services/platform-api domain.ToolPermission.
type PlatformToolPermission struct {
	ID       string `json:"id"`
	ToolName string `json:"tool_name"`
	RoleID   string `json:"role_id,omitempty"`
	Allowed  bool   `json:"allowed"`
	MaxRisk  string `json:"max_risk,omitempty"`
}

// platformSyncResponse is the JSON shape returned by
// /agent/v1/governance/sync. The outer envelope follows the standard platform
// response wrapper: {"success": true, "data": {"rules": [...], "tool_permissions": [...]}}
type platformSyncResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Rules           []PlatformRule           `json:"rules"`
		ToolPermissions []PlatformToolPermission `json:"tool_permissions"`
	} `json:"data"`
}

// rule condition JSON structure — a subset we know how to translate into a
// WatchItem. Unknown shapes are ignored gracefully.
type ruleCondition struct {
	ToolName   string                 `json:"tool_name"`
	ToolParams map[string]interface{} `json:"tool_params,omitempty"`
	Type       string                 `json:"type"`
	JSONPath   string                 `json:"json_path,omitempty"`
	Operator   string                 `json:"operator,omitempty"`
	Threshold  interface{}            `json:"threshold,omitempty"`
	Pattern    string                 `json:"pattern,omitempty"`
	Interval   string                 `json:"interval,omitempty"`
}

// PlatformSync periodically pulls governance rules + tool permissions from
// the platform and registers them as WatchItems on the Manager.
type PlatformSync struct {
	manager     *Manager
	baseURL     string
	deviceToken string
	interval    time.Duration
	httpClient  *http.Client

	mu        sync.Mutex
	stopCh    chan struct{}
	running   bool
	toolPerms []PlatformToolPermission
}

// ToolPermissionProvider lets other parts of the agent (e.g. policy engine)
// see the platform-pushed tool permission list.
type ToolPermissionProvider interface {
	ToolPermissions() []PlatformToolPermission
}

func NewPlatformSync(manager *Manager, baseURL, deviceToken string, interval time.Duration) *PlatformSync {
	if interval <= 0 {
		interval = DefaultPlatformSyncInterval
	}
	return &PlatformSync{
		manager:     manager,
		baseURL:     strings.TrimRight(baseURL, "/"),
		deviceToken: deviceToken,
		interval:    interval,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

// Start launches a background goroutine that triggers an initial sync and
// then fires every interval. Returns immediately. Safe to call once.
func (p *PlatformSync) Start(ctx context.Context) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.stopCh = make(chan struct{})
	p.mu.Unlock()

	go func() {
		// Kick off one immediate sync; failures are logged but do not stop the loop.
		if err := p.SyncOnce(ctx); err != nil {
			slog.Warn("[watchlist/platform_sync] initial sync failed", "error", err)
		}
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-p.stopCh:
				return
			case <-ticker.C:
				if err := p.SyncOnce(ctx); err != nil {
					slog.Warn("[watchlist/platform_sync] sync failed", "error", err)
				}
			}
		}
	}()
}

// Stop signals the background loop to exit.
func (p *PlatformSync) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return
	}
	p.running = false
	if p.stopCh != nil {
		close(p.stopCh)
	}
}

// ToolPermissions returns the latest platform-pushed tool permission list.
func (p *PlatformSync) ToolPermissions() []PlatformToolPermission {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]PlatformToolPermission, len(p.toolPerms))
	copy(out, p.toolPerms)
	return out
}

// SyncOnce performs a single pull + apply cycle. Exported so tests can drive it.
func (p *PlatformSync) SyncOnce(ctx context.Context) error {
	if p.manager == nil {
		return fmt.Errorf("platform sync: manager not configured")
	}
	if p.baseURL == "" {
		return fmt.Errorf("platform sync: baseURL not configured")
	}

	url := p.baseURL + "/agent/v1/governance/sync"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if p.deviceToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.deviceToken)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("platform sync: HTTP %d: %s", resp.StatusCode, truncateBody(string(body), 200))
	}

	var parsed platformSyncResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	p.mu.Lock()
	p.toolPerms = parsed.Data.ToolPermissions
	p.mu.Unlock()

	p.applyRules(ctx, parsed.Data.Rules)
	slog.Info("[watchlist/platform_sync] applied",
		"rules", len(parsed.Data.Rules),
		"tool_permissions", len(parsed.Data.ToolPermissions),
	)
	return nil
}

// applyRules reconciles platform-sourced rules with the local scheduler.
// Existing items from SourcePlatform that no longer appear on the platform
// are removed; new/updated items are registered.
func (p *PlatformSync) applyRules(ctx context.Context, rules []PlatformRule) {
	existing, err := p.manager.ListItems(string(SourcePlatform))
	if err != nil {
		slog.Warn("[watchlist/platform_sync] list existing platform items failed", "error", err)
		existing = nil
	}
	desiredIDs := make(map[string]struct{}, len(rules))

	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		item, ok := ruleToWatchItem(r)
		if !ok {
			slog.Debug("[watchlist/platform_sync] rule skipped (cannot translate)", "rule_id", r.ID, "name", r.Name)
			continue
		}
		desiredIDs[item.ID] = struct{}{}
		if regErr := p.manager.ConfirmItems(ctx, []WatchItem{*item}); regErr != nil {
			slog.Warn("[watchlist/platform_sync] register rule failed", "rule_id", r.ID, "error", regErr)
		}
	}

	for _, cur := range existing {
		if _, keep := desiredIDs[cur.ID]; keep {
			continue
		}
		if err := p.manager.DeleteItem(cur.ID); err != nil {
			slog.Warn("[watchlist/platform_sync] remove stale rule failed", "id", cur.ID, "error", err)
		}
	}
}

// ruleToWatchItem converts a platform rule into a local WatchItem. Returns
// (nil, false) when the rule cannot be translated safely.
func ruleToWatchItem(r PlatformRule) (*WatchItem, bool) {
	if r.ConditionJSON == "" {
		return nil, false
	}
	var cond ruleCondition
	if err := json.Unmarshal([]byte(r.ConditionJSON), &cond); err != nil {
		return nil, false
	}
	if cond.ToolName == "" {
		return nil, false
	}

	interval := DefaultPlatformSyncInterval
	if cond.Interval != "" {
		if d, err := time.ParseDuration(cond.Interval); err == nil && d > 0 {
			interval = d
		}
	}

	wc := WatchCondition{
		JSONPath:  cond.JSONPath,
		Threshold: cond.Threshold,
		Pattern:   cond.Pattern,
	}
	switch cond.Type {
	case string(CondThreshold):
		wc.Type = CondThreshold
	case string(CondExists):
		wc.Type = CondExists
	case string(CondReachable):
		wc.Type = CondReachable
	case string(CondContains):
		wc.Type = CondContains
	case string(CondCustom):
		wc.Type = CondCustom
	default:
		wc.Type = CondThreshold
	}
	switch cond.Operator {
	case string(OpLT):
		wc.Operator = OpLT
	case string(OpGT):
		wc.Operator = OpGT
	case string(OpEQ):
		wc.Operator = OpEQ
	case string(OpNE):
		wc.Operator = OpNE
	case string(OpContains):
		wc.Operator = OpContains
	case string(OpNotContains):
		wc.Operator = OpNotContains
	}

	id := r.ID
	if id == "" {
		id = ulid.Make().String()
	} else {
		id = "platform:" + id
	}

	return &WatchItem{
		ID:          id,
		Name:        r.Name,
		Description: r.Description,
		Source:      SourcePlatform,
		ToolName:    cond.ToolName,
		ToolParams:  cond.ToolParams,
		Condition:   wc,
		Interval:    interval,
		Enabled:     true,
		CreatedAt:   time.Now().UTC(),
	}, true
}

func truncateBody(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
