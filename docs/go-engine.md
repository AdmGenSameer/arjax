# ⚡ High-Performance Go Provider Engine

Arjax includes an optional compiled Go provider engine located in `arjax-provider-engine/`. It provides microsecond-level concurrent querying and standalone Bubbletea terminal interfaces.

---

## Architecture of `arjax-provider-engine`

```
arjax-provider-engine/
├── cache/            # In-memory and disk TTL cache
├── config/           # Provider configuration models
├── models/           # Go struct definitions (Package, Result, Info)
├── providers/        # Native providers (AUR, Repo, Flatpak, Snap, GitHub)
├── tui/              # Bubbletea interactive terminal UI
├── utils/            # Logging, system package manager detection
└── main.go           # CLI entrypoint with JSON IPC interface
```

---

## 🛠️ Building the Go Engine

```bash
cd arjax-provider-engine
go build -o bin/arjax-provider main.go
```

Once built in `bin/arjax-provider` (or placed in your `$PATH`), Arjax automatically detects it and uses it as the accelerated provider bridge.

---

## 🚀 Running the Go Engine Standalone

```bash
# JSON search output
./bin/arjax-provider search zapfast --json

# Standalone Bubbletea TUI
./bin/arjax-provider tui

# Package information
./bin/arjax-provider info firefox --json
```

---

## 🤝 Python Bridge Integration

The Python class `GoProviderEngine` (`arjax/provider_engine.py`) automatically discovers the binary:
1. Checks `$PATH` for `arjax-provider`.
2. Checks `arjax-provider-engine/bin/arjax-provider`.
3. Checks `~/.local/bin/arjax-provider`.
4. If missing, Arjax falls back to its built-in Python search and orchestrator pipelines without crashing.

