package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// ScopedUpdater allows providers such as Flatpak (--user/--system ref) and Snap (single snap refresh)
// to upgrade a specific scoped offer safely, while system repos (Pacman) perform full upgrades.
type ScopedUpdater interface {
	UpdatePackage(ctx context.Context, pkg models.Package) (models.Result, error)
}

type FlatpakProvider struct{}

func NewFlatpakProvider() *FlatpakProvider {
	return &FlatpakProvider{}
}

func (fp *FlatpakProvider) Name() string {
	return "flatpak"
}

func (fp *FlatpakProvider) IsSupported() bool {
	return utils.CommandExists("flatpak")
}

func flatpakScopeFlag(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "user":
		return "--user"
	case "system":
		return "--system"
	default:
		return ""
	}
}

func flatpakQualifiedRef(appID, branch string) string {
	if b := strings.TrimSpace(branch); b != "" {
		return appID + "//" + b
	}
	return appID
}

func (fp *FlatpakProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	if !fp.IsSupported() {
		return nil, fmt.Errorf("flatpak binary not found")
	}

	res := utils.RunCommand(ctx, "flatpak", "search", query)
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("flatpak search failed: %s", res.Stderr)
	}

	var pkgs []models.Package
	lines := strings.Split(res.Stdout, "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			appID := strings.TrimSpace(parts[1])
			desc := ""
			if len(parts) >= 3 {
				desc = strings.TrimSpace(parts[2])
			}
			version := ""
			if len(parts) >= 4 {
				version = strings.TrimSpace(parts[3])
			}
			branch := ""
			if len(parts) >= 5 {
				branch = strings.TrimSpace(parts[4])
			}

			pkgs = append(pkgs, models.Package{
				Name:        appID,
				Version:     version,
				Description: name + " - " + desc,
				Provider:    "flatpak",
				Source:      "flathub",
				Branch:      branch,
			})
		}
	}
	return pkgs, nil
}

func (fp *FlatpakProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	if !fp.IsSupported() {
		return models.Result{}, fmt.Errorf("flatpak binary not found")
	}

	remote := pkg.Repo
	if remote == "" {
		remote = "flathub"
	}
	ref := flatpakQualifiedRef(pkg.Name, pkg.Branch)
	args := []string{"install", "-y"}
	if sf := flatpakScopeFlag(pkg.Scope); sf != "" {
		args = append(args, sf)
	}
	args = append(args, remote, ref)

	installCmd := "flatpak " + strings.Join(args, " ")
	res := utils.RunCommand(ctx, "flatpak", args...)
	endTime := time.Now()

	return models.Result{
		Package:    pkg.Name,
		Provider:   "flatpak",
		Success:    res.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     res.Stdout,
		Error:      res.Stderr,
		InstallCmd: installCmd,
	}, nil
}

func (fp *FlatpakProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	ref := flatpakQualifiedRef(pkg.Name, pkg.Branch)
	args := []string{"uninstall", "-y"}
	if sf := flatpakScopeFlag(pkg.Scope); sf != "" {
		args = append(args, sf)
	}
	args = append(args, ref)

	removeCmd := "flatpak " + strings.Join(args, " ")
	res := utils.RunCommand(ctx, "flatpak", args...)
	endTime := time.Now()

	return models.Result{
		Package:    pkg.Name,
		Provider:   "flatpak",
		Success:    res.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     res.Stdout,
		Error:      res.Stderr,
		InstallCmd: removeCmd,
	}, nil
}

// Update upgrades all Flatpak installations cleanly.
func (fp *FlatpakProvider) Update(ctx context.Context) (models.Result, error) {
	start := time.Now()
	res := utils.RunCommand(ctx, "flatpak", "update", "-y")
	endTime := time.Now()

	return models.Result{
		Package:    "flatpak-update",
		Provider:   "flatpak",
		Success:    res.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     res.Stdout,
		Error:      res.Stderr,
	}, nil
}

// UpdatePackage upgrades a specific Flatpak ref while preserving user vs system scope.
func (fp *FlatpakProvider) UpdatePackage(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	if !fp.IsSupported() {
		return models.Result{}, fmt.Errorf("flatpak binary not found")
	}

	args := []string{"update", "-y"}
	if sf := flatpakScopeFlag(pkg.Scope); sf != "" {
		args = append(args, sf)
	}
	if pkg.Name != "" && pkg.Name != "flatpak-update" {
		args = append(args, flatpakQualifiedRef(pkg.Name, pkg.Branch))
	}

	res := utils.RunCommand(ctx, "flatpak", args...)
	endTime := time.Now()

	return models.Result{
		Package:    pkg.Name,
		Provider:   "flatpak",
		Success:    res.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     res.Stdout,
		Error:      res.Stderr,
	}, nil
}

func (fp *FlatpakProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	res := utils.RunCommand(ctx, "flatpak", "info", pkgName)
	if res.ExitCode != 0 {
		return models.PackageInfo{}, fmt.Errorf("flatpak info failed for %s", pkgName)
	}

	info := models.PackageInfo{
		Name:     pkgName,
		Provider: "flatpak",
	}

	lines := strings.Split(res.Stdout, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if strings.EqualFold(k, "Version") {
				info.Version = v
			} else if strings.EqualFold(k, "Installation") {
				info.Repo = strings.TrimSpace(v)
			}
		}
	}
	return info, nil
}

func (fp *FlatpakProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	var results []models.Result
	for _, pkg := range pkgs {
		res, err := fp.Install(ctx, pkg)
		results = append(results, res)
		if err != nil {
			return results, err
		}
	}
	return results, nil
}
