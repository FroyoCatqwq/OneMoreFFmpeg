// Package provider defines the FFmpeg release providers and their registry.
// Resolution is intentionally not implemented yet: Phase 4 adds the GitHub
// client and per-provider asset matching on top of this registry.
package provider

import (
	"context"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const (
	IDBtbn          = "btbn"
	IDBnqdzjNonfree = "bnqdzj-nonfree"
	IDCustom        = "custom"
)

// Request selects one release/asset. Empty fields fall back to documented
// defaults once resolution is implemented.
type Request struct {
	Provider    string
	Channel     string // stable | daily
	Target      string // win64 | linux64
	Variant     string // gpl | lgpl | nonfree | nonfree-shared
	Linkage     string // static | shared
	ReleaseTag  string
	ReleaseLine string // master | n8.1 | n9.0 ...
}

// Provider resolves one pinned release for a provider id.
type Provider interface {
	ID() string
	Summary() model.ProviderSummary
	Resolve(ctx context.Context, req Request) (model.Release, error)
}

// Registry holds the built-in providers. It is immutable after construction
// and safe for concurrent reads.
type Registry struct {
	order     []string
	providers map[string]Provider
}

// NewRegistry returns the built-in provider registry. Listing is offline and
// keeps the documented order (btbn first, then bnqdzj-nonfree).
func NewRegistry() *Registry {
	registry := &Registry{providers: map[string]Provider{}}
	for _, item := range []Provider{
		summaryProvider{summary: btbnSummary()},
		summaryProvider{summary: bnqdzjSummary()},
	} {
		registry.order = append(registry.order, item.ID())
		registry.providers[item.ID()] = item
	}
	return registry
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

// Get returns a provider by id.
func (r *Registry) Get(id string) (Provider, error) {
	if r == nil {
		return nil, apperrors.New(apperrors.E3001, "provider registry is not configured")
	}
	item, ok := r.providers[id]
	if !ok {
		return nil, apperrors.New(apperrors.E3001, "provider id is not registered").WithDetails(map[string]any{"provider": id})
	}
	return item, nil
}

func btbnSummary() model.ProviderSummary {
	return model.ProviderSummary{
		ID:              IDBtbn,
		Repository:      "FFmpeg-Builds/FFmpeg-Builds",
		Channels:        []string{"stable", "daily"},
		Variants:        []string{"gpl", "lgpl", "gpl-shared", "lgpl-shared"},
		Linkages:        []string{"static", "shared"},
		Redistributable: true,
	}
}

func bnqdzjSummary() model.ProviderSummary {
	return model.ProviderSummary{
		ID:              IDBnqdzjNonfree,
		Repository:      "BnqDzj/FFmpeg-Builds-nonfree",
		Channels:        []string{"daily"},
		Variants:        []string{"nonfree", "nonfree-shared"},
		Linkages:        []string{"static", "shared"},
		Redistributable: false,
	}
}

// summaryProvider exposes metadata only; Resolve is wired in Phase 4.
type summaryProvider struct {
	summary model.ProviderSummary
}

func (p summaryProvider) ID() string {
	return p.summary.ID
}

func (p summaryProvider) Summary() model.ProviderSummary {
	return p.summary
}

func (p summaryProvider) Resolve(context.Context, Request) (model.Release, error) {
	return model.Release{}, apperrors.New(apperrors.E3007, "provider resolution is not implemented yet").WithDetails(map[string]any{"provider": p.summary.ID})
}
