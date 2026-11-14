package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

type AURProvider struct {
	buildDir string
	useParu  bool
}

type aurRPCResponse struct {
	ResultCount int         `json:"resultcount"`
	Type        string      `json:"type"`
	Results     []aurResult `json:"results"`
}

type aurResult struct {
	Name        string   `json:"Name"`
	Version     string   `json:"Version"`
	Description string   `json:"Description"`
	NumVotes    int      `json:"NumVotes"`
	Popularity  float64  `json:"Popularity"`
	URLPath     string   `json:"URLPath"`
	Depends     []string `json:"Depends"`
}

func NewAURProvider(buildDir string, useParu bool) *AURProvider {
	if buildDir == "" {
		buildDir = os.ExpandEnv("$HOME/.cache/arjax/aur")
	}
	return &AURProvider{
		buildDir: buildDir,
		useParu:  useParu,
	}
}

func (ap *AURProvider) Name() string {
	return "aur"
}

func (ap *AURProvider) IsSupported() bool {
	return utils.CommandExists("pacman") || utils.CommandExists("paru") || utils.CommandExists("makepkg")
}

func (ap *AURProvider) Search(ctx context.Context, query string) ([]models.Package, error) {
	if !ap.IsSupported() {
		return nil, fmt.Errorf("AUR provider not supported")
	}

	endpoint := fmt.Sprintf("https://aur.archlinux.org/rpc/?v=5&type=search&arg=%s", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp aurRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}

	var pkgs []models.Package
	for _, res := range rpcResp.Results {
		pkgs = append(pkgs, models.Package{
			Name:        res.Name,
			Version:     res.Version,
			Description: res.Description,
			Provider:    "aur",
			Source:      "aur",
		})
	}
	return pkgs, nil
}

func (ap *AURProvider) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	if !ap.IsSupported() {
		return models.Result{}, fmt.Errorf("AUR provider not supported")
	}

	var installCmd string
	var execRes utils.ExecResult

	if ap.useParu && utils.CommandExists("paru") {
		installCmd = fmt.Sprintf("paru -S --noconfirm %s", pkg.Name)
		execRes = utils.RunCommand(ctx, "paru", "-S", "--noconfirm", pkg.Name)
	} else if utils.CommandExists("yay") {
		installCmd = fmt.Sprintf("yay -S --noconfirm %s", pkg.Name)
		execRes = utils.RunCommand(ctx, "yay", "-S", "--noconfirm", pkg.Name)
	} else {
		// Manual makepkg clone and build
		_ = os.MkdirAll(ap.buildDir, 0755)
		pkgDir := filepath.Join(ap.buildDir, pkg.Name)
		_ = os.RemoveAll(pkgDir)

		gitUrl := fmt.Sprintf("https://aur.archlinux.org/%s.git", pkg.Name)
		cloneRes := utils.RunCommand(ctx, "git", "clone", gitUrl, pkgDir)
		if cloneRes.ExitCode != 0 {
			return models.Result{
				Package:    pkg.Name,
				Provider:   "aur",
				Success:    false,
				StartTime:  start,
				EndTime:    time.Now(),
				DurationMs: time.Since(start).Milliseconds(),
				Output:     cloneRes.Stdout,
				Error:      fmt.Sprintf("git clone failed: %s", cloneRes.Stderr),
				InstallCmd: fmt.Sprintf("git clone %s %s", gitUrl, pkgDir),
			}, fmt.Errorf("git clone failed for AUR package %s", pkg.Name)
		}

		installCmd = fmt.Sprintf("cd %s && makepkg -si --noconfirm", pkgDir)
		makeRes := utils.RunCommand(ctx, "makepkg", "-si", "--noconfirm")
		execRes = makeRes
	}

	endTime := time.Now()
	return models.Result{
		Package:    pkg.Name,
		Provider:   "aur",
		Success:    execRes.ExitCode == 0,
		StartTime:  start,
		EndTime:    endTime,
		DurationMs: endTime.Sub(start).Milliseconds(),
		Output:     execRes.Stdout,
		Error:      execRes.Stderr,
		InstallCmd: installCmd,
	}, nil
}

func (ap *AURProvider) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	rp := NewRepositoryProvider("pacman")
	return rp.Remove(ctx, pkg)
}

func (ap *AURProvider) Update(ctx context.Context) (models.Result, error) {
	start := time.Now()
	if ap.useParu && utils.CommandExists("paru") {
		res := utils.RunCommand(ctx, "paru", "-Sua", "--noconfirm")
		return models.Result{
			Package:    "aur-update",
			Provider:   "aur",
			Success:    res.ExitCode == 0,
			StartTime:  start,
			EndTime:    time.Now(),
			DurationMs: time.Since(start).Milliseconds(),
			Output:     res.Stdout,
			Error:      res.Stderr,
		}, nil
	}
	return models.Result{
		Package:  "aur-update",
		Provider: "aur",
		Success:  true,
	}, nil
}

func (ap *AURProvider) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	endpoint := fmt.Sprintf("https://aur.archlinux.org/rpc/?v=5&type=info&arg[]=%s", url.QueryEscape(pkgName))
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return models.PackageInfo{}, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.PackageInfo{}, err
	}
	defer resp.Body.Close()

	var rpcResp aurRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return models.PackageInfo{}, err
	}

	if len(rpcResp.Results) == 0 {
		return models.PackageInfo{}, fmt.Errorf("AUR info not found for %s", pkgName)
	}

	res := rpcResp.Results[0]
	return models.PackageInfo{
		Name:         res.Name,
		Version:      res.Version,
		Description:  res.Description,
		Dependencies: res.Depends,
		Provider:     "aur",
		Repo:         "aur",
	}, nil
}

func (ap *AURProvider) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	var results []models.Result
	for _, pkg := range pkgs {
		res, err := ap.Install(ctx, pkg)
		results = append(results, res)
		if err != nil {
			return results, err
		}
	}
	return results, nil
}
