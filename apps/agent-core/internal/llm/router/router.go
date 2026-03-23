package router

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

type CompletionResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	PromptTokens int    `json:"prompt_tokens"`
	CompTokens   int    `json:"completion_tokens"`
	DurationMs   int64  `json:"duration_ms"`
}

type Provider interface {
	Name() string
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
	IsAvailable() bool
}

type ProviderConfig struct {
	Name        string
	BaseURL     string
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

type Router struct {
	mu        sync.RWMutex
	providers []Provider
	primary   string
	fallbacks []string
	timeout   time.Duration
}

func NewRouter(timeout time.Duration) *Router {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return &Router{
		providers: make([]Provider, 0),
		timeout:   timeout,
	}
}

func (r *Router) RegisterProvider(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = append(r.providers, p)
	if r.primary == "" {
		r.primary = p.Name()
	}
}

func (r *Router) SetPrimary(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.primary = name
}

func (r *Router) SetFallbacks(names []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallbacks = names
}

func (r *Router) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	r.mu.RLock()
	primary := r.primary
	fallbacks := r.fallbacks
	providers := r.providers
	r.mu.RUnlock()

	order := r.buildProviderOrder(primary, fallbacks, providers)
	if len(order) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}

	var lastErr error
	for _, p := range order {
		if !p.IsAvailable() {
			log.Printf("[llm/router] Provider %s unavailable, skipping", p.Name())
			continue
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
		start := time.Now()
		resp, err := p.Complete(timeoutCtx, req)
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("provider %s: %w", p.Name(), err)
			log.Printf("[llm/router] Provider %s failed in %dms: %v", p.Name(), time.Since(start).Milliseconds(), err)
			continue
		}

		resp.DurationMs = time.Since(start).Milliseconds()
		resp.Provider = p.Name()
		log.Printf("[llm/router] Provider %s completed in %dms (tokens: %d+%d)",
			p.Name(), resp.DurationMs, resp.PromptTokens, resp.CompTokens)
		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
	}
	return nil, fmt.Errorf("no available providers")
}

func (r *Router) buildProviderOrder(primary string, fallbacks []string, providers []Provider) []Provider {
	byName := make(map[string]Provider)
	for _, p := range providers {
		byName[p.Name()] = p
	}

	var order []Provider
	if p, ok := byName[primary]; ok {
		order = append(order, p)
	}
	for _, name := range fallbacks {
		if p, ok := byName[name]; ok && name != primary {
			order = append(order, p)
		}
	}
	for _, p := range providers {
		found := false
		for _, o := range order {
			if o.Name() == p.Name() {
				found = true
				break
			}
		}
		if !found {
			order = append(order, p)
		}
	}
	return order
}

func (r *Router) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = p.Name()
	}
	return names
}
