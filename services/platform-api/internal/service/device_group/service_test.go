package device_group

import (
	"context"
	"testing"

	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
)

type memGroupRepo struct {
	groups  map[string]*domain.DeviceGroup
	members map[string][]*domain.DeviceGroupMember
	batches map[string]*domain.BatchTask
}

func newMemGroupRepo() *memGroupRepo {
	return &memGroupRepo{
		groups:  make(map[string]*domain.DeviceGroup),
		members: make(map[string][]*domain.DeviceGroupMember),
		batches: make(map[string]*domain.BatchTask),
	}
}

func (r *memGroupRepo) CreateGroup(_ context.Context, g *domain.DeviceGroup) error {
	r.groups[g.ID] = g
	return nil
}
func (r *memGroupRepo) GetGroup(_ context.Context, id string) (*domain.DeviceGroup, error) {
	g, ok := r.groups[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return g, nil
}
func (r *memGroupRepo) ListGroups(_ context.Context, tenantID string) ([]*domain.DeviceGroup, error) {
	var result []*domain.DeviceGroup
	for _, g := range r.groups {
		if g.TenantID == tenantID {
			result = append(result, g)
		}
	}
	return result, nil
}
func (r *memGroupRepo) UpdateGroup(_ context.Context, g *domain.DeviceGroup) error {
	r.groups[g.ID] = g
	return nil
}
func (r *memGroupRepo) DeleteGroup(_ context.Context, id string) error {
	delete(r.groups, id)
	delete(r.members, id)
	return nil
}
func (r *memGroupRepo) AddMembers(_ context.Context, members []*domain.DeviceGroupMember) error {
	for _, m := range members {
		r.members[m.DeviceGroupID] = append(r.members[m.DeviceGroupID], m)
	}
	return nil
}
func (r *memGroupRepo) RemoveMember(_ context.Context, groupID, deviceID string) error {
	mems := r.members[groupID]
	var filtered []*domain.DeviceGroupMember
	for _, m := range mems {
		if m.DeviceID != deviceID {
			filtered = append(filtered, m)
		}
	}
	r.members[groupID] = filtered
	return nil
}
func (r *memGroupRepo) ListMembers(_ context.Context, groupID string) ([]*domain.DeviceGroupMember, error) {
	return r.members[groupID], nil
}
func (r *memGroupRepo) CreateBatchTask(_ context.Context, t *domain.BatchTask) error {
	r.batches[t.ID] = t
	return nil
}
func (r *memGroupRepo) GetBatchTask(_ context.Context, id string) (*domain.BatchTask, error) {
	bt, ok := r.batches[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return bt, nil
}
func (r *memGroupRepo) ListBatchTasks(_ context.Context, tenantID string) ([]*domain.BatchTask, error) {
	var result []*domain.BatchTask
	for _, bt := range r.batches {
		if bt.TenantID == tenantID {
			result = append(result, bt)
		}
	}
	return result, nil
}
func (r *memGroupRepo) UpdateBatchTask(_ context.Context, t *domain.BatchTask) error {
	r.batches[t.ID] = t
	return nil
}

func TestCreateGroupAndMembers(t *testing.T) {
	repo := newMemGroupRepo()
	svc := NewService(repo)
	ctx := context.Background()

	g, err := svc.CreateGroup(ctx, "t1", "production", "prod servers", "", "u1")
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if g.Name != "production" {
		t.Errorf("expected name 'production', got %q", g.Name)
	}

	err = svc.AddMembers(ctx, g.ID, []string{"d1", "d2", "d3"})
	if err != nil {
		t.Fatalf("AddMembers: %v", err)
	}

	members, err := svc.ListMembers(ctx, g.ID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members, got %d", len(members))
	}
}

func TestCreateBatchTask(t *testing.T) {
	repo := newMemGroupRepo()
	svc := NewService(repo)
	ctx := context.Background()

	g, _ := svc.CreateGroup(ctx, "t1", "staging", "", "", "u1")
	_ = svc.AddMembers(ctx, g.ID, []string{"d1", "d2"})

	bt, err := svc.CreateBatchTask(ctx, "t1", g.ID, "cmd1", "rolling", "u1", 1)
	if err != nil {
		t.Fatalf("CreateBatchTask: %v", err)
	}
	if bt.TotalDevices != 2 {
		t.Errorf("expected 2 total_devices, got %d", bt.TotalDevices)
	}
	if bt.Status != "pending" {
		t.Errorf("expected 'pending', got %q", bt.Status)
	}
}

func TestBatchTaskEmptyGroup(t *testing.T) {
	repo := newMemGroupRepo()
	svc := NewService(repo)
	ctx := context.Background()

	g, _ := svc.CreateGroup(ctx, "t1", "empty", "", "", "u1")
	_, err := svc.CreateBatchTask(ctx, "t1", g.ID, "cmd1", "", "u1", 0)
	if err == nil {
		t.Error("expected error for empty group")
	}
}

func TestUpdateBatchTaskProgress(t *testing.T) {
	repo := newMemGroupRepo()
	svc := NewService(repo)
	ctx := context.Background()

	g, _ := svc.CreateGroup(ctx, "t1", "test", "", "", "u1")
	_ = svc.AddMembers(ctx, g.ID, []string{"d1", "d2", "d3"})
	bt, _ := svc.CreateBatchTask(ctx, "t1", g.ID, "cmd1", "all_at_once", "u1", 0)

	err := svc.UpdateBatchTaskProgress(ctx, bt.ID, 2, 0)
	if err != nil {
		t.Fatalf("UpdateBatchTaskProgress: %v", err)
	}
	updated, _ := svc.GetBatchTask(ctx, bt.ID)
	if updated.Status != "executing" {
		t.Errorf("expected 'executing', got %q", updated.Status)
	}

	err = svc.UpdateBatchTaskProgress(ctx, bt.ID, 3, 0)
	if err != nil {
		t.Fatalf("UpdateBatchTaskProgress final: %v", err)
	}
	final, _ := svc.GetBatchTask(ctx, bt.ID)
	if final.Status != "completed" {
		t.Errorf("expected 'completed', got %q", final.Status)
	}
}
