package proxy

import (
	"fmt"
	"sync"

	"github.com/youorg/ai-proxy-platform/backend/internal/proxy/providers"
)

// Registry maps model IDs to their provider.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]providers.Provider // key: model_id
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]providers.Provider)}
}

// Register registers all models for a provider.
func (r *Registry) Register(p providers.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, modelID := range p.ModelIDs() {
		r.providers[modelID] = p
	}
}

// Get returns the provider for a model ID.
func (r *Registry) Get(modelID string) (providers.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[modelID]
	if !ok {
		return nil, fmt.Errorf("model %q not found", modelID)
	}
	return p, nil
}

// ModelIDs returns all registered model IDs.
func (r *Registry) ModelIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	return ids
}
