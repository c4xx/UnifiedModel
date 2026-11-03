package bootstrap

import (
	"github.com/alibaba/UnifiedModel/internal/agentgateway"
	"github.com/alibaba/UnifiedModel/internal/entitystore"
	"github.com/alibaba/UnifiedModel/internal/query"
	"github.com/alibaba/UnifiedModel/internal/search"
	"github.com/alibaba/UnifiedModel/internal/umodel"
	"github.com/alibaba/UnifiedModel/internal/workspace"
	"github.com/alibaba/UnifiedModel/pkg/contract"
)

// pkg/embed hands these services to code outside the module as the public
// interfaces. Assert it at compile time so a signature change in a service
// breaks the build here instead of surfacing in a host program.
var (
	_ contract.WorkspaceManager   = (*workspace.Service)(nil)
	_ contract.UModelService      = (*umodel.Service)(nil)
	_ contract.EntityWriteService = (*entitystore.Service)(nil)
	_ contract.QueryService       = (*query.Service)(nil)
	_ contract.SearchService      = (*search.Service)(nil)
	_ contract.AgentGateway       = (*agentgateway.Service)(nil)
)
