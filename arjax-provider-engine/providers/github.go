package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

type GitHubProvider struct{}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Name    string    `json:"name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type ghSearchResult struct {
	Items []struct {
		FullName    string `json:"full_name"`
		Description string `json:"description"`
	} `json:"items"`
}

func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{}
}

func (gp *GitHubProvider) Name() string {
	return "github"
}

func (gp *GitHubProvider) IsSupported() bool {
	return true
}

func (gp *GitHubProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	endpoint := fmt.Sprintf("https://api.github.com/search/repositories?q=%s", query)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "arjax-provider-engine")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var searchRes ghSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return nil, err
	}

	var pkgs []models.Package
	for i, item := range searchRes.Items {
		if i >= 10 {
			break
		}
		pkgs = append(pkgs, models.Package{
			Name:        item.FullName,
			Description: item.Description,
			Provider:    "github",
			Source:      "github",
		})
	}
	return pkgs, nil
}

func (gp *GitHubProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	repo := pkg.Name
	if !strings.Contains(repo, "/") {
		pkgs, err := gp.Search(ctx, repo)
		if err != nil || len(pkgs) == 0 {
			return models.Result{}, fmt.Errorf("github repository not found for %s", repo)
		}
		repo = pkgs[0].Name
	}

	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return models.Result{}, err
	}
	req.Header.Set("User-Agent", "arjax-provider-engine")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.Result{}, fmt.Errorf("github release not found for %s (status %d)", repo, resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return models.Result{}, err
	}

	pm := utils.DetectPackageManager()
	var targetAsset *ghAsset

	// On Debian/Pop!_OS (apt), look for .deb first
	if pm == "apt" {
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if strings.HasSuffix(name, ".deb") && (strings.Contains(name, "amd64") || strings.Contains(name, "x86_64") || strings.Contains(name, "all")) {
				targetAsset = &asset
				break
			}
		}
	}

	if targetAsset == nil {
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if strings.HasSuffix(name, ".appimage") || strings.HasSuffix(name, "-linux") || strings.HasSuffix(name, ".tar.gz") {
				targetAsset = &asset
				break
			}
		}
	}

	if targetAsset == nil && len(release.Assets) > 0 {
		targetAsset = &release.Assets[0]
	}

	if targetAsset == nil {
		return models.Result{}, fmt.Errorf("no suitable asset found in latest github release for %s", repo)
	}

	pkgBaseName := filepath.Base(repo)

	dlReq, _ := http.NewRequestWithContext(ctx, "GET", targetAsset.BrowserDownloadURL, nil)
	dlResp, err := client.Do(dlReq)
	if err != nil {
		return models.Result{}, err
	}
	defer dlResp.Body.Close()

	// If deb asset on apt system, install via apt
	if strings.HasSuffix(strings.ToLower(targetAsset.Name), ".deb") {
		tmpDeb := filepath.Join(os.TempDir(), targetAsset.Name)
		out, err := os.Create(tmpDeb)
		if err != nil {
			return models.Result{}, err
		}
		_, err = io.Copy(out, dlResp.Body)
		out.Close()
		if err != nil {
			return models.Result{}, err
		}
		defer os.Remove(tmpDeb)

		aptRes := utils.RunCommand(ctx, "sudo", "apt", "install", "-y", tmpDeb)
		endTime := time.Now()
		return models.Result{
			Package:    pkg.Name,
			Provider:   "github",
			Success:    aptRes.ExitCode == 0,
			StartTime:  start,
			EndTime:    endTime,
			DurationMs: endTime.Sub(start).Milliseconds(),
			Output:     fmt.Sprintf("Installed %s (.deb) via apt", targetAsset.Name),
			InstallCmd: fmt.Sprintf("sudo apt install -y %s", tmpDeb),
			Error:      aptRes.Stderr,
		}, aptRes.Error
	}

	// Download asset to ~/.local/bin
	localBin := os.ExpandEnv("$HOME/.local/bin")
	_ = os.MkdirAll(localBin, 0755)

	targetPath := filepath.Join(localBin, pkgBaseName)

	out, err := os.Create(targetPath)
	if err != nil {
		return models.Result{}, err
	}
	defer out.Close()

	_, err = io.Copy(out, dlResp.Body)
	if err != nil {
		return models.Result{}, err
	}

	_ = os.Chmod(targetPath, 0755)

	// Create desktop entry for desktop integration
	desktopDir := os.ExpandEnv("$HOME/.local/share/applications")
	_ = os.MkdirAll(desktopDir, 0755)
	desktopFile := filepath.Join(desktopDir, fmt.Sprintf("%s.desktop", pkgBaseName))
	desktopContent := fmt.Sprintf("[Desktop Entry]\nVersion=1.0\nType=Application\nName=%s\nExec=%s %%U\nTerminal=false\nCategories=Utility;Application;\n", strings.Title(pkgBaseName), targetPath)
	_ = os.WriteFile(desktopFile, []byte(desktopContent), 0644)

	endTime := time.Now()

	return models.Result{
		Package:    pkg.Name,
		Provider:   "github",
		Success:    true,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     fmt.Sprintf("Downloaded %s to %s", targetAsset.Name, targetPath),
		InstallCmd: fmt.Sprintf("curl -L %s -o %s && chmod +x %s", targetAsset.BrowserDownloadURL, targetPath, targetPath),
	}, nil
}

func (gp *GitHubProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	pkgBaseName := filepath.Base(pkg.Name)
	targetPath := os.ExpandEnv(fmt.Sprintf("$HOME/.local/bin/%s", pkgBaseName))

	err := os.Remove(targetPath)
	endTime := time.Now()

	return models.Result{
		Package:    pkg.Name,
		Provider:   "github",
		Success:    err == nil,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     fmt.Sprintf("Removed %s", targetPath),
		Error:      fmt.Sprintf("%v", err),
	}, err
}

func (gp *GitHubProvider) Update(ctx context.Context) (models.Result, error) {
	return models.Result{
		Package:  "github-update",
		Provider: "github",
		Success:  true,
	}, nil
}

func (gp *GitHubProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	if !strings.Contains(pkgName, "/") {
		return models.PackageInfo{}, fmt.Errorf("github provider requires owner/repo slug, got %q", pkgName)
	}
	return models.PackageInfo{
		Name:     pkgName,
		Provider: "github",
		Repo:     "github",
	}, nil
}

func (gp *GitHubProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	var results []models.Result
	for _, pkg := range pkgs {
		res, err := gp.Install(ctx, pkg)
		results = append(results, res)
		if err != nil {
			return results, err
		}
	}
	return results, nil
}
