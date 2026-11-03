package umodel

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/alibaba/UnifiedModel/internal/graphstore"
	"github.com/alibaba/UnifiedModel/pkg/model"
)

func importOne(t *testing.T, body string) model.UModelElement {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "e.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(graphstore.NewMemoryStore(), WithImportRoot(root))
	if _, err := svc.Import(ctx, "demo", model.UModelImportRequest{Path: root}); err != nil {
		t.Fatal(err)
	}
	snap, err := svc.graph.GetUModelSnapshot(ctx, model.UModelSnapshotRequest{Workspace: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Elements) != 1 {
		t.Fatalf("elements = %d: %+v", len(snap.Elements), snap.Elements)
	}
	return snap.Elements[0]
}

const spec = "spec:\n  fields:\n    - name: id\n      type: string\n  primary_key_fields: [id]\n  id_generator: id\n"

// The metadata block of an imported element reaches the store: the schema
// defines properties, tags, launch_stage and bilingual text, and an import used
// to keep only name and domain.
func TestImportKeepsMetadata(t *testing.T) {
	el := importOne(t, "kind: entity_set\nschema:\n  url: u\n  version: v0.1.0\nmetadata:\n"+
		"  name: devops.service\n  domain: devops\n"+
		"  display_name:\n    en_us: Service\n    zh_cn: 服务\n"+
		"  description:\n    en_us: A running service\n"+
		"  launch_stage: ga\n"+
		"  tags:\n    team: sre\n"+
		"  properties:\n    host.depends: src_on_dest\n    host.params: \"a, b\"\n"+spec)

	want := &model.ElementMetadata{
		DisplayName: map[string]string{"en_us": "Service", "zh_cn": "服务"},
		Description: map[string]string{"en_us": "A running service"},
		LaunchStage: "ga",
		Tags:        map[string]string{"team": "sre"},
		Properties:  map[string]string{"host.depends": "src_on_dest", "host.params": "a, b"},
	}
	if !reflect.DeepEqual(el.Metadata, want) {
		t.Errorf("metadata = %+v\nwant     %+v", el.Metadata, want)
	}
	// Identity is unchanged.
	if el.Name != "devops.service" || el.Domain != "devops" || el.Version != "v0.1.0" {
		t.Errorf("identity = %+v", el)
	}
}

// An element with no metadata beyond its identity keeps the compact form: no
// empty metadata object appears.
func TestImportWithoutMetadataHasNone(t *testing.T) {
	el := importOne(t, "kind: entity_set\nschema:\n  url: u\n  version: v0.1.0\nmetadata:\n  name: devops.service\n  domain: devops\n"+spec)
	if el.Metadata != nil {
		t.Errorf("metadata = %+v, want nil", el.Metadata)
	}
}

// A property written as a YAML number or bool is kept as text. Dropping it
// would make "min_count: 2" disappear without an error.
func TestImportKeepsNonStringProperties(t *testing.T) {
	el := importOne(t, "kind: entity_set\nschema:\n  url: u\n  version: v0.1.0\nmetadata:\n  name: devops.service\n  domain: devops\n"+
		"  properties:\n    min_count: 2\n    strict: true\n"+spec)
	want := map[string]string{"min_count": "2", "strict": "true"}
	if el.Metadata == nil || !reflect.DeepEqual(el.Metadata.Properties, want) {
		t.Errorf("properties = %+v, want %v", el.Metadata, want)
	}
}

// A plain string for display_name is accepted and kept under en_us, so a
// hand-written element is not silently stripped of its name.
func TestImportAcceptsPlainStringDisplayName(t *testing.T) {
	el := importOne(t, "kind: entity_set\nschema:\n  url: u\n  version: v0.1.0\nmetadata:\n  name: devops.service\n  domain: devops\n  display_name: Service\n"+spec)
	if el.Metadata == nil || el.Metadata.DisplayName["en_us"] != "Service" {
		t.Errorf("display_name = %+v", el.Metadata)
	}
}

// What is stored is a copy: changing the element after the write, or the one
// read back, does not change the store.
func TestStoredMetadataIsCopied(t *testing.T) {
	ctx := context.Background()
	store := graphstore.NewMemoryStore()
	el := model.UModelElement{
		Kind: "entity_set", Domain: "d", Name: "d.e", Version: "v0.1.0",
		Metadata: &model.ElementMetadata{Properties: map[string]string{"k": "v"}},
		Spec:     map[string]any{"fields": []any{map[string]any{"name": "id", "type": "string"}}, "primary_key_fields": []any{"id"}, "id_generator": "id"},
	}
	if _, err := store.PutUModelElements(ctx, model.UModelElementBatch{Workspace: "w", Elements: []model.UModelElement{el}}); err != nil {
		t.Fatal(err)
	}
	el.Metadata.Properties["k"] = "changed-after-write"
	snap, _ := store.GetUModelSnapshot(ctx, model.UModelSnapshotRequest{Workspace: "w"})
	if got := snap.Elements[0].Metadata.Properties["k"]; got != "v" {
		t.Errorf("store saw the caller's later change: %q", got)
	}
	snap.Elements[0].Metadata.Properties["k"] = "changed-after-read"
	again, _ := store.GetUModelSnapshot(ctx, model.UModelSnapshotRequest{Workspace: "w"})
	if got := again.Elements[0].Metadata.Properties["k"]; got != "v" {
		t.Errorf("a read returned the stored map itself: %q", got)
	}
}
