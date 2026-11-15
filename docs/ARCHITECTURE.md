# 🏛️ Arjax System Architecture & Topology

Arjax separates **discovery (read-only search)** from **execution (isolated installation and state change)** across Linux distributions.

---

## 📐 High-Level Architecture

```mermaid
graph TD
    subgraph Client["Presentation & Interaction Tier"]
        TUI["Interactive TUI (Textual)"]
        CLI["Typer CLI"]
        GUI["PyQt5 GUI"]
    end

    subgraph Core["Core Python Orchestrator"]
        Router{"Subcommand Router"}
        SearchEng["Search Engine (Read-Only)"]
        InstallOrch["Installation Orchestrator"]
        Store["Recipe Store (YAML)"]
        TxMgr["Transaction Manager"]
        OrphanDAG["DAG Orphan Analyzer"]
    end

    subgraph Engine["Provider Engine Bridge"]
        GoBridge["GoProviderEngine (IPC)"]
        PyFallback["Native Python Search Fallback"]
        Cache["LRU Search Cache"]
    end

    subgraph Providers["Execution Providers"]
        P_Repo["Repository Provider (APT/Pacman/DNF/Zypper)"]
        P_AUR["AUR Provider (RPC v5 / Paru)"]
        P_Flatpak["Flatpak Provider (Flathub CLI)"]
        P_Snap["Snap Provider (Snapd REST)"]
        P_GH["GitHub Release / AppImage Provider"]
    end

    Client --> Router
    Router -->|search| SearchEng
    Router -->|install/remove| InstallOrch
    Router -->|history| TxMgr
    Router -->|maintenance| OrphanDAG

    SearchEng --> Store
    SearchEng --> Engine
    InstallOrch --> Store
    InstallOrch --> Engine

    Engine --> Providers
```

---

## 🔄 Search vs. Install Pipeline

### 1. Search Pipeline (Read-Only)

Search operations never require elevated privileges (`sudo`) and never alter system packages or state.

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant UI as TUI / CLI
    participant Engine as Search Engine
    participant Recipe as RecipeStore
    participant Providers as Native PMs / Flatpak / AUR

    User->>UI: Type query (e.g. 'zapfast')
    UI->>Engine: search_all_sources('zapfast')
    par Query Recipes
        Engine->>Recipe: load_all() & match aliases
        Recipe-->>Engine: zapfast (appimage, github, repo)
    and Query Engine & PMs
        Engine->>Providers: search_apt / search_pacman / search_flatpak
        Providers-->>Engine: Native package matches
    end
    Engine-->>UI: Deduplicated package list
    UI-->>User: Display results & highlight top match
```

### 2. Installation Pipeline (Timed & Benchmarked)

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant UI as TUI / CLI
    participant Orch as InstallationOrchestrator
    participant PM as ProviderManager
    participant Provider as Selected Provider
    participant Tx as TransactionManager

    User->>UI: Click [Install] or run 'arjax install <pkg>'
    UI->>Orch: install(pkg)
    Orch->>Orch: Step 1: Resolve recipe / package
    Orch->>PM: Step 2: Select candidate providers
    PM-->>Orch: Provider candidate list (priority ordered)
    loop For each candidate
        Orch->>Provider: Step 3: Check dependencies
        alt Dependencies met
            Orch->>Provider: Step 4: Run installation
            Provider-->>Orch: ProviderResult(success, message, duration)
        else Dependencies missing
            Orch->>Orch: Fallback to next candidate
        end
    end
    Orch->>Tx: Log transaction (id, action, duration_ms, status)
    Orch-->>UI: Complete with formatted timing summary
    UI-->>User: Visual success / error confirmation
```

---

## 📦 Provider Priority Matrix

When a package is installed, providers are evaluated strictly in the following priority order:

| Priority | Provider | Type | Description |
| :---: | :--- | :--- | :--- |
| **1** | `repository` | Native | Package manager detected on the host (`pacman`, `apt`, `dnf`, `zypper`). |
| **2** | `vendor` | Vendor | Third-party repositories or vendor repositories specified in recipe. |
| **3** | `github` | Binary / Source | Automated release resolution via GitHub REST API. |
| **4** | `flatpak` | Universal | Flathub containerized application. |
| **5** | `snap` | Universal | Canonical Snap Store package. |
| **6** | `appimage` | Standalone | Direct `.AppImage` executable placed in `~/.local/bin/`. |

---

## 🛡️ Reliability & Rollback

Every mutating operation (install, remove, rollback) is tracked in Arjax's **Transaction Manager**:
* Each transaction records: UUID, package name, provider, outcome, duration in milliseconds, and timestamp.
* The interactive TUI (`h` key) or CLI (`arjax history list`) provides real-time auditability.
* Failed transactions leave no lingering lock files or broken temporary extraction folders.

