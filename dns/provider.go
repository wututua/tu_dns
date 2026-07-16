package dns

import (
	"context"
	"sort"
	"sync"
)

type Provider interface {
	Key() string
	Label() string
	ConfigFields() []ConfigField
	Configure(config map[string]string) error
	Check(ctx context.Context) error
	ListZones(ctx context.Context) ([]Zone, error)
	CreateRecord(ctx context.Context, zone Zone, input RecordInput) (Record, error)
	UpdateRecord(ctx context.Context, zone Zone, remoteID string, input RecordInput) (Record, error)
	DeleteRecord(ctx context.Context, zone Zone, remoteID string) error
}

type ConfigField struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Description string `json:"description,omitempty"`
}

type Zone struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
}

type RecordInput struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
	Line  string `json:"line"`
}

type Record struct {
	RemoteID string `json:"remote_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Line     string `json:"line"`
}

type Factory func() Provider

var registry = struct {
	sync.RWMutex
	items map[string]Factory
}{items: map[string]Factory{}}

func Register(key string, factory Factory) {
	registry.Lock()
	defer registry.Unlock()
	registry.items[key] = factory
}

func New(key string) (Provider, bool) {
	registry.RLock()
	defer registry.RUnlock()
	f, ok := registry.items[key]
	if !ok {
		return nil, false
	}
	return f(), true
}

func List() []map[string]interface{} {
	registry.RLock()
	defer registry.RUnlock()
	keys := make([]string, 0, len(registry.items))
	for k := range registry.items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]interface{}, 0, len(keys))
	for _, k := range keys {
		p := registry.items[k]()
		out = append(out, map[string]interface{}{
			"key":           p.Key(),
			"label":         p.Label(),
			"config_fields": p.ConfigFields(),
		})
	}
	return out
}
