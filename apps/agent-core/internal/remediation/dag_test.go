package remediation

import (
	"testing"
)

func TestBuildDAG_Simple(t *testing.T) {
	steps := []RemediationStep{
		{StepID: 1, ToolName: "a", Status: StepStatusPending},
		{StepID: 2, ToolName: "b", DependsOn: []int{1}, Status: StepStatusPending},
		{StepID: 3, ToolName: "c", DependsOn: []int{1}, Status: StepStatusPending},
		{StepID: 4, ToolName: "d", DependsOn: []int{2, 3}, Status: StepStatusPending},
	}

	dag, err := BuildDAG(steps)
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	layers, err := dag.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	if len(layers) != 3 {
		t.Errorf("expected 3 layers, got %d", len(layers))
	}

	if len(layers[0]) != 1 || layers[0][0] != 1 {
		t.Errorf("layer 0 should be [1], got %v", layers[0])
	}

	if len(layers[1]) != 2 {
		t.Errorf("layer 1 should have 2 items, got %d", len(layers[1]))
	}

	if len(layers[2]) != 1 || layers[2][0] != 4 {
		t.Errorf("layer 2 should be [4], got %v", layers[2])
	}
}

func TestBuildDAG_CycleDetection(t *testing.T) {
	steps := []RemediationStep{
		{StepID: 1, ToolName: "a", DependsOn: []int{3}, Status: StepStatusPending},
		{StepID: 2, ToolName: "b", DependsOn: []int{1}, Status: StepStatusPending},
		{StepID: 3, ToolName: "c", DependsOn: []int{2}, Status: StepStatusPending},
	}

	_, err := BuildDAG(steps)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
}

func TestBuildDAG_MissingDependency(t *testing.T) {
	steps := []RemediationStep{
		{StepID: 1, ToolName: "a", Status: StepStatusPending},
		{StepID: 2, ToolName: "b", DependsOn: []int{99}, Status: StepStatusPending},
	}

	_, err := BuildDAG(steps)
	if err == nil {
		t.Fatal("expected missing dependency error, got nil")
	}
}

func TestBuildDAG_SingleStep(t *testing.T) {
	steps := []RemediationStep{
		{StepID: 1, ToolName: "a", Status: StepStatusPending},
	}

	dag, err := BuildDAG(steps)
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	order, err := dag.ExecutionOrder()
	if err != nil {
		t.Fatalf("ExecutionOrder failed: %v", err)
	}

	if len(order) != 1 || order[0] != 1 {
		t.Errorf("expected [1], got %v", order)
	}
}

func TestDAG_ReverseOrder(t *testing.T) {
	steps := []RemediationStep{
		{StepID: 1, ToolName: "a", Status: StepStatusPending},
		{StepID: 2, ToolName: "b", DependsOn: []int{1}, Status: StepStatusPending},
		{StepID: 3, ToolName: "c", DependsOn: []int{2}, Status: StepStatusPending},
	}

	dag, err := BuildDAG(steps)
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	rev, err := dag.ReverseOrder()
	if err != nil {
		t.Fatalf("ReverseOrder failed: %v", err)
	}

	if len(rev) != 3 || rev[0] != 3 || rev[1] != 2 || rev[2] != 1 {
		t.Errorf("expected [3,2,1], got %v", rev)
	}
}

func TestOverallRiskLevel(t *testing.T) {
	plan := &RemediationPlan{
		Steps: []RemediationStep{
			{StepID: 1, RiskLevel: RiskL0},
			{StepID: 2, RiskLevel: RiskL2},
			{StepID: 3, RiskLevel: RiskL1},
		},
	}

	if got := plan.OverallRiskLevel(); got != RiskL2 {
		t.Errorf("expected L2, got %s", got)
	}
}

func TestOverallRiskLevel_AllL0(t *testing.T) {
	plan := &RemediationPlan{
		Steps: []RemediationStep{
			{StepID: 1, RiskLevel: RiskL0},
			{StepID: 2, RiskLevel: RiskL0},
		},
	}

	if got := plan.OverallRiskLevel(); got != RiskL0 {
		t.Errorf("expected L0, got %s", got)
	}
}

func TestHasWriteOperations(t *testing.T) {
	readOnly := &RemediationPlan{
		Steps: []RemediationStep{
			{StepID: 1, RiskLevel: RiskL0},
		},
	}
	if readOnly.HasWriteOperations() {
		t.Error("expected no write operations for L0-only plan")
	}

	mixed := &RemediationPlan{
		Steps: []RemediationStep{
			{StepID: 1, RiskLevel: RiskL0},
			{StepID: 2, RiskLevel: RiskL2},
		},
	}
	if !mixed.HasWriteOperations() {
		t.Error("expected write operations for plan with L2 step")
	}
}
