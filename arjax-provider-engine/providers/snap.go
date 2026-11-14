package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

type SnapProvider struct{}

func NewSnapProvider() *SnapProvider {
	return &SnapProvider{}
}

func (sp *SnapProvider) Name() string {
	return "snap"
}

func (sp *SnapProvider) IsSupported() bool {
	return utils.CommandExists("snap")
}

func (sp *SnapProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	if !sp.IsSupported() {
		return nil, fmt.Errorf("snap binary not found")
	}

	res := utils.RunCommand(ctx, "snap", "find", query)
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("snap find failed: %s", res.Stderr)
	}

	var pkgs []models.Package
	lines := strings.Split(res.Stdout, "\n")
	for i, line := range lines {
		if i == 0 || strings.HasPrefix(line, "Name") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			name := fields[0]
			version := fields[1]
			desc := strings.Join(fields[3:], " ")
			pkgs = append(pkgs, models.Package{
				Name:        name,
				Version:     version,
				Description: desc,
				Provider:    "snap",
				Source:      "snapcraft",
			})
		}
	}
	return pkgs, nil
}

func (sp *SnapProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	if !sp.IsSupported() {
		return models.Result{}, fmt.Errorf("snap binary not found")
	}

	installCmd := fmt.Sprintf("sudo snap install %s", pkg.Name)
	res := utils.RunCommand(ctx, "snap", "install", pkg.Name)
	endTime := time.Now()

	return models.Result{
		Package:    pkg.Name,
		Provider:   "snap",
		Success:    res.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     res.Stdout,
		Error:      res.Stderr,
		InstallCmd: installCmd,
	}, nil
}

func (sp *SnapProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	removeCmd := fmt.Sprintf("sudo snap remove %s", pkg.Name)
	res := utils.RunCommand(ctx, "snap", "remove", pkg.Name)
	endTime := time.Now()

	return models.Result{
		Package:    pkg.Name,
		Provider:   "snap",
		Success:    res.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     res.Stdout,
		Error:      res.Stderr,
		InstallCmd: removeCmd,
	}, nil
}

func (sp *SnapProvider) Update(ctx context.Context) (models.Result, error) {
	start := time.Now()
	res := utils.RunCommand(ctx, "snap", "refresh")
	endTime := time.Now()

	return models.Result{
		Package:    "snap-refresh",
		Provider:   "snap",
		Success:    res.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     res.Stdout,
		Error:      res.Stderr,
	}, nil
}

func (sp *SnapProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	res := utils.RunCommand(ctx, "snap", "info", pkgName)
	if res.ExitCode != 0 {
		return models.PackageInfo{}, fmt.Errorf("snap info failed for %s", pkgName)
	}

	info := models.PackageInfo{
		Name:     pkgName,
		Provider: "snap",
	}

	lines := strings.Split(res.Stdout, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if strings.EqualFold(k, "summary") {
				info.Description = v
			}
		}
	}
	return info, nil
}

func (sp *SnapProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	var results []models.Result
	for _, pkg := range pkgs {
		res, err := sp.Install(ctx, pkg)
		results = append(results, res)
		if err != nil {
			return results, err
		}
	}
	return results, nil
}
