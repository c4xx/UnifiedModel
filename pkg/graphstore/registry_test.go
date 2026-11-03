package graphstore

import (
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/alibaba/UnifiedModel/pkg/contract"
)

type fakeStore struct {
	contract.GraphStore
	cfg ProviderConfig
}

func TestRegisterAndNewProvider(t *testing.T) {
	RegisterProvider("test.registry.basic", func(cfg ProviderConfig) (contract.GraphStore, error) {
		return &fakeStore{cfg: cfg}, nil
	})
	got, err := NewProvider(ProviderConfig{Type: "test.registry.basic", DataRoot: "/data", Options: map[string]string{"k": "v"}})
	if err != nil {
		t.Fatal(err)
	}
	fs, ok := got.(*fakeStore)
	if !ok {
		t.Fatalf("type = %T", got)
	}
	// The config reaches the factory unchanged.
	if fs.cfg.DataRoot != "/data" || fs.cfg.Options["k"] != "v" || fs.cfg.Type != "test.registry.basic" {
		t.Errorf("config = %+v", fs.cfg)
	}
}

func TestNewProviderUnregistered(t *testing.T) {
	_, err := NewProvider(ProviderConfig{Type: "test.registry.missing"})
	if err == nil {
		t.Fatal("an unregistered provider must be an error")
	}
	if want := `graphstore provider "test.registry.missing" is not registered`; err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
}

func TestFactoryErrorIsReturned(t *testing.T) {
	boom := errors.New("boom")
	RegisterProvider("test.registry.fail", func(ProviderConfig) (contract.GraphStore, error) { return nil, boom })
	if _, err := NewProvider(ProviderConfig{Type: "test.registry.fail"}); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the factory's error", err)
	}
}

func TestDefaultProvider(t *testing.T) {
	prev := DefaultProvider()
	t.Cleanup(func() { SetDefaultProvider(prev) })

	RegisterProvider("test.registry.default", func(cfg ProviderConfig) (contract.GraphStore, error) {
		return &fakeStore{cfg: cfg}, nil
	})
	SetDefaultProvider("test.registry.default")
	if DefaultProvider() != "test.registry.default" {
		t.Fatalf("DefaultProvider = %q", DefaultProvider())
	}
	// An empty Type selects the default.
	got, err := NewProvider(ProviderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.(*fakeStore); !ok {
		t.Errorf("type = %T", got)
	}

	// No default set: an empty Type is an error, not a panic or a silent pick.
	SetDefaultProvider("")
	if _, err := NewProvider(ProviderConfig{}); err == nil {
		t.Error("empty Type with no default must be an error")
	}
}

func TestRegisterReplaces(t *testing.T) {
	RegisterProvider("test.registry.replace", func(ProviderConfig) (contract.GraphStore, error) { return &fakeStore{}, nil })
	second := &fakeStore{cfg: ProviderConfig{Type: "second"}}
	RegisterProvider("test.registry.replace", func(ProviderConfig) (contract.GraphStore, error) { return second, nil })
	got, err := NewProvider(ProviderConfig{Type: "test.registry.replace"})
	if err != nil || got != contract.GraphStore(second) {
		t.Errorf("got %v, %v; the later registration must win", got, err)
	}
}

func TestRegisterPanicsOnMisuse(t *testing.T) {
	mustPanic := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s must panic", name)
			}
		}()
		f()
	}
	mustPanic("empty name", func() {
		RegisterProvider("", func(ProviderConfig) (contract.GraphStore, error) { return nil, nil })
	})
	mustPanic("nil factory", func() { RegisterProvider("test.registry.nil", nil) })
}

func TestRegisteredProvidersSorted(t *testing.T) {
	for _, n := range []string{"test.registry.z", "test.registry.a", "test.registry.m"} {
		RegisterProvider(n, func(ProviderConfig) (contract.GraphStore, error) { return nil, nil })
	}
	got := RegisteredProviders()
	if !sort.StringsAreSorted(got) {
		t.Errorf("not sorted: %v", got)
	}
	var mine []string
	for _, n := range got {
		if len(n) > 14 && n[:14] == "test.registry." && (n == "test.registry.a" || n == "test.registry.m" || n == "test.registry.z") {
			mine = append(mine, n)
		}
	}
	if !reflect.DeepEqual(mine, []string{"test.registry.a", "test.registry.m", "test.registry.z"}) {
		t.Errorf("mine = %v", mine)
	}
}
