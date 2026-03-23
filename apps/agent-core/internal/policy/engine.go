package policy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools"
)

type ApprovalStatus string

const (
	StatusPending  ApprovalStatus = "pending_user"
	StatusApproved ApprovalStatus = "approved"
	StatusDenied   ApprovalStatus = "denied"
	StatusExpired  ApprovalStatus = "expired"
)

type ApprovalRequest struct {
	ID        string                 `json:"id"`
	ToolName  string                 `json:"tool_name"`
	RiskLevel string                 `json:"risk_level"`
	Params    map[string]interface{} `json:"params"`
	Status    ApprovalStatus         `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	ResultCh  chan bool              `json:"-"`
}

type Engine struct {
	mu               sync.RWMutex
	pendingApprovals map[string]*ApprovalRequest
}

func NewEngine() *Engine {
	return &Engine{
		pendingApprovals: make(map[string]*ApprovalRequest),
	}
}

func (e *Engine) Check(ctx context.Context, tool tools.Tool, params map[string]interface{}) (bool, error) {
	if tool.IsReadOnly() {
		return true, nil
	}

	reqID := generateID()
	req := &ApprovalRequest{
		ID:        reqID,
		ToolName:  tool.Name(),
		RiskLevel: tool.RiskLevel(),
		Params:    params,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		ResultCh:  make(chan bool, 1),
	}

	e.mu.Lock()
	e.pendingApprovals[reqID] = req
	e.mu.Unlock()

	log.Printf("[policy] Tool '%s' (risk: %s) requires approval. Request: %s\n", tool.Name(), tool.RiskLevel(), reqID)

	select {
	case <-ctx.Done():
		e.removeRequest(reqID)
		return false, ctx.Err()
	case <-time.After(time.Until(req.ExpiresAt)):
		e.removeRequest(reqID)
		return false, fmt.Errorf("approval request expired")
	case approved := <-req.ResultCh:
		e.removeRequest(reqID)
		if !approved {
			return false, fmt.Errorf("execution denied by user")
		}
		return true, nil
	}
}

func (e *Engine) removeRequest(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.pendingApprovals, id)
}

func (e *Engine) Resolve(id string, approved bool) error {
	e.mu.Lock()
	req, exists := e.pendingApprovals[id]
	e.mu.Unlock()

	if !exists {
		return fmt.Errorf("approval request not found or already resolved")
	}

	if approved {
		req.Status = StatusApproved
	} else {
		req.Status = StatusDenied
	}

	req.ResultCh <- approved
	return nil
}

func (e *Engine) GetPending() []*ApprovalRequest {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var list []*ApprovalRequest
	for _, req := range e.pendingApprovals {
		list = append(list, req)
	}
	return list
}

func generateID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
