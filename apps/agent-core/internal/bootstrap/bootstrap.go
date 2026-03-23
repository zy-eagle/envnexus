package bootstrap

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/api"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/audit"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/config"
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
	configManager   *config.Manager
	configDir       string
}

func NewBootstrapper() *Bootstrapper {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configDir := filepath.Join(homeDir, ".envnexus", "agent")

	return &Bootstrapper{
		identityManager: device.NewIdentityManager(configDir),
		configManager:   config.NewManager(configDir),
		configDir:       configDir,
	}
}

func (b *Bootstrapper) Run(ctx context.Context) error {
	log.Println("[boot] Starting agent bootstrap sequence...")
	cfg := b.configManager.Get()

	// Step 1: Check device identity / enroll if needed
	var deviceID, deviceToken, tenantID string

	if b.identityManager.HasIdentity() {
		id, _ := b.identityManager.GetIdentity()
		deviceID = id.DeviceID
		deviceToken = id.Token
		tenantID = id.TenantID
		log.Printf("[boot] Loaded existing identity: device=%s, tenant=%s\n", deviceID, tenantID)
	} else {
		log.Println("[boot] No identity found, attempting enrollment...")
		if cfg.EnrollmentToken == "" {
			log.Println("[boot] No enrollment token configured. Set ENX_ENROLLMENT_TOKEN or wait for installer.")
			log.Println("[boot] Running in standalone mode...")
			deviceID = "standalone"
		} else {
			enrollClient := enrollment.NewClient(cfg.PlatformURL)
			resp, err := enrollClient.Enroll(ctx, cfg.EnrollmentToken, cfg.AgentVersion)
			if err != nil {
				log.Printf("[boot] Enrollment failed: %v\n", err)
				log.Println("[boot] Running in standalone mode...")
				deviceID = "standalone"
			} else {
				deviceID = resp.DeviceID
				deviceToken = resp.DeviceToken
				tenantID = resp.TenantID
				log.Printf("[boot] Enrolled successfully: device=%s, tenant=%s\n", deviceID, tenantID)

				if err := b.identityManager.SaveIdentity(&device.Identity{
					DeviceID: deviceID,
					TenantID: tenantID,
					Token:    deviceToken,
				}); err != nil {
					log.Printf("[boot] Failed to save identity: %v\n", err)
				}

				b.configManager.Update(func(c *config.AgentConfig) {
					c.ConfigVersion = resp.ConfigVersion
				})
			}
		}
	}

	// Step 2: Pull remote config
	if deviceToken != "" {
		log.Println("[boot] Pulling remote configuration...")
		lifecycleClient := lifecycle.NewClient(cfg.PlatformURL, deviceID, deviceToken)
		configResp, err := lifecycleClient.GetConfig(ctx, cfg.ConfigVersion)
		if err != nil {
			log.Printf("[boot] Config pull failed: %v\n", err)
		} else if configResp.HasUpdate {
			log.Printf("[boot] Config updated to version %d\n", configResp.ConfigVersion)
			b.configManager.Update(func(c *config.AgentConfig) {
				c.ConfigVersion = configResp.ConfigVersion
			})
		} else {
			log.Println("[boot] Config is up to date")
		}
	}

	// Step 3: Initialize tool registry
	registry := tools.NewRegistry()
	registry.Register(network.NewReadNetworkConfigTool())
	registry.Register(network.NewFlushDNSTool())
	registry.Register(system.NewReadSystemInfoTool())
	log.Printf("[boot] Registered %d tools\n", registry.Count())

	// Step 4: Initialize engines
	policyEngine := policy.NewEngine()
	auditClient := audit.NewClient(cfg.PlatformURL, deviceID, deviceToken)

	// Start audit batch flush loop
	go auditClient.StartFlushLoop(ctx, 15*time.Second)

	// Step 5: Start local API
	localServer := api.NewLocalServer(17700, b.identityManager, policyEngine)
	if err := localServer.Start(); err != nil {
		log.Printf("[boot] Failed to start local API: %v\n", err)
	}

	// Step 6: Start heartbeat loop
	if deviceToken != "" {
		lifecycleClient := lifecycle.NewClient(cfg.PlatformURL, deviceID, deviceToken)
		go func() {
			interval := time.Duration(cfg.HeartbeatSeconds) * time.Second
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					currentCfg := b.configManager.Get()
					if err := lifecycleClient.Heartbeat(ctx, currentCfg.AgentVersion, currentCfg.ConfigVersion); err != nil {
						log.Printf("[heartbeat] Failed: %v\n", err)
					}
				}
			}
		}()
	}

	// Step 7: Start governance engine
	governanceEngine := governance.NewEngine()
	governanceEngine.Start(ctx)

	// Step 8: Connect to session gateway
	if deviceToken != "" {
		wsClient := session.NewWSClient(cfg.WSURL, deviceID, tenantID, registry, auditClient, policyEngine)
		wsClient.Start(ctx)
	}

	// Step 9: Initialize diagnosis engine
	diagnosisEngine := diagnosis.NewEngine(registry)
	_ = diagnosisEngine

	log.Println("[boot] Bootstrap complete. Agent is running.")
	return nil
}
