// Package embed runs UModel inside another Go program.
//
// The UModel server is a thin HTTP layer over a set of services. A program
// that already is a Go service can call those services as functions instead of
// starting a second process and talking to it over HTTP. Runtime gives access
// to them through the interfaces in pkg/contract, so the caller never sees a
// type from an internal package.
//
//	rt, err := embed.New("/data", graphstore.ProviderConfig{Type: "my.store"})
//	if err != nil { ... }
//	defer rt.Close()
//	res, err := rt.Query.Execute(ctx, "prod", model.QueryRequest{Query: ".umodel | limit 5"})
//
// A GraphStore of your own is plugged in through pkg/graphstore.RegisterProvider
// before New is called; see contract.PersistentProvider for how it declares what
// survives a restart.
package embed

import (
	"net/http"

	"github.com/alibaba/UnifiedModel/internal/bootstrap"
	"github.com/alibaba/UnifiedModel/pkg/contract"
	"github.com/alibaba/UnifiedModel/pkg/graphstore"
)

// Runtime is a running UModel instance.
type Runtime struct {
	// Workspace manages workspace metadata.
	Workspace contract.WorkspaceManager
	// UModel validates, imports and writes model elements.
	UModel contract.UModelService
	// Entities writes, expires and deletes runtime entities and relations.
	Entities contract.EntityWriteService
	// Query is the only read path: .umodel, .entity, .entity_set and .topo.
	Query contract.QueryService
	// Search is keyword, vector and hybrid search over models and entities.
	Search contract.SearchService
	// Agent is the agent-facing surface: discovery, tools and resources.
	Agent contract.AgentGateway
	// GraphStore is the store behind all of the above. It is exposed so a host
	// can check its health and capabilities; reads and writes should go through
	// the services, which validate against the model.
	GraphStore contract.GraphStore

	app *bootstrap.App
}

// Option configures New.
type Option func(*options)

type options struct {
	importRoot string
}

// WithImportRoot confines UModel imports that name a path on disk to that
// directory. The empty default confines them to the process working directory;
// "/" allows any path.
func WithImportRoot(root string) Option {
	return func(o *options) { o.importRoot = root }
}

// New starts a Runtime. dataRoot is the directory for local state, and config
// selects the GraphStore: Type names a provider registered with
// pkg/graphstore, and an empty Type selects the default provider.
func New(dataRoot string, config graphstore.ProviderConfig, opts ...Option) (*Runtime, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	app, err := bootstrap.NewAppWithGraphStore(dataRoot, config, bootstrap.WithImportRoot(o.importRoot))
	if err != nil {
		return nil, err
	}
	return &Runtime{
		Workspace:  app.Workspace,
		UModel:     app.UModel,
		Entities:   app.EntityStore,
		Query:      app.Query,
		Search:     app.Search,
		Agent:      app.AgentGateway,
		GraphStore: app.GraphStore,
		app:        app,
	}, nil
}

// Handler returns the UModel REST API and MCP-compatible endpoints as an
// http.Handler. A host that wants UModel's own web UI or tooling for debugging
// can mount it; a host that only calls functions never needs it.
func (r *Runtime) Handler() http.Handler { return r.app.Handler() }

// HandlerWithUI is Handler plus the built web UI served from uiDir.
func (r *Runtime) HandlerWithUI(uiDir string) http.Handler { return r.app.HandlerWithUI(uiDir) }

// Close releases the GraphStore.
func (r *Runtime) Close() error { return r.app.Close() }
