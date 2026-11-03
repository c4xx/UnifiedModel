package contract

import (
	"context"
	"time"

	"github.com/alibaba/UnifiedModel/pkg/model"
)

type WorkspaceManager interface {
	CreateWorkspace(ctx context.Context, req model.CreateWorkspaceRequest) (model.WorkspaceMetadata, error)
	GetWorkspace(ctx context.Context, id string) (model.WorkspaceMetadata, error)
	ListWorkspaces(ctx context.Context, req model.WorkspaceListRequest) (model.Page[model.WorkspaceMetadata], error)
	UpdateWorkspace(ctx context.Context, id string, req model.UpdateWorkspaceRequest) (model.WorkspaceMetadata, error)
	DeleteWorkspace(ctx context.Context, id string) (model.WorkspaceMetadata, error)
}

type WorkspaceMetadataReader interface {
	GetWorkspace(ctx context.Context, id string) (model.WorkspaceMetadata, error)
	ListWorkspaces(ctx context.Context, req model.WorkspaceListRequest) (model.Page[model.WorkspaceMetadata], error)
}

type WorkspaceConfigSchemaRegistry interface {
	ValidateNamespace(ctx context.Context, namespace string, value map[string]any) error
}

type GraphStore interface {
	OpenWorkspace(ctx context.Context, workspace model.WorkspaceMetadata) error
	EnsureSchema(ctx context.Context, workspace string) error
	PutUModelElements(ctx context.Context, batch model.UModelElementBatch) (model.WriteResult, error)
	DeleteUModelElements(ctx context.Context, workspace string, ids []string) (model.WriteResult, error)
	GetUModelSnapshot(ctx context.Context, req model.UModelSnapshotRequest) (model.UModelSnapshot, error)
	WriteEntities(ctx context.Context, batch model.EntityWriteBatch) (model.WriteResult, error)
	WriteRelations(ctx context.Context, batch model.RelationWriteBatch) (model.WriteResult, error)
	QueryEntities(ctx context.Context, plan model.EntityQueryPlan) (model.QueryResult, error)
	QueryTopo(ctx context.Context, plan model.TopoQueryPlan) (model.QueryResult, error)
	Capabilities(ctx context.Context) (model.GraphStoreCapabilities, error)
	Health(ctx context.Context) (model.GraphStoreHealth, error)
}

// PersistenceMode describes how a GraphStore keeps data across restarts. The
// service uses it at startup to decide what to restore. The zero value means
// "in memory only", which is how the built-in memory provider behaves.
type PersistenceMode struct {
	// PersistWorkspaceMetadata keeps workspace metadata on disk under the data
	// root and reloads it at startup.
	PersistWorkspaceMetadata bool
	// RestoreDerivedIndexes rebuilds the search and schema indexes at startup
	// from the entities already in the store. Set it when the store holds data
	// that the in-memory indexes have not seen yet.
	RestoreDerivedIndexes bool
	// DiscoverWorkspaces optionally reports workspace IDs that already exist in
	// the store but are missing from the persisted metadata, so a lost or
	// partial metadata file does not hide them. dataRoot is the service data
	// directory. Return nil when there is nothing to discover. May be nil.
	DiscoverWorkspaces func(dataRoot string) ([]WorkspaceDiscovery, error)
}

// WorkspaceDiscovery is a workspace found in the store at startup.
type WorkspaceDiscovery struct {
	ID string
	// ModTime is used as the created/updated time of the recovered workspace.
	// The zero value means "now".
	ModTime time.Time
}

// PersistentProvider is an optional interface for a GraphStore. A store that
// does not implement it is treated as one of the built-in providers, selected
// by name, so existing providers keep their behavior unchanged.
type PersistentProvider interface {
	PersistenceMode() PersistenceMode
}

type UModelService interface {
	Import(ctx context.Context, workspace string, req model.UModelImportRequest) (model.UModelImportResult, error)
	Validate(ctx context.Context, workspace string, elements []model.UModelElement) (model.ValidationResult, error)
	PutElements(ctx context.Context, batch model.UModelElementBatch) (model.WriteResult, error)
	DeleteElements(ctx context.Context, workspace string, ids []string) (model.WriteResult, error)
	RebuildIndex(ctx context.Context, workspace string) error
}

type UModelSchemaResolver interface {
	ResolveEntitySet(ctx context.Context, ref model.EntityTypeRef) (model.EntitySetSchema, error)
	ResolveRelationType(ctx context.Context, ref model.RelationTypeRef) (model.RelationSchema, error)
	ValidateEntityPayload(ctx context.Context, payload model.EntityPayload) (model.ValidationResult, error)
	ValidateRelationPayload(ctx context.Context, payload model.RelationPayload) (model.ValidationResult, error)
	SnapshotVersion(ctx context.Context, workspace string) (model.SchemaVersion, error)
}

type EntityWriteService interface {
	WriteEntities(ctx context.Context, workspace string, batch model.EntityWriteBatch) (model.WriteResult, error)
	WriteRelations(ctx context.Context, workspace string, batch model.RelationWriteBatch) (model.WriteResult, error)
	ExpireEntities(ctx context.Context, workspace string, req model.ExpireRequest) (model.WriteResult, error)
	ExpireRelations(ctx context.Context, workspace string, req model.ExpireRequest) (model.WriteResult, error)
}

type QueryService interface {
	Execute(ctx context.Context, workspace string, req model.QueryRequest) (model.QueryResult, error)
	Explain(ctx context.Context, workspace string, req model.QueryRequest) (model.QueryExplain, error)
	Examples(ctx context.Context) ([]string, error)
}

// SearchService is the runtime-facing semantic search entrypoint for the
// `.umodel`, `.entity`, and `.runbook_set` query sources. It mirrors the SLS
// USearch SPL contract so the same `with(...)` invocation flows through both
// the open-source engine and the SLS backend.
type SearchService interface {
	Keyword(ctx context.Context, workspace string, req model.SearchRequest) (model.SearchResult, error)
	Vector(ctx context.Context, workspace string, req model.SearchRequest) (model.SearchResult, error)
	Hybrid(ctx context.Context, workspace string, req model.SearchRequest) (model.SearchResult, error)
	Capabilities(ctx context.Context) (model.SearchCapabilities, error)
	Health(ctx context.Context) (model.SearchHealth, error)
}

type AgentGateway interface {
	Discover(ctx context.Context, workspace string) (model.AgentDiscovery, error)
	Tools(ctx context.Context) ([]model.AgentTool, error)
	ReadResource(ctx context.Context, workspace string, req model.AgentResourceReadRequest) (model.AgentResourceReadResult, error)
	ExecuteTool(ctx context.Context, workspace string, req model.AgentToolCallRequest) (model.AgentToolCallResult, error)
}
