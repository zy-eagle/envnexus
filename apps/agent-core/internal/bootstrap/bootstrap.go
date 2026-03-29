package bootstrap

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/zy-eagle/envnexus/apps/agent-core/internal/activation"
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
	agentruntime "github.com/zy-eagle/envnexus/apps/agent-core/internal/runtime"
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
	runtime         *agentruntime.Runtime
}

// NewBootstrapper creates a bootstrapper with the default config directory (~/.envnexus/agent).
// Use SetDataDir before Run() to override the data/config location (e.g. when running under Electron).
func NewBootstrapper() *Bootstrapper {
	configDir := DefaultConfigDir()
	return &Bootstrapper{
		identityManager: device.NewIdentityManager(configDir),
		configManager:   config.NewManager(configDir),
		configDir:       configDir,
	}
}

// SetDataDir overrides the config/data directory (used by Electron to align with install path).
func (b *Bootstrapper) SetDataDir(dir string) {
	b.configDir = dir
	b.identityManager = device.NewIdentityManager(dir)
	b.configManager = config.NewManager(dir)
}

func DefaultConfigDir() string {
	if dir := portableDataDir(); dir != "" {
		return dir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".envnexus", "agent")
}

// portableDataDir returns the data directory when a `.portable` marker
// exists next to the executable or in a parent directory (ZIP/portable distribution).
// When bundled inside Electron, the binary lives at resources/bin/enx-agent.exe
// while .portable is at the app root, so we check up to 3 levels.
func portableDataDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exePath)
	for i := 0; i < 4; i++ {
		marker := filepath.Join(dir, ".portable")
		if _, err := os.Stat(marker); err == nil {
			dataDir := filepath.Join(dir, "data")
			os.MkdirAll(dataDir, 0o755)
			return dataDir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func (b *Bootstrapper) ApplyCLIOverrides(o config.CLIOverrides) {
	b.configManager.ApplyCLIOverrides(o)
}

func (b *Bootstrapper) Run(ctx context.Context) error {
	slog.Info("[boot] Starting agent bootstrap sequence...")
	cfg := b.configManager.Get()

	// Step 0: Initialize local SQLite store (agentdb lives alongside config)
	dataDir := filepath.Join(b.configDir, "data")
	localStore, err := store.New(dataDir)
	if err != nil {
		slog.Warn("[boot] SQLite store init failed, running without local persistence", "error", err)
	} else {
		b.localStore = localStore
	}

	// Step 0.5: Activation check (if activation_mode is configured)
	var activationMgr *activation.Manager
	if cfg.ActivationMode != "" {
		activationMgr = activation.NewManager(cfg.PlatformURL, cfg.ActivationMode, cfg.ActivationKey, b.configDir)
		if !activationMgr.IsActivated() {
			slog.Info("[boot] Activation required", "mode", cfg.ActivationMode)
			if err := activationMgr.Activate(ctx); err != nil {
				slog.Warn("[boot] Activation attempt failed", "error", err)
			}
			if cfg.ActivationMode == "manual" && !activationMgr.IsActivated() {
				slog.Info("[boot] Waiting for admin to bind device", "device_code", activationMgr.GetDeviceCode())
			}
		} else {
			slog.Info("[boot] Device is activated", "device_code", activationMgr.GetDeviceCode())
		}
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
	var remoteModelProfile *remoteModelConfig
	if deviceToken != "" && platformReachable {
		slog.Info("[boot] Pulling remote configuration...")
		lifecycleClient := lifecycle.NewClient(cfg.PlatformURL, deviceID, deviceToken)
		configResp, err := lifecycleClient.GetConfig(ctx, cfg.ConfigVersion)
		if err != nil {
			slog.Warn("[boot] Config pull failed, entering offline mode", "error", err)
			platformReachable = false
		} else {
			if configResp.HasUpdate {
				slog.Info("[boot] Config updated", "config_version", configResp.ConfigVersion)
				b.configManager.Update(func(c *config.AgentConfig) {
					c.ConfigVersion = configResp.ConfigVersion
				})
			} else {
				slog.Info("[boot] Config is up to date")
			}
			if len(configResp.ModelProfile) > 0 {
				var mp remoteModelConfig
				if err := json.Unmarshal(configResp.ModelProfile, &mp); err != nil {
					slog.Warn("[boot] Failed to parse remote model profile", "error", err)
				} else if mp.Provider != "" {
					remoteModelProfile = &mp
					slog.Info("[boot] Remote model profile loaded", "provider", mp.Provider, "model", mp.ModelName)
				}
			}
		}
	}

	// Step 3: Initialize tool registry
	registry := tools.NewRegistry()
	registry.Register(network.NewReadNetworkConfigTool())
	registry.Register(network.NewReadProxyConfigTool())
	registry.Register(network.NewFlushDNSTool())
	registry.Register(network.NewPingTool())
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
	llmRouter := b.initLLMRouter(remoteModelProfile)

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

	// Step 7: Start local API
	localServer := api.NewLocalServer(17700, b.identityManager, policyEngine, diagnosisEngine)
	localServer.SetPlatformConnected(platformReachable)
	localServer.SetGovernanceEngine(governanceEngine)
	if activationMgr != nil {
		localServer.SetActivationManager(activationMgr)
	}
	if localStore != nil {
		localServer.SetStore(localStore)
	}
	if err := localServer.Start(); err != nil {
		slog.Error("[boot] Failed to start local API", "error", err)
	}
	b.localServer = localServer

	// Step 8: Initialize runtime with background tasks
	rt := agentruntime.New()
	b.runtime = rt

	rt.Register(agentruntime.Task{
		Name:     "governance_baseline",
		Interval: 5 * time.Minute,
		Fn: func(ctx context.Context) error {
			governanceEngine.RunBaselineCheck(ctx)
			return nil
		},
	})

	if deviceToken != "" && platformReachable {
		lifecycleClient := lifecycle.NewClient(cfg.PlatformURL, deviceID, deviceToken)
		rt.Register(agentruntime.Task{
			Name:     "heartbeat",
			Interval: time.Duration(cfg.HeartbeatSeconds) * time.Second,
			Fn: func(ctx context.Context) error {
				currentCfg := b.configManager.Get()
				return lifecycleClient.Heartbeat(ctx, currentCfg.AgentVersion, currentCfg.ConfigVersion)
			},
		})
	}

	if activationMgr != nil {
		if activationMgr.IsActivated() {
			rt.Register(agentruntime.Task{
				Name:     "activation_heartbeat",
				Interval: 5 * time.Minute,
				Fn: func(ctx context.Context) error {
					return activationMgr.SendHeartbeat(ctx)
				},
			})
		} else if cfg.ActivationMode == "manual" {
			rt.Register(agentruntime.Task{
				Name:     "activation_status_poll",
				Interval: 30 * time.Second,
				Fn: func(ctx context.Context) error {
					return activationMgr.CheckStatus(ctx)
				},
			})
		}
	}

	if localStore != nil {
		rt.Register(agentruntime.Task{
			Name:     "store_vacuum",
			Interval: 24 * time.Hour,
			Fn: func(ctx context.Context) error {
				return localStore.Vacuum()
			},
		})
	}

	rt.Start(ctx)

	// Step 9: Connect to session gateway (only when platform reachable)
	if deviceToken != "" && platformReachable {
		tokenProvider := lifecycle.NewClient(cfg.PlatformURL, deviceID, deviceToken)
		wsClient := session.NewWSClient(cfg.WSURL, deviceID, tenantID, tokenProvider, registry, auditClient, policyEngine, diagnosisEngine)
		wsClient.Start(ctx)
	}

	if !platformReachable {
		slog.Info("[boot] Bootstrap complete. Agent is running in OFFLINE mode (read-only only).")
	} else {
		slog.Info("[boot] Bootstrap complete. Agent is running.", "runtime_tasks", rt.TaskCount())
	}
	return nil
}

func (b *Bootstrapper) Shutdown(ctx context.Context) {
	slog.Info("[boot] Shutting down...")
	if b.runtime != nil {
		b.runtime.Stop()
	}
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

type remoteModelConfig struct {
	Provider  string `json:"provider"`
	BaseURL   string `json:"base_url"`
	ModelName string `json:"model_name"`
	APIKey    string `json:"api_key"`
}

var providerFactory = map[string]func(router.ProviderConfig) router.Provider{
	"openai":     func(c router.ProviderConfig) router.Provider { return providers.NewOpenAIProvider(c) },
	"deepseek":   func(c router.ProviderConfig) router.Provider { return providers.NewDeepSeekProvider(c) },
	"anthropic":  func(c router.ProviderConfig) router.Provider { return providers.NewAnthropicProvider(c) },
	"gemini":     func(c router.ProviderConfig) router.Provider { return providers.NewGeminiProvider(c) },
	"openrouter": func(c router.ProviderConfig) router.Provider { return providers.NewOpenRouterProvider(c) },
	"ollama":     func(c router.ProviderConfig) router.Provider { return providers.NewOllamaProvider(c) },
}

func (b *Bootstrapper) initLLMRouter(remote *remoteModelConfig) *router.Router {
	llmRouter := router.NewRouter(90 * time.Second)

	// Priority 1: Remote model profile from platform (pushed via config API)
	if remote != nil {
		factory, ok := providerFactory[remote.Provider]
		if ok {
			p := factory(router.ProviderConfig{
				Name:    "remote-" + remote.Provider,
				BaseURL: remote.BaseURL,
				APIKey:  remote.APIKey,
				Model:   remote.ModelName,
			})
			llmRouter.RegisterProvider(p)
			llmRouter.SetPrimary("remote-" + remote.Provider)
			slog.Info("[boot] Registered remote model provider (from platform)", "provider", remote.Provider, "model", remote.ModelName)
		} else {
			slog.Warn("[boot] Unknown remote model provider, falling back to env config", "provider", remote.Provider)
		}
	}

	// Priority 2: Environment variable providers
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

	// Priority 3: Ollama as local fallback
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
