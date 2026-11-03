package embed_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alibaba/UnifiedModel/pkg/contract"
	"github.com/alibaba/UnifiedModel/pkg/embed"
	"github.com/alibaba/UnifiedModel/pkg/graphstore"
	"github.com/alibaba/UnifiedModel/pkg/model"
)

func newRuntime(t *testing.T) *embed.Runtime {
	t.Helper()
	rt, err := embed.New(t.TempDir(), graphstore.ProviderConfig{Type: "memory"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rt.Close() })
	return rt
}

// Everything a host needs is reachable through public interfaces, with no
// internal type in sight: a model written through the Go API is read back
// through the Query service, and so is an entity written through Entities.
func TestRuntimeEndToEndInProcess(t *testing.T) {
	ctx := context.Background()
	rt := newRuntime(t)

	if _, err := rt.Workspace.CreateWorkspace(ctx, model.CreateWorkspaceRequest{ID: "demo"}); err != nil {
		t.Fatal(err)
	}
	put, err := rt.UModel.PutElements(ctx, model.UModelElementBatch{
		Workspace: "demo",
		Elements: []model.UModelElement{{
			Kind: "entity_set", Domain: "apm", Name: "apm.service", Version: "v0.1.0",
			Spec: map[string]any{
				"fields":             []any{map[string]any{"name": "id", "type": "string"}},
				"primary_key_fields": []any{"id"},
				"id_generator":       "id",
			},
		}},
	})
	if err != nil || put.Accepted != 1 {
		t.Fatalf("put umodel: %+v err=%v", put, err)
	}

	models, err := rt.Query.Execute(ctx, "demo", model.QueryRequest{Query: ".umodel with(kind='entity_set') | limit 10"})
	if err != nil {
		t.Fatal(err)
	}
	if len(models.Rows) != 1 {
		t.Fatalf("expected the entity_set written through the Go API, got %d rows: %+v", len(models.Rows), models.Rows)
	}

	wrote, err := rt.Entities.WriteEntities(ctx, "demo", model.EntityWriteBatch{
		Workspace: "demo",
		Entities: []model.EntityPayload{{
			"__domain__": "apm", "__entity_type__": "apm.service", "__entity_id__": "177627f91af678a9b03e993f1a91917f",
			"__method__": "Update", "__first_observed_time__": int64(1), "__last_observed_time__": int64(2),
			"display_name": "checkout",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if wrote.Failed != 0 {
		t.Fatalf("write entity: %+v", wrote)
	}
	ents, err := rt.Query.Execute(ctx, "demo", model.QueryRequest{Query: ".entity with(domain='apm', name='apm.service') | limit 10"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ents.Rows) != 1 {
		t.Fatalf("expected the entity written through the Go API, got %d rows: %+v", len(ents.Rows), ents.Rows)
	}
}

// A GraphStore of the host's own, registered through the public registry, is
// the one the runtime uses.
func TestRuntimeUsesRegisteredProvider(t *testing.T) {
	var built int
	graphstore.RegisterProvider("test.embed.custom", func(graphstore.ProviderConfig) (contract.GraphStore, error) {
		built++
		// Reuse a real store so the runtime works; what matters is that this
		// factory, not a built-in one, is what New asked for.
		return graphstore.NewProvider(graphstore.ProviderConfig{Type: "memory"})
	})
	rt, err := embed.New(t.TempDir(), graphstore.ProviderConfig{Type: "test.embed.custom"})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	if built != 1 {
		t.Errorf("custom factory called %d times, want 1", built)
	}
	if rt.GraphStore == nil {
		t.Error("GraphStore must be exposed")
	}
}

func TestNewFailsForUnknownProvider(t *testing.T) {
	if _, err := embed.New(t.TempDir(), graphstore.ProviderConfig{Type: "test.embed.nope"}); err == nil {
		t.Fatal("an unregistered provider must be an error, not a half-built runtime")
	}
}

// The HTTP surface stays available for debugging and answers from the same
// services the functions use.
func TestRuntimeHandlerServesHealth(t *testing.T) {
	rt := newRuntime(t)
	srv := httptest.NewServer(rt.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/healthz = %d", resp.StatusCode)
	}
}
