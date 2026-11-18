package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

type RepositoryProvider struct {
	pm string // pacman, apt, dnf, zypper
}

func NewRepositoryProvider(preferredPM string) *RepositoryProvider {
	pm := preferredPM
	if pm == "" {
		pm = utils.DetectPackageManager()
	}
	return &RepositoryProvider{pm: pm}
}

func (rp *RepositoryProvider) Name() string {
	return "repository"
}

func (rp *RepositoryProvider) IsSupported() bool {
	return rp.pm != "" && utils.CommandExists(rp.pm)
}

func (rp *RepositoryProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	if !rp.IsSupported() {
		return nil, fmt.Errorf("repository provider not supported on this system")
	}

	var args []string
	switch rp.pm {
	case "paru", "pacman":
		args = []string{"-Ss", query}
	case "apt":
		args = []string{"search", query}
	case "dnf":
		args = []string{"search", query}
	case "zypper":
		args = []string{"search", query}
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", rp.pm)
	}

	res := utils.RunCommand(ctx, rp.pm, args...)
	if res.Error != nil && res.ExitCode != 0 && res.Stdout == "" {
		return nil, res.Error
	}

	return rp.parseSearchResults(res.Stdout, query), nil
}

func (rp *RepositoryProvider) parseSearchResults(output string, query string) []models.Package {
	var pkgs []models.Package
	lines := strings.Split(output, "\n")

	switch rp.pm {
	case "paru", "pacman":
		for i := 0; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 && strings.Contains(parts[0], "/") {
				repoPkg := strings.Split(parts[0], "/")
				repo := repoPkg[0]
				name := repoPkg[1]
				version := parts[1]
				desc := ""
				if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "    ") {
					desc = strings.TrimSpace(lines[i+1])
					i++
				}
				pkgs = append(pkgs, models.Package{
					Name:        name,
					Version:     version,
					Description: desc,
					Provider:    "repository",
					Source:      "repository",
					Repo:        repo,
				})
			}
		}
	default:
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "Sorting...") || strings.HasPrefix(line, "Full Text Search...") {
				continue
			}
			parts := strings.SplitN(line, "-", 2)
			nameVer := strings.TrimSpace(parts[0])
			desc := ""
			if len(parts) > 1 {
				desc = strings.TrimSpace(parts[1])
			}
			nameParts := strings.Fields(nameVer)
			if len(nameParts) > 0 {
				pkgs = append(pkgs, models.Package{
					Name:        nameParts[0],
					Description: desc,
					Provider:    "repository",
					Source:      "repository",
				})
			}
		}
	}
	return pkgs
}

func (rp *RepositoryProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	if !rp.IsSupported() {
		return models.Result{}, fmt.Errorf("repository provider not supported")
	}

	var installCmd string
	var args []string

	switch rp.pm {
	case "paru":
		installCmd = fmt.Sprintf("paru -S --noconfirm %s", pkg.Name)
		args = []string{"-S", "--noconfirm", pkg.Name}
	case "pacman":
		installCmd = fmt.Sprintf("sudo pacman -S --noconfirm %s", pkg.Name)
		args = []string{"-S", "--noconfirm", pkg.Name}
	case "apt":
		installCmd = fmt.Sprintf("sudo apt-get install -y %s", pkg.Name)
		args = []string{"install", "-y", pkg.Name}
	case "dnf":
		installCmd = fmt.Sprintf("sudo dnf install -y %s", pkg.Name)
		args = []string{"install", "-y", pkg.Name}
	case "zypper":
		installCmd = fmt.Sprintf("sudo zypper install -n %s", pkg.Name)
		args = []string{"install", "-n", pkg.Name}
	}

	execRes := utils.RunCommand(ctx, rp.pm, args...)
	endTime := time.Now()
	duration := endTime.Sub(start).Milliseconds()

	success := execRes.ExitCode == 0
	errMsg := ""
	if !success {
		errMsg = execRes.Stderr
		if errMsg == "" {
			errMsg = fmt.Sprintf("command exited with code %d", execRes.ExitCode)
		}
	}

	return models.Result{
		Package:    pkg.Name,
		Provider:   "repository",
		Success:    success,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: duration,
		Output:     execRes.Stdout,
		Error:      errMsg,
		InstallCmd: installCmd,
	}, nil
}

func (rp *RepositoryProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	var removeCmd string
	var args []string

	switch rp.pm {
	case "paru":
		removeCmd = fmt.Sprintf("paru -R --noconfirm %s", pkg.Name)
		args = []string{"-R", "--noconfirm", pkg.Name}
	case "pacman":
		removeCmd = fmt.Sprintf("sudo pacman -R --noconfirm %s", pkg.Name)
		args = []string{"-R", "--noconfirm", pkg.Name}
	case "apt":
		removeCmd = fmt.Sprintf("sudo apt-get remove -y %s", pkg.Name)
		args = []string{"remove", "-y", pkg.Name}
	case "dnf":
		removeCmd = fmt.Sprintf("sudo dnf remove -y %s", pkg.Name)
		args = []string{"remove", "-y", pkg.Name}
	case "zypper":
		removeCmd = fmt.Sprintf("sudo zypper remove -n %s", pkg.Name)
		args = []string{"remove", "-n", pkg.Name}
	}

	execRes := utils.RunCommand(ctx, rp.pm, args...)
	endTime := time.Now()

	return models.Result{
		Package:    pkg.Name,
		Provider:   "repository",
		Success:    execRes.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     execRes.Stdout,
		Error:      execRes.Stderr,
		InstallCmd: removeCmd,
	}, nil
}

func (rp *RepositoryProvider) Update(ctx context.Context) (models.Result, error) {
	start := time.Now()
	var args []string

	switch rp.pm {
	case "pacman":
		args = []string{"-Syu", "--noconfirm"}
	case "apt":
		args = []string{"update"}
	case "dnf":
		args = []string{"check-update"}
	case "zypper":
		args = []string{"refresh"}
	}

	execRes := utils.RunCommand(ctx, rp.pm, args...)
	endTime := time.Now()

	return models.Result{
		Package:    "system-update",
		Provider:   "repository",
		Success:    execRes.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     execRes.Stdout,
		Error:      execRes.Stderr,
	}, nil
}

func (rp *RepositoryProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	var args []string
	switch rp.pm {
	case "paru", "pacman":
		args = []string{"-Si", pkgName}
	case "apt":
		args = []string{"show", pkgName}
	case "dnf":
		args = []string{"info", pkgName}
	case "zypper":
		args = []string{"info", pkgName}
	default:
		return models.PackageInfo{}, fmt.Errorf("unsupported package manager: %s", rp.pm)
	}

	if len(args) == 0 {
		return models.PackageInfo{}, fmt.Errorf("no arguments generated for package manager: %s", rp.pm)
	}

	res := utils.RunCommand(ctx, rp.pm, args...)
	if res.ExitCode != 0 {
		return models.PackageInfo{}, fmt.Errorf("info failed: %s", res.Stderr)
	}

	info := models.PackageInfo{
		Name:     pkgName,
		Provider: "repository",
	}

	lines := strings.Split(res.Stdout, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			switch strings.ToLower(key) {
			case "version":
				info.Version = val
			case "description":
				info.Description = val
			case "installed size", "download size", "size":
				info.Size = val
			case "repository", "repo":
				info.Repo = val
			case "depends on", "depends", "requires":
				if val != "" && strings.ToLower(val) != "none" {
					for _, rawDep := range strings.Fields(val) {
						clean := strings.Trim(rawDep, ",()")
						if clean != "" && clean != "None" && !strings.HasPrefix(clean, ">=") && !strings.HasPrefix(clean, "=") && !strings.HasPrefix(clean, "<=") {
							info.Dependencies = append(info.Dependencies, clean)
						}
					}
				}
			}
		}
	}
	return info, nil
}

func (rp *RepositoryProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	var results []models.Result
	for _, pkg := range pkgs {
		res, err := rp.Install(ctx, pkg)
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	return results, nil
}
