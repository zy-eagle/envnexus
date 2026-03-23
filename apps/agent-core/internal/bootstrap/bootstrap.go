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
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/llm/providers"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/llm/router"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/policy"
	"github.com/zy-eagle/envnexus/apps/agent-core/internal/session"
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
	registry.Register(service.NewRestartTool())
	registry.Register(cache.NewRebuildTool())
	log.Printf("[boot] Registered %d tools\n", registry.Count())

	// Step 4: Initialize LLM router
	llmRouter := b.initLLMRouter()

	// Step 5: Initialize engines
	policyEngine := policy.NewEngine()
	if deviceToken != "" {
		policyEngine.SetPlatformClient(policy.NewPlatformClient(cfg.PlatformURL, deviceToken))
	}
	auditClient := audit.NewClient(cfg.PlatformURL, deviceID, deviceToken)
	diagnosisEngine := diagnosis.NewEngine(registry, llmRouter)

	go auditClient.StartFlushLoop(ctx, 15*time.Second)

	// Step 6: Start local API
	localServer := api.NewLocalServer(17700, b.identityManager, policyEngine, diagnosisEngine)
	if err := localServer.Start(); err != nil {
		log.Printf("[boot] Failed to start local API: %v\n", err)
	}

	// Step 7: Start heartbeat loop
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

	// Step 8: Start governance engine
	governanceEngine := governance.NewEngine()
	governanceEngine.Start(ctx)

	// Step 9: Connect to session gateway
	if deviceToken != "" {
		wsClient := session.NewWSClient(cfg.WSURL, deviceID, tenantID, deviceToken, registry, auditClient, policyEngine, diagnosisEngine)
		wsClient.Start(ctx)
	}

	log.Println("[boot] Bootstrap complete. Agent is running.")
	return nil
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
		log.Println("[boot] Registered OpenAI provider")
	}

	if apiKey := os.Getenv("ENX_OPENROUTER_API_KEY"); apiKey != "" {
		p := providers.NewOpenRouterProvider(router.ProviderConfig{
			Name:    "openrouter",
			BaseURL: envOrDefault("ENX_OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
			APIKey:  apiKey,
			Model:   envOrDefault("ENX_OPENROUTER_MODEL", ""),
		})
		llmRouter.RegisterProvider(p)
		log.Println("[boot] Registered OpenRouter provider")
	}

	if apiKey := os.Getenv("ENX_DEEPSEEK_API_KEY"); apiKey != "" {
		p := providers.NewDeepSeekProvider(router.ProviderConfig{
			Name:    "deepseek",
			BaseURL: envOrDefault("ENX_DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
			APIKey:  apiKey,
			Model:   envOrDefault("ENX_DEEPSEEK_MODEL", "deepseek-chat"),
		})
		llmRouter.RegisterProvider(p)
		log.Println("[boot] Registered DeepSeek provider")
	}

	if apiKey := os.Getenv("ENX_ANTHROPIC_API_KEY"); apiKey != "" {
		p := providers.NewAnthropicProvider(router.ProviderConfig{
			Name:    "anthropic",
			BaseURL: envOrDefault("ENX_ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
			APIKey:  apiKey,
			Model:   envOrDefault("ENX_ANTHROPIC_MODEL", "claude-sonnet-4-20250514"),
		})
		llmRouter.RegisterProvider(p)
		log.Println("[boot] Registered Anthropic provider")
	}

	if apiKey := os.Getenv("ENX_GEMINI_API_KEY"); apiKey != "" {
		p := providers.NewGeminiProvider(router.ProviderConfig{
			Name:    "gemini",
			BaseURL: envOrDefault("ENX_GEMINI_BASE_URL", "https://generativelanguage.googleapis.com"),
			APIKey:  apiKey,
			Model:   envOrDefault("ENX_GEMINI_MODEL", "gemini-2.0-flash"),
		})
		llmRouter.RegisterProvider(p)
		log.Println("[boot] Registered Gemini provider")
	}

	ollamaURL := envOrDefault("ENX_OLLAMA_URL", "http://localhost:11434")
	p := providers.NewOllamaProvider(router.ProviderConfig{
		Name:    "ollama",
		BaseURL: ollamaURL,
		Model:   envOrDefault("ENX_OLLAMA_MODEL", "llama3.2"),
	})
	llmRouter.RegisterProvider(p)
	log.Println("[boot] Registered Ollama provider (fallback)")

	if primary := os.Getenv("ENX_LLM_PRIMARY"); primary != "" {
		llmRouter.SetPrimary(primary)
	}

	log.Printf("[boot] LLM router initialized with providers: %v", llmRouter.ListProviders())
	return llmRouter
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
