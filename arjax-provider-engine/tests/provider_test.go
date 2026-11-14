package tests

import (
	"context"
	"testing"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/config"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
)

type mockProvider struct {
	name      string
	supported bool
}

func (m *mockProvider) Name() string     { return m.name }
func (m *mockProvider) IsSupported() bool { return m.supported }
func (m *mockProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	return []models.Package{{Name: query, Provider: m.name}}, nil
}
func (m *mockProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	return models.Result{Package: pkg.Name, Provider: m.name, Success: true}, nil
}
func (m *mockProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	return models.Result{Package: pkg.Name, Provider: m.name, Success: true}, nil
}
func (m *mockProvider) Update(ctx context.Context) (models.Result, error) {
	return models.Result{Package: "update", Provider: m.name, Success: true}, nil
}
func (m *mockProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	return models.PackageInfo{Name: pkgName, Provider: m.name}, nil
}
func (m *mockProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	var res []models.Result
	for _, p := range pkgs {
		res = append(res, models.Result{Package: p.Name, Provider: m.name, Success: true})
	}
	return res, nil
}

func TestProviderManagerSearchAll(t *testing.T) {
	cfg := config.DefaultConfig()
	mp1 := &mockProvider{name: "p1", supported: true}
	mp2 := &mockProvider{name: "p2", supported: true}

	pm := providers.NewProviderManager(cfg, mp1, mp2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pkgs, err := pm.SearchAll(ctx, "testpkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pkgs) != 2 {
		t.Errorf("expected 2 packages, got %d", len(pkgs))
	}
}

func TestProviderInfoDependencies(t *testing.T) {
	mp := &mockProvider{name: "p1", supported: true}
	info, err := mp.Info(context.Background(), "testpkg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "testpkg" {
		t.Errorf("expected name 'testpkg', got '%s'", info.Name)
	}
}

func BenchmarkProviderManagerSearchAll(b *testing.B) {
	cfg := config.DefaultConfig()
	mp1 := &mockProvider{name: "p1", supported: true}
	mp2 := &mockProvider{name: "p2", supported: true}

	pm := providers.NewProviderManager(cfg, mp1, mp2)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pm.SearchAll(ctx, "testpkg")
	}
}
