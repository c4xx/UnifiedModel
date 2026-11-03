package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alibaba/UnifiedModel/internal/graphstore"
	"github.com/alibaba/UnifiedModel/pkg/contract"
	pubgraphstore "github.com/alibaba/UnifiedModel/pkg/graphstore"
	"github.com/alibaba/UnifiedModel/pkg/model"
)

// A store that does not declare a persistence mode is memory only, except for
// the built-in providers, which keep the behavior they had before the interface.
func TestBuiltinPersistenceUnchanged(t *testing.T) {
	cases := map[string]contract.PersistenceMode{
		graphstore.ProviderTypeMemory:     {},
		graphstore.ProviderTypeFileMemory: {PersistWorkspaceMetadata: true, RestoreDerivedIndexes: true},
		graphstore.ProviderTypeLadybug:    {PersistWorkspaceMetadata: true},
		graphstore.ProviderTypeNeo4j:      {PersistWorkspaceMetadata: true, RestoreDerivedIndexes: true},
		graphstore.ProviderTypeGremlin:    {PersistWorkspaceMetadata: true, RestoreDerivedIndexes: true},
		"some.unknown.provider":           {},
	}
	for name, want := range cases {
		// These stores do not implement PersistentProvider, so the table decides.
		got := persistenceModeOf(graphstore.NewMemoryStore(), name)
		if got.PersistWorkspaceMetadata != want.PersistWorkspaceMetadata || got.RestoreDerivedIndexes != want.RestoreDerivedIndexes {
			t.Errorf("%s: mode = %+v, want %+v", name, got, want)
		}
	}
}

// persistentStore is an external-style provider: a GraphStore the engine knows
// nothing about by name, which declares how it persists data.
type persistentStore struct {
	*graphstore.MemoryStore
	mode contract.PersistenceMode
}

func (p *persistentStore) PersistenceMode() contract.PersistenceMode { return p.mode }

func register(t *testing.T, name string, mode contract.PersistenceMode) {
	t.Helper()
	pubgraphstore.RegisterProvider(name, func(pubgraphstore.ProviderConfig) (contract.GraphStore, error) {
		return &persistentStore{MemoryStore: graphstore.NewMemoryStore(), mode: mode}, nil
	})
}

// An external provider registered through the public package can persist
// workspace metadata, which a provider unknown to the engine could not before.
func TestExternalProviderPersistsWorkspaceMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	name := "test.external.persist"
	register(t, name, contract.PersistenceMode{PersistWorkspaceMetadata: true})

	first, err := NewAppWithGraphStore(root, pubgraphstore.ProviderConfig{Type: name, DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Workspace.CreateWorkspace(ctx, model.CreateWorkspaceRequest{ID: "demo"}); err != nil {
		t.Fatal(err)
	}
	_ = first.Close()

	second, err := NewAppWithGraphStore(root, pubgraphstore.ProviderConfig{Type: name, DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })
	page, err := second.Workspace.ListWorkspaces(ctx, model.WorkspaceListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "demo" {
		t.Fatalf("the workspace must survive a restart, got %+v", page.Items)
	}
}

// The control: the same store without the declaration is treated as memory only,
// so the workspace is gone after a restart. This is the behavior external
// providers had before.
func TestUndeclaredProviderIsMemoryOnly(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	name := "test.external.undeclared"
	pubgraphstore.RegisterProvider(name, func(pubgraphstore.ProviderConfig) (contract.GraphStore, error) {
		return graphstore.NewMemoryStore(), nil
	})

	first, err := NewAppWithGraphStore(root, pubgraphstore.ProviderConfig{Type: name, DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Workspace.CreateWorkspace(ctx, model.CreateWorkspaceRequest{ID: "demo"}); err != nil {
		t.Fatal(err)
	}
	_ = first.Close()
	if _, err := os.Stat(filepath.Join(root, "workspaces.json")); !os.IsNotExist(err) {
		t.Errorf("an undeclared provider must not write workspaces.json, stat err = %v", err)
	}
}

// DiscoverWorkspaces lets a provider recover workspaces that exist in its store
// but are missing from the metadata file, using its own layout.
func TestExternalProviderDiscoversWorkspaces(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	name := "test.external.discover"
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	var gotRoot string
	register(t, name, contract.PersistenceMode{
		PersistWorkspaceMetadata: true,
		DiscoverWorkspaces: func(dataRoot string) ([]contract.WorkspaceDiscovery, error) {
			gotRoot = dataRoot
			return []contract.WorkspaceDiscovery{{ID: "from-store", ModTime: when}}, nil
		},
	})
	app, err := NewAppWithGraphStore(root, pubgraphstore.ProviderConfig{Type: name, DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })
	if gotRoot != root {
		t.Errorf("discovery got dataRoot %q, want %q", gotRoot, root)
	}
	page, _ := app.Workspace.ListWorkspaces(ctx, model.WorkspaceListRequest{})
	if len(page.Items) != 1 || page.Items[0].ID != "from-store" || !page.Items[0].CreatedAt.Equal(when) {
		t.Fatalf("discovered workspace missing or wrong time: %+v", page.Items)
	}
}

// A provider that declares a mode owns its discovery. One registered under a
// built-in name must not inherit that built-in provider's directory layout.
func TestDeclaredProviderDoesNotInheritBuiltinDiscovery(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	// A directory that looks like a ladybug workspace. If the built-in ladybug
	// discovery leaked into a declared provider, "trap" would show up.
	if err := os.MkdirAll(filepath.Join(root, "instances", "trap", "storage", "graph", "local", "ladybug"), 0o755); err != nil {
		t.Fatal(err)
	}
	name := graphstore.ProviderTypeLadybug // same name as a built-in, but replaced
	t.Cleanup(func() {
		// Put the real factory back for other tests in this package.
		graphstore.RegisterProvider(graphstore.ProviderTypeLadybug, func(c graphstore.ProviderConfig) (contract.GraphStore, error) {
			return graphstore.NewMemoryStore(), nil
		})
	})
	register(t, name, contract.PersistenceMode{PersistWorkspaceMetadata: true})

	app, err := NewAppWithGraphStore(root, pubgraphstore.ProviderConfig{Type: name, DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Close() })
	page, _ := app.Workspace.ListWorkspaces(ctx, model.WorkspaceListRequest{})
	if len(page.Items) != 0 {
		t.Fatalf("a declared provider with no discovery must find nothing, got %+v", page.Items)
	}
}

// A provider that asks for derived indexes to be restored gets them restored
// from the entities already in its store when the app starts.
func TestExternalProviderRestoresDerivedIndexes(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	name := "test.external.restore"
	store := &persistentStore{MemoryStore: graphstore.NewMemoryStore(), mode: contract.PersistenceMode{PersistWorkspaceMetadata: true, RestoreDerivedIndexes: true}}
	pubgraphstore.RegisterProvider(name, func(pubgraphstore.ProviderConfig) (contract.GraphStore, error) { return store, nil })

	// Create the workspace first (metadata only), then reopen over the same store.
	first, err := NewAppWithGraphStore(root, pubgraphstore.ProviderConfig{Type: name, DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Workspace.CreateWorkspace(ctx, model.CreateWorkspaceRequest{ID: "demo"}); err != nil {
		t.Fatal(err)
	}
	_ = first.Close()
	second, err := NewAppWithGraphStore(root, pubgraphstore.ProviderConfig{Type: name, DataRoot: root})
	if err != nil {
		t.Fatalf("restoring derived indexes at startup failed: %v", err)
	}
	_ = second.Close()
}
