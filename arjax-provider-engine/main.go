package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/appstream"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/config"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/tui"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	jsonOutput := flag.Bool("json", false, "Output results in JSON format")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.CommandLine.Parse(os.Args[2:])

	utils.InitLogger(*verbose)
	cfg := config.DefaultConfig()

	pm := providers.NewProviderManager(
		cfg,
		providers.NewRepositoryProvider(cfg.Repository.PreferredPM),
		providers.NewAURProvider(cfg.AUR.BuildDir, cfg.AUR.UseParu),
		providers.NewGitHubProvider(),
		providers.NewFlatpakProvider(),
		providers.NewSnapProvider(),
		providers.NewVendorProvider(),
	)

	cm := catalog.NewCatalogManager(pm, appstream.NewCrawler(), nil, catalog.DefaultCatalogConfig(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout())
	defer cancel()

	args := flag.Args()

	switch command {
	case "search":
		if len(args) < 1 {
			exitWithError("search command requires a query string", *jsonOutput)
		}
		query := args[0]
		apps, err := cm.ProgressiveSearch().SearchSynchronous(ctx, query)
		if err != nil {
			exitWithError(err.Error(), *jsonOutput)
		}
		outputResponse(map[string]interface{}{
			"command":      "search",
			"query":        query,
			"count":        len(apps),
			"applications": apps,
		}, *jsonOutput)

	case "capabilities":
		outputResponse(map[string]interface{}{
			"command":      "capabilities",
			"capabilities": cm.Capabilities(),
		}, *jsonOutput)

	case "install":
		if len(args) < 1 {
			exitWithError("install command requires a package name", *jsonOutput)
		}
		pkgName := args[0]
		res, err := pm.Install(ctx, models.Package{Name: pkgName})
		if err != nil && !res.Success {
			res.Error = err.Error()
		}
		outputResponse(res, *jsonOutput)

	case "remove":
		if len(args) < 1 {
			exitWithError("remove command requires a package name", *jsonOutput)
		}
		pkgName := args[0]
		res, err := pm.Remove(ctx, models.Package{Name: pkgName})
		if err != nil && !res.Success {
			res.Error = err.Error()
		}
		outputResponse(res, *jsonOutput)

	case "info":
		if len(args) < 1 {
			exitWithError("info command requires a package name", *jsonOutput)
		}
		pkgName := args[0]
		info, err := pm.Info(ctx, pkgName)
		if err != nil {
			exitWithError(err.Error(), *jsonOutput)
		}
		outputResponse(info, *jsonOutput)

	case "tui":
		if err := tui.RunTUIWithCatalog(pm, cm); err != nil {
			exitWithError(err.Error(), false)
		}

	case "batch-install":
		if len(args) < 1 {
			exitWithError("batch-install command requires at least one package name", *jsonOutput)
		}
		var pkgs []models.Package
		for _, name := range args {
			pkgs = append(pkgs, models.Package{Name: name})
		}
		results, err := pm.BatchInstall(ctx, pkgs)
		if err != nil {
			utils.LogError("Batch install encountered errors: %v", err)
		}
		outputResponse(map[string]interface{}{
			"command": "batch-install",
			"results": results,
		}, *jsonOutput)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Arjax Provider Engine (Go Implementation)")
	fmt.Println("Usage: arjax-provider <command> [arguments] [--json] [--verbose]")
	fmt.Println("\nCommands:")
	fmt.Println("  search <query>          Search packages across all providers")
	fmt.Println("  install <package>       Install a package")
	fmt.Println("  remove <package>        Remove a package")
	fmt.Println("  info <package>          Get package details")
	fmt.Println("  batch-install <p1..pn>  Install multiple packages concurrently")
}

func outputResponse(data interface{}, isJSON bool) {
	if isJSON {
		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Printf(`{"error": "%s"}`+"\n", err.Error())
			os.Exit(1)
		}
		fmt.Println(string(out))
	} else {
		fmt.Printf("%+v\n", data)
	}
}

func exitWithError(msg string, isJSON bool) {
	if isJSON {
		out, _ := json.MarshalIndent(map[string]string{"error": msg}, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
	os.Exit(1)
}
