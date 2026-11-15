# 📦 Adding New Packages (Recipe Guide)

Adding new packages to Arjax is fast and requires **no Python or Go programming**. You simply author a declarative YAML recipe.

---

## Step 1: Create the Recipe File

Create a YAML file named after your package in `arjax/recipes/<package>.yaml` (or in `~/.config/arjax/recipes/<package>.yaml` for local testing):

```yaml
name: myapp
display_name: My Awesome App
description: A short 1-sentence description of what your app does.
aliases:
  - myapp-cli
  - awesomeapp

providers:
  repository:
    package: myapp-bin
  flatpak:
    flatpak_id: com.example.MyApp
  github:
    github_repo: author/myapp
  appimage:
    appimage_url: https://github.com/author/myapp/releases/download/v1.0.0/myapp.AppImage
    appimage_name: myapp
```

---

## Step 2: Validate the Recipe

Run the recipe validation script to verify YAML syntax and required fields:

```bash
uv run python scripts/validate_mapping.py
```

---

## Step 3: Test Installation Locally

```bash
# Test through the interactive TUI
arjax

# Or test via CLI
arjax install myapp
```

Verify that:
1. The app resolves correctly.
2. If an AppImage is downloaded, it is installed to `~/.local/bin/myapp`.
3. A desktop launcher is created at `~/.local/share/applications/myapp.desktop`.
4. The application executes properly.

---

## Step 4: Submit a Pull Request

Commit your recipe in `arjax/recipes/<package>.yaml` and open a Pull Request against `AdmGenSameer/archpkg-helper`.

