package bootstrap

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/api"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/audit"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/device"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/diagnosis"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/enrollment"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/governance"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/lifecycle"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/policy"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/session"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools/network"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools/system"
)

type Bootstrapper struct {
	identityManager *device.IdentityManager
}

func NewBootstrapper() *Bootstrapper {
	// Default to a local config directory for now
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configDir := filepath.Join(homeDir, ".envnexus", "agent")

	return &Bootstrapper{
		identityManager: device.NewIdentityManager(configDir),
	}
}

func (b *Bootstrapper) Run(ctx context.Context) error {
	log.Println("Starting agent bootstrap sequence...")

	// 1. Load Local Config
	log.Println("[boot.load_local_config] Loading configuration...")

	// 2. Check Device Identity
	log.Println("[boot.check_device_identity] Checking device identity...")
	deviceID, err := b.identityManager.GetOrCreateDeviceID()
	if err != nil {
		return fmt.Errorf("failed to get/create device identity: %w", err)
	}
	log.Printf("Device ID: %s\n", deviceID)

	// 3. Enroll if needed
	log.Println("[boot.enroll_if_needed] Checking enrollment status...")
	
	var deviceToken string
	var tenantID string
	
	// Check if we already have an identity
	if id, err := b.identityManager.GetIdentity(); err == nil && id.Token != "" {
		log.Println("Found existing identity, skipping enrollment.")
		deviceToken = id.Token
		tenantID = id.TenantID
	} else {
		// For MVP, we use a hardcoded demo token if no local token exists
		// In production, this would be read from a bootstrap config file dropped by the installer
		demoToken := "demo-token"
		platformURL := "http://localhost:8080" // Should come from config

		enrollClient := enrollment.NewClient(platformURL)
		enrollResp, err := enrollClient.Enroll(ctx, demoToken, deviceID)
		if err != nil {
			log.Printf("Enrollment failed (might be already enrolled or platform unreachable): %v\n", err)
			// Fallback for MVP if not enrolled and no token saved
			deviceToken = "dummy-device-token"
			tenantID = "dummy-tenant"
		} else {
			log.Printf("Successfully enrolled! TenantID: %s\n", enrollResp.TenantID)
			deviceToken = enrollResp.DeviceToken
			tenantID = enrollResp.TenantID
			
			// Save the identity
			err = b.identityManager.SaveIdentity(&device.Identity{
				DeviceID: deviceID,
				TenantID: tenantID,
				Token:    deviceToken,
			})
			if err != nil {
				log.Printf("Failed to save identity: %v\n", err)
			}
		}
	}

	platformURL := "http://localhost:8080" // Should come from config

	// 4. Pull remote config
	log.Println("[boot.pull_remote_config] Pulling remote configuration...")
	lifecycleClient := lifecycle.NewClient(platformURL, deviceID, deviceToken)
	configResp, err := lifecycleClient.GetConfig(ctx)
	if err != nil {
		log.Printf("Failed to pull config: %v\n", err)
	} else {
		log.Printf("Successfully pulled config version: %s\n", configResp.ConfigVersion)
	}

	// Start heartbeat loop
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := lifecycleClient.Heartbeat(ctx); err != nil {
					log.Printf("Heartbeat failed: %v\n", err)
				} else {
					log.Println("Heartbeat successful")
				}
			}
		}
	}()

	// 5. Start local API
	log.Println("[boot.start_local_api] Starting local API...")
	
	// Initialize Policy & Approval Engine
	policyEngine := policy.NewEngine()
	
	localServer := api.NewLocalServer(17700, b.identityManager, policyEngine)
	if err := localServer.Start(); err != nil {
		log.Printf("Failed to start local API: %v\n", err)
	}

	// 6. Start workers
	log.Println("[boot.start_workers] Starting background workers...")
	
	// Initialize Tool Registry
	registry := tools.NewRegistry()
	registry.Register(network.NewReadNetworkConfigTool())
	registry.Register(network.NewFlushDNSTool()) // Add a write tool
	registry.Register(system.NewReadSystemInfoTool())

	// Initialize Audit Client
	auditClient := audit.NewClient(platformURL, deviceID)

	// Initialize Diagnosis Engine
	diagnosisEngine := diagnosis.NewEngine(registry)
	
	// Initialize Governance Engine
	governanceEngine := governance.NewEngine()
	governanceEngine.Start(ctx)

	// Start WebSocket client to connect to session-gateway
	wsURL := "ws://localhost:8081/ws/v1/agent" // Should come from config
	wsClient := session.NewWSClient(wsURL, deviceID, registry, auditClient, policyEngine)
	wsClient.Start(ctx)

	// Run a mock diagnosis to show it works
	go func() {
		time.Sleep(2 * time.Second)
		diagnosisEngine.RunDiagnosis(ctx, "network is slow")
	}()

	log.Println("[boot.ready] Bootstrap complete.")
	return nil
}
