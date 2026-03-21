package policy

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
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
	ID        string
	ToolName  string
	Params    map[string]interface{}
	Status    ApprovalStatus
	CreatedAt time.Time
	ExpiresAt time.Time
	ResultCh  chan bool // true if approved, false if denied/expired
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

// Check evaluates if a tool execution requires approval.
// If it does, it blocks until approval is granted or denied.
func (e *Engine) Check(ctx context.Context, tool tools.Tool, params map[string]interface{}) (bool, error) {
	if tool.IsReadOnly() {
		// Read-only tools pass immediately
		return true, nil
	}

	// Write tools require approval
	reqID := uuid.New().String()
	req := &ApprovalRequest{
		ID:        reqID,
		ToolName:  tool.Name(),
		Params:    params,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		ResultCh:  make(chan bool, 1),
	}

	e.mu.Lock()
	e.pendingApprovals[reqID] = req
	e.mu.Unlock()

	log.Printf("Tool '%s' requires approval. Created approval request: %s\n", tool.Name(), reqID)

	// In a real app, we would notify the local UI or platform about this pending request here.
	// For MVP, we'll just wait on the channel.

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

// Resolve is called by the UI or API to approve or deny a request
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

// GetPending returns all currently pending approval requests (useful for UI)
func (e *Engine) GetPending() []*ApprovalRequest {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var list []*ApprovalRequest
	for _, req := range e.pendingApprovals {
		list = append(list, req)
	}
	return list
}
