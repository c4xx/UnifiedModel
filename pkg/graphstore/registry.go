// Package graphstore is the public registry of GraphStore providers.
//
// A provider is a contract.GraphStore implementation selected by name, for
// example "memory" or "neo4j". The registry lives in a public package so that
// code outside this module can register its own provider before the service
// starts:
//
//	func init() {
//		graphstore.RegisterProvider("my.store", func(cfg graphstore.ProviderConfig) (contract.GraphStore, error) {
//			return newMyStore(cfg)
//		})
//	}
//
// The registry holds factories only. Built-in providers stay under
// internal/graphstore and register themselves here from their own init
// functions, so importing this package pulls in no provider implementation.
package graphstore

import (
	"fmt"
	"sort"
	"sync"

	"github.com/alibaba/UnifiedModel/pkg/contract"
)

// ProviderConfig is what a ProviderFactory receives.
type ProviderConfig struct {
	// Type is the registered provider name. Empty selects DefaultProviderType.
	Type string
	// DataRoot is the directory a provider may use for local state.
	DataRoot string
	// Options carries provider-specific settings, such as connection URIs.
	Options map[string]string
}

// ProviderFactory builds a GraphStore from a config.
type ProviderFactory func(config ProviderConfig) (contract.GraphStore, error)

var (
	mu       sync.RWMutex
	registry = map[string]ProviderFactory{}
	// defaultType is the provider used when ProviderConfig.Type is empty.
	defaultType string
)

// RegisterProvider registers factory under providerType.
//
// It panics on an empty name or a nil factory, the same way database/sql.Register
// does, because both are programming errors that would otherwise surface later as
// a confusing "not registered" failure. Registering a name twice replaces the
// earlier factory; this keeps tests that swap in a fake provider working.
func RegisterProvider(providerType string, factory ProviderFactory) {
	if providerType == "" {
		panic("graphstore: RegisterProvider called with an empty provider type")
	}
	if factory == nil {
		panic("graphstore: RegisterProvider called with a nil factory for " + providerType)
	}
	mu.Lock()
	defer mu.Unlock()
	registry[providerType] = factory
}

// SetDefaultProvider sets the provider used when ProviderConfig.Type is empty.
func SetDefaultProvider(providerType string) {
	mu.Lock()
	defer mu.Unlock()
	defaultType = providerType
}

// DefaultProvider returns the provider used when ProviderConfig.Type is empty.
func DefaultProvider() string {
	mu.RLock()
	defer mu.RUnlock()
	return defaultType
}

// NewProvider builds the provider named by config.Type, or the default one when
// Type is empty.
func NewProvider(config ProviderConfig) (contract.GraphStore, error) {
	mu.RLock()
	providerType := config.Type
	if providerType == "" {
		providerType = defaultType
	}
	factory := registry[providerType]
	mu.RUnlock()
	if factory == nil {
		return nil, fmt.Errorf("graphstore provider %q is not registered", providerType)
	}
	return factory(config)
}

// RegisteredProviders returns the registered provider names in sorted order.
func RegisteredProviders() []string {
	mu.RLock()
	defer mu.RUnlock()
	providers := make([]string, 0, len(registry))
	for providerType := range registry {
		providers = append(providers, providerType)
	}
	sort.Strings(providers)
	return providers
}
