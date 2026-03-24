package bootstrap

import (
	"context"
	"log/slog"
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
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/llm/providers"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/llm/router"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/policy"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/session"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/store"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools/cache"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools/network"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools/service"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/tools/system"
)

type Bootstrapper struct {
	identityManager *device.IdentityManager
	configManager   *config.Manager
	configDir       string
	localServer     *api.LocalServer
	localStore      *store.Store
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
	slog.Info("[boot] Starting agent bootstrap sequence...")
	cfg := b.configManager.Get()

	// Step 0: Initialize local SQLite store
	dataDir := filepath.Join(b.configDir, "data")
	localStore, err := store.New(dataDir)
	if err != nil {
		slog.Warn("[boot] SQLite store init failed, running without local persistence", "error", err)
	} else {
		b.localStore = localStore
	}

	// Step 1: Check device identity / enroll if needed
	var deviceID, deviceToken, tenantID string
	platformReachable := true

	if b.identityManager.HasIdentity() {
		id, _ := b.identityManager.GetIdentity()
		deviceID = id.DeviceID
		deviceToken = id.Token
		tenantID = id.TenantID
		slog.Info("[boot] Loaded existing identity", "device", deviceID, "tenant", tenantID)
	} else {
		slog.Info("[boot] No identity found, attempting enrollment...")
		if cfg.EnrollmentToken == "" {
			slog.Info("[boot] No enrollment token configured. Set ENX_ENROLLMENT_TOKEN or wait for installer.")
			slog.Info("[boot] Running in standalone mode...")
			deviceID = "standalone"
		} else {
			enrollClient := enrollment.NewClient(cfg.PlatformURL)
			resp, err := enrollClient.Enroll(ctx, cfg.EnrollmentToken, cfg.AgentVersion)
			if err != nil {
				slog.Warn("[boot] Enrollment failed, running in standalone mode", "error", err)
				deviceID = "standalone"
				platformReachable = false
			} else {
				deviceID = resp.DeviceID
				deviceToken = resp.DeviceToken
				tenantID = resp.TenantID
				slog.Info("[boot] Enrolled successfully", "device", deviceID, "tenant", tenantID)

				if err := b.identityManager.SaveIdentity(&device.Identity{
					DeviceID: deviceID,
					TenantID: tenantID,
					Token:    deviceToken,
				}); err != nil {
					slog.Error("[boot] Failed to save identity", "error", err)
				}

				b.configManager.Update(func(c *config.AgentConfig) {
					c.ConfigVersion = resp.ConfigVersion
				})
			}
		}
	}

	// Step 2: Pull remote config (skip in offline mode)
	if deviceToken != "" && platformReachable {
		slog.Info("[boot] Pulling remote configuration...")
		lifecycleClient := lifecycle.NewClient(cfg.PlatformURL, deviceID, deviceToken)
		configResp, err := lifecycleClient.GetConfig(ctx, cfg.ConfigVersion)
		if err != nil {
			slog.Warn("[boot] Config pull failed, entering offline mode", "error", err)
			platformReachable = false
		} else if configResp.HasUpdate {
			slog.Info("[boot] Config updated", "config_version", configResp.ConfigVersion)
			b.configManager.Update(func(c *config.AgentConfig) {
				c.ConfigVersion = configResp.ConfigVersion
			})
		} else {
			slog.Info("[boot] Config is up to date")
		}
	}

	// Step 3: Initialize tool registry
	registry := tools.NewRegistry()
	registry.Register(network.NewReadNetworkConfigTool())
	registry.Register(network.NewFlushDNSTool())
	registry.Register(system.NewReadSystemInfoTool())
	registry.Register(system.NewReadDiskUsageTool())
	registry.Register(system.NewReadProcessListTool())
	registry.Register(system.NewProxyToggleTool())
	registry.Register(system.NewConfigModifyTool())
	registry.Register(service.NewRestartTool())
	registry.Register(service.NewContainerReloadTool())
	registry.Register(cache.NewRebuildTool())

	if !platformReachable {
		slog.Info("[boot] Offline mode: only read-only tools available", "count", countReadOnly(registry))
	} else {
		slog.Info("[boot] Registered tools", "count", registry.Count())
	}

	// Step 4: Initialize LLM router
	llmRouter := b.initLLMRouter()

	// Step 5: Initialize engines
	policyEngine := policy.NewEngine()
	if deviceToken != "" && platformReachable {
		policyEngine.SetPlatformClient(policy.NewPlatformClient(cfg.PlatformURL, deviceToken))
	}

	auditClient := audit.NewClient(cfg.PlatformURL, deviceID, deviceToken)
	if platformReachable {
		go auditClient.StartFlushLoop(ctx, 15*time.Second)
	}

	diagnosisEngine := diagnosis.NewEngine(registry, llmRouter)

	// Step 6: Start governance engine
	governanceEngine := governance.NewEngine()
	if localStore != nil {
		governanceEngine.SetStore(localStore)
	}
	governanceEngine.Start(ctx)

	// Step 7: Start local API
	localServer := api.NewLocalServer(17700, b.identityManager, policyEngine, diagnosisEngine)
	if err := localServer.Start(); err != nil {
		slog.Error("[boot] Failed to start local API", "error", err)
	}
	b.localServer = localServer

	// Step 8: Start heartbeat loop (only when platform reachable)
	if deviceToken != "" && platformReachable {
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
						slog.Warn("[heartbeat] Failed", "error", err)
					}
				}
			}
		}()
	}

	// Step 9: Connect to session gateway (only when platform reachable)
	if deviceToken != "" && platformReachable {
		tokenProvider := lifecycle.NewClient(cfg.PlatformURL, deviceID, deviceToken)
		wsClient := session.NewWSClient(cfg.WSURL, deviceID, tenantID, tokenProvider, registry, auditClient, policyEngine, diagnosisEngine)
		wsClient.Start(ctx)
	}

	if !platformReachable {
		slog.Info("[boot] Bootstrap complete. Agent is running in OFFLINE mode (read-only only).")
	} else {
		slog.Info("[boot] Bootstrap complete. Agent is running.")
	}
	return nil
}

func (b *Bootstrapper) Shutdown(ctx context.Context) {
	slog.Info("[boot] Shutting down...")
	if b.localServer != nil {
		if err := b.localServer.Stop(ctx); err != nil {
			slog.Error("[boot] Failed to stop local server", "error", err)
		}
	}
	if b.localStore != nil {
		if err := b.localStore.Close(); err != nil {
			slog.Error("[boot] Failed to close store", "error", err)
		}
	}
	slog.Info("[boot] Shutdown complete")
}

func (b *Bootstrapper) initLLMRouter() *router.Router {
	llmRouter := router.NewRouter(90 * time.Second)

	if apiKey := os.Getenv("ENX_OPENAI_API_KEY"); apiKey != "" {
		p := providers.NewOpenAIProvider(router.ProviderConfig{
			Name:    "openai",
			BaseURL: envOrDefault("ENX_OPENAI_BASE_URL", "https://api.openai.com/v1"),
			APIKey:  apiKey,
			Model:   envOrDefault("ENX_OPENAI_MODEL", "gpt-4o-mini"),
		})
		llmRouter.RegisterProvider(p)
		slog.Info("[boot] Registered OpenAI provider")
	}

	if apiKey := os.Getenv("ENX_OPENROUTER_API_KEY"); apiKey != "" {
		p := providers.NewOpenRouterProvider(router.ProviderConfig{
			Name:    "openrouter",
			BaseURL: envOrDefault("ENX_OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
			APIKey:  apiKey,
			Model:   envOrDefault("ENX_OPENROUTER_MODEL", ""),
		})
		llmRouter.RegisterProvider(p)
		slog.Info("[boot] Registered OpenRouter provider")
	}

	if apiKey := os.Getenv("ENX_DEEPSEEK_API_KEY"); apiKey != "" {
		p := providers.NewDeepSeekProvider(router.ProviderConfig{
			Name:    "deepseek",
			BaseURL: envOrDefault("ENX_DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
			APIKey:  apiKey,
			Model:   envOrDefault("ENX_DEEPSEEK_MODEL", "deepseek-chat"),
		})
		llmRouter.RegisterProvider(p)
		slog.Info("[boot] Registered DeepSeek provider")
	}

	if apiKey := os.Getenv("ENX_ANTHROPIC_API_KEY"); apiKey != "" {
		p := providers.NewAnthropicProvider(router.ProviderConfig{
			Name:    "anthropic",
			BaseURL: envOrDefault("ENX_ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
			APIKey:  apiKey,
			Model:   envOrDefault("ENX_ANTHROPIC_MODEL", "claude-sonnet-4-20250514"),
		})
		llmRouter.RegisterProvider(p)
		slog.Info("[boot] Registered Anthropic provider")
	}

	if apiKey := os.Getenv("ENX_GEMINI_API_KEY"); apiKey != "" {
		p := providers.NewGeminiProvider(router.ProviderConfig{
			Name:    "gemini",
			BaseURL: envOrDefault("ENX_GEMINI_BASE_URL", "https://generativelanguage.googleapis.com"),
			APIKey:  apiKey,
			Model:   envOrDefault("ENX_GEMINI_MODEL", "gemini-2.0-flash"),
		})
		llmRouter.RegisterProvider(p)
		slog.Info("[boot] Registered Gemini provider")
	}

	ollamaURL := envOrDefault("ENX_OLLAMA_URL", "http://localhost:11434")
	p := providers.NewOllamaProvider(router.ProviderConfig{
		Name:    "ollama",
		BaseURL: ollamaURL,
		Model:   envOrDefault("ENX_OLLAMA_MODEL", "llama3.2"),
	})
	llmRouter.RegisterProvider(p)
	slog.Info("[boot] Registered Ollama provider (fallback)")

	if primary := os.Getenv("ENX_LLM_PRIMARY"); primary != "" {
		llmRouter.SetPrimary(primary)
	}

	slog.Info("[boot] LLM router initialized", "providers", llmRouter.ListProviders())
	return llmRouter
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func countReadOnly(registry *tools.Registry) int {
	count := 0
	for _, t := range registry.List() {
		if t.IsReadOnly() {
			count++
		}
	}
	return count
}
