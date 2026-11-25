package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

type VendorProvider struct{}

func NewVendorProvider() *VendorProvider {
	return &VendorProvider{}
}

func (vp *VendorProvider) Name() string {
	return "vendor"
}

func (vp *VendorProvider) IsSupported() bool {
	return true
}

func (vp *VendorProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	return []models.Package{}, nil
}

func (vp *VendorProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	if pkg.Source == "" {
		return models.Result{}, fmt.Errorf("vendor package requires custom command in source field")
	}

	installCmd := pkg.Source
	res := utils.RunCommand(ctx, "sh", "-c", installCmd)
	endTime := time.Now()

	return models.Result{
		Package:    pkg.Name,
		Provider:   "vendor",
		Success:    res.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     res.Stdout,
		Error:      res.Stderr,
		InstallCmd: installCmd,
	}, nil
}

func (vp *VendorProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	return models.Result{
		Package:  pkg.Name,
		Provider: "vendor",
		Success:  true,
	}, nil
}

func (vp *VendorProvider) Update(ctx context.Context) (models.Result, error) {
	return models.Result{
		Package:  "vendor-update",
		Provider: "vendor",
		Success:  true,
	}, nil
}

func (vp *VendorProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	return models.PackageInfo{}, fmt.Errorf("vendor info requires explicit custom source")
}

func (vp *VendorProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	var results []models.Result
	for _, pkg := range pkgs {
		res, err := vp.Install(ctx, pkg)
		results = append(results, res)
		if err != nil {
			return results, err
		}
	}
	return results, nil
}
