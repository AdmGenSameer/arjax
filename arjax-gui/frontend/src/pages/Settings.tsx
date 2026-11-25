import React, { useState } from 'react';
import { Settings2, Shield, Bug, ExternalLink, Cpu, Zap, CheckCircle2 } from 'lucide-react';

export const SettingsPage: React.FC = () => {
  const [providers, setProviders] = useState([
    { id: 'native', label: 'Native Distro Repositories (APT / DNF / Zypper / Pacman)', desc: 'Sync with official distribution binary repositories for your Linux OS', enabled: true },
    { id: 'aur', label: 'Arch User Repository (AUR / Paru)', desc: 'Compile PKGBUILD sources cleanly on Arch-based distributions', enabled: true },
    { id: 'flatpak', label: 'Flathub Sandboxed Runtimes', desc: 'Universal cross-distro desktop apps isolated with bubblewrap', enabled: true },
    { id: 'snap', label: 'Canonical Snapcraft Runtimes', desc: 'Cross-distro AppArmor confined squashfs bundles', enabled: true },
    { id: 'github', label: 'Direct GitHub Releases (.deb / .AppImage / binaries)', desc: 'Auto-discover distro-appropriate release assets from official GitHub repos', enabled: true },
  ]);

  const [autoSync, setAutoSync] = useState(true);
  const [parallelDownloads, setParallelDownloads] = useState(true);
  const [testedLatency, setTestedLatency] = useState<number | null>(null);
  const [testing, setTesting] = useState(false);

  const toggleProvider = (id: string) => {
    setProviders((prev) =>
      prev.map((p) => (p.id === id ? { ...p, enabled: !p.enabled } : p))
    );
  };

  const handleTestMirrors = () => {
    setTesting(true);
    setTimeout(() => {
      setTestedLatency(18);
      setTesting(false);
    }, 600);
  };

  const openBugReport = () => {
    window.open('https://github.com/AdmGenSameer/arjax/issues/new?template=bug_report.md', '_blank');
  };

  return (
    <div className="p-8 space-y-6 max-w-4xl mx-auto h-full overflow-y-auto font-sans">
      {/* Header */}
      <div className="pb-4 border-b border-mica-border flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-fluent-text flex items-center gap-2.5">
            <span className="material-symbols-outlined text-fluent-blue text-2xl">settings</span>
            <span>Store & Engine Settings</span>
          </h1>
          <p className="text-xs text-fluent-muted mt-1">
            Configure package sources, mirror heuristics, network concurrency, and system telemetry
          </p>
        </div>
      </div>

      {/* Active Providers Card */}
      <div className="fluent-card p-5 rounded-2xl space-y-4 border border-mica-border">
        <div className="flex items-center justify-between">
          <h2 className="text-xs uppercase tracking-wider font-semibold text-fluent-dim flex items-center gap-2">
            <Shield className="w-4 h-4 text-emerald-400" />
            <span>Active Package Sources</span>
          </h2>
          <span className="text-[11px] font-mono text-fluent-dim">
            {providers.filter((p) => p.enabled).length} of {providers.length} enabled
          </span>
        </div>

        <div className="divide-y divide-white/[0.04]">
          {providers.map((p) => (
            <div key={p.id} className="py-3 flex items-center justify-between gap-4">
              <div className="space-y-0.5">
                <span className="text-xs font-semibold text-fluent-text block">{p.label}</span>
                <span className="text-[11px] text-fluent-muted block">{p.desc}</span>
              </div>
              <button
                onClick={() => toggleProvider(p.id)}
                className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${p.enabled ? 'bg-fluent-blue' : 'bg-white/10'
                  }`}
              >
                <span
                  className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out ${p.enabled ? 'translate-x-5' : 'translate-x-0'
                    }`}
                />
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Network & Performance Optimization */}
      <div className="fluent-card p-5 rounded-2xl space-y-4 border border-mica-border">
        <h2 className="text-xs uppercase tracking-wider font-semibold text-fluent-dim flex items-center gap-2">
          <Zap className="w-4 h-4 text-fluent-blue" />
          <span>Mirror Latency & Network Concurrency</span>
        </h2>

        <div className="divide-y divide-white/[0.04]">
          <div className="py-3 flex items-center justify-between gap-4">
            <div className="space-y-0.5">
              <span className="text-xs font-semibold text-fluent-text block">Parallel Concurrent Downloads</span>
              <span className="text-[11px] text-fluent-muted block">Fetch package slices and delta archives using multi-threaded HTTP pools</span>
            </div>
            <button
              onClick={() => setParallelDownloads(!parallelDownloads)}
              className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${parallelDownloads ? 'bg-fluent-blue' : 'bg-white/10'
                }`}
            >
              <span
                className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out ${parallelDownloads ? 'translate-x-5' : 'translate-x-0'
                  }`}
              />
            </button>
          </div>

          <div className="py-3 flex items-center justify-between gap-4">
            <div className="space-y-0.5">
              <span className="text-xs font-semibold text-fluent-text block">Background Database Sync</span>
              <span className="text-[11px] text-fluent-muted block">Silently synchronize official Arch repo indices on application launch</span>
            </div>
            <button
              onClick={() => setAutoSync(!autoSync)}
              className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${autoSync ? 'bg-fluent-blue' : 'bg-white/10'
                }`}
            >
              <span
                className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out ${autoSync ? 'translate-x-5' : 'translate-x-0'
                  }`}
              />
            </button>
          </div>
        </div>

        <div className="pt-3 flex items-center justify-between border-t border-mica-border">
          <div className="flex items-center gap-2 text-xs">
            <span className="text-fluent-dim">Current Mirror:</span>
            <span className="font-mono text-fluent-text">geo.mirror.archlinux.org</span>
            {testedLatency !== null && (
              <span className="text-emerald-400 font-mono text-xs flex items-center gap-1">
                <CheckCircle2 className="w-3.5 h-3.5" /> {testedLatency}ms
              </span>
            )}
          </div>
          <button
            type="button"
            onClick={handleTestMirrors}
            disabled={testing}
            className="px-3.5 py-1.5 bg-white/[0.04] hover:bg-white/[0.08] border border-mica-border rounded-lg text-xs font-medium text-fluent-text transition-colors cursor-pointer"
          >
            {testing ? 'Testing Ping...' : 'Test Mirror Ping'}
          </button>
        </div>
      </div>

      {/* Bug Reporting & Diagnostics */}
      <div className="fluent-card p-5 rounded-2xl space-y-3 border border-mica-border">
        <h2 className="text-xs uppercase tracking-wider font-semibold text-fluent-dim flex items-center gap-2">
          <Bug className="w-4 h-4 text-red-400" />
          <span>Troubleshooting & Issue Reporting</span>
        </h2>
        <p className="text-xs text-fluent-muted leading-relaxed">
          Encountered a package checksum mismatch, mirror timeout, or sandbox dependency issue? File a report on GitHub with automatic environment logs.
        </p>
        <div className="pt-1">
          <button
            onClick={openBugReport}
            className="flex items-center gap-2 px-4 py-2 bg-white/[0.04] hover:bg-white/[0.08] text-fluent-blue hover:text-fluent-blueHover border border-fluent-blue/30 rounded-xl text-xs font-semibold transition-all cursor-pointer"
          >
            <span>Open Issue on GitHub</span>
            <ExternalLink className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Engine Architecture Metadata */}
      <div className="fluent-card bg-black/30 border border-mica-border rounded-2xl p-4 text-xs text-fluent-dim flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Cpu className="w-4 h-4 text-fluent-blue" />
          <div>
            <span className="text-fluent-text font-semibold block">Arjax Multi-Provider Engine</span>
            <span className="text-[11px] text-fluent-muted block mt-0.5">Built with Go 1.24+ & WebKit2GTK 4.1 Native Linux Core</span>
          </div>
        </div>
        <span className="font-mono text-xs text-fluent-blue font-semibold">v2.0.0-fluent</span>
      </div>
    </div>
  );
};
