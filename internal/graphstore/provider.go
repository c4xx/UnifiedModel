package graphstore

import (
	"github.com/alibaba/UnifiedModel/pkg/contract"
	pubgraphstore "github.com/alibaba/UnifiedModel/pkg/graphstore"
)

const (
	ProviderTypeMemory     = "memory"
	ProviderTypeFileMemory = "file.memory"
	ProviderTypeLadybug    = "local.ladybug"
	ProviderTypeNeo4j      = "neo4j"
	ProviderTypeGremlin    = "gremlin"
	DefaultProviderType    = ProviderTypeLadybug
)

// The registry itself is public (pkg/graphstore) so providers outside this
// module can register. These aliases and forwarding functions keep the
// internal API unchanged: built-in providers and tests keep calling
// graphstore.RegisterProvider and friends.
type (
	ProviderConfig  = pubgraphstore.ProviderConfig
	ProviderFactory = pubgraphstore.ProviderFactory
)

func RegisterProvider(providerType string, factory ProviderFactory) {
	pubgraphstore.RegisterProvider(providerType, factory)
}

func NewProvider(config ProviderConfig) (contract.GraphStore, error) {
	return pubgraphstore.NewProvider(config)
}

func RegisteredProviders() []string {
	return pubgraphstore.RegisteredProviders()
}

func init() {
	pubgraphstore.SetDefaultProvider(DefaultProviderType)
	RegisterProvider(ProviderTypeMemory, func(config ProviderConfig) (contract.GraphStore, error) {
		return NewMemoryStore(), nil
	})
	RegisterProvider(ProviderTypeFileMemory, func(config ProviderConfig) (contract.GraphStore, error) {
		return NewFileMemoryStore(config)
	})
}
