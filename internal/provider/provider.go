// Package provider defines custom (user-supplied URL) FFmpeg providers and
// the resolver that pins them. OneMoreFFmpeg no longer ships built-in
// providers: every download source is an explicit URL + SHA-256 registered by
// the user via `provider add`.
package provider

import (
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const IDCustom = "custom"

// Request selects a custom provider release.
type Request struct {
	Provider     string
	Channel      string
	Target       string
	Variant      string
	Linkage      string
	ReleaseTag   string
	ReleaseLine  string
	AllowNonfree bool
}

// Provider exposes one provider's metadata. Custom providers are the only
// kind; the registry remains as the offline listing surface used by
// `provider list`.
type Provider interface {
	ID() string
	Summary() model.ProviderSummary
}

// Registry holds registered providers. Built-in providers were removed; the
// registry is empty unless callers add providers explicitly.
type Registry struct {
	order     []string
	providers map[string]Provider
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{providers: map[string]Provider{}}
}

// List returns provider summaries in registry order.
func (r *Registry) List() []model.ProviderSummary {
	if r == nil {
		return []model.ProviderSummary{}
	}
	out := make([]model.ProviderSummary, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.providers[id].Summary())
	}
	return out
}

// Get returns a registered provider by id.
func (r *Registry) Get(id string) (Provider, error) {
	if r != nil {
		if item, ok := r.providers[id]; ok {
			return item, nil
		}
	}
	return nil, apperrors.New(apperrors.E3001, "provider id is not registered").
		WithDetails(map[string]any{"provider": id})
}

// Add registers a provider (used for custom entries if needed in future).
func (r *Registry) Add(item Provider) {
	if r == nil || r.providers == nil {
		return
	}
	if _, exists := r.providers[item.ID()]; !exists {
		r.order = append(r.order, item.ID())
	}
	r.providers[item.ID()] = item
}
