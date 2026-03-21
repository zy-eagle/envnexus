package diagnosis

import (
	"context"
	"log"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools"
)

type Engine struct {
	registry *tools.Registry
}

func NewEngine(registry *tools.Registry) *Engine {
	return &Engine{
		registry: registry,
	}
}

// RunDiagnosis simulates the 5-step diagnosis process defined in the proposal
func (e *Engine) RunDiagnosis(ctx context.Context, intent string) {
	log.Printf("[DiagnosisEngine] Starting diagnosis for intent: %s\n", intent)

	// 1. IntentParse (Mock)
	log.Println("[DiagnosisEngine] Step 1: IntentParse")
	
	// 2. EvidencePlan (Mock)
	log.Println("[DiagnosisEngine] Step 2: EvidencePlan - Selecting read_system_info tool")
	
	// 3. EvidenceCollect
	log.Println("[DiagnosisEngine] Step 3: EvidenceCollect")
	tool, ok := e.registry.Get("read_system_info")
	if ok {
		res, err := tool.Execute(ctx, nil)
		if err != nil {
			log.Printf("[DiagnosisEngine] Tool execution failed: %v\n", err)
		} else {
			log.Printf("[DiagnosisEngine] Tool result: %v\n", res.Summary)
		}
	}

	// 4. Reasoning (Mock)
	log.Println("[DiagnosisEngine] Step 4: Reasoning - Analyzing evidence")

	// 5. ActionDraft (Mock)
	log.Println("[DiagnosisEngine] Step 5: ActionDraft - Suggesting flush_dns")
	log.Println("[DiagnosisEngine] Diagnosis complete.")
}
