package bootstrap

import (
	"github.com/alibaba/UnifiedModel/internal/graphstore"
	"github.com/alibaba/UnifiedModel/internal/workspace"
	"github.com/alibaba/UnifiedModel/pkg/contract"
)

// builtinPersistence is how the built-in providers persist data, by name. It is
// the fallback for a store that does not implement contract.PersistentProvider,
// so these providers behave exactly as they did before the interface existed.
//
// local.ladybug does not restore derived indexes at startup. That is how it has
// always behaved; it is kept as is here.
var builtinPersistence = map[string]contract.PersistenceMode{
	graphstore.ProviderTypeFileMemory: {PersistWorkspaceMetadata: true, RestoreDerivedIndexes: true},
	graphstore.ProviderTypeLadybug:    {PersistWorkspaceMetadata: true},
	graphstore.ProviderTypeNeo4j:      {PersistWorkspaceMetadata: true, RestoreDerivedIndexes: true},
	graphstore.ProviderTypeGremlin:    {PersistWorkspaceMetadata: true, RestoreDerivedIndexes: true},
}

// persistenceModeOf returns how graph persists data. A store that declares it
// through contract.PersistentProvider wins; otherwise the built-in table is
// used by provider name. A store that is neither is treated as memory only.
func persistenceModeOf(graph contract.GraphStore, providerType string) contract.PersistenceMode {
	if p, ok := graph.(contract.PersistentProvider); ok {
		return p.PersistenceMode()
	}
	return builtinPersistence[providerType]
}

// discoveryOf returns the workspace discovery to use. A store that declares its
// own persistence mode owns its discovery, including having none: it must not
// fall back to a built-in provider's directory layout just because it was
// registered under the same name. Only a store without a declaration uses the
// built-in discovery for its name.
func discoveryOf(graph contract.GraphStore, mode contract.PersistenceMode, providerType string) workspace.Discovery {
	if _, declared := graph.(contract.PersistentProvider); declared {
		if mode.DiscoverWorkspaces == nil {
			return nil
		}
		return mode.DiscoverWorkspaces
	}
	return workspace.BuiltinDiscovery(providerType)
}
