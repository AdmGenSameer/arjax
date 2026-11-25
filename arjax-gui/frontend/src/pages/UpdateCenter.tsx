import React, { useState, useEffect } from 'react';
import {
  UpdateSummaryDTO,
  AvailableUpdateDTO,
  ProviderStatusDTO,
  TransactionDTO,
  TransactionPreviewDTO,
  UpdateBatchResultDTO,
} from '../types';

export const UpdateCenterPage: React.FC = () => {
  const [summary, setSummary] = useState<UpdateSummaryDTO | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [providerFilter, setProviderFilter] = useState<string>('ALL');
  const [providerStatuses, setProviderStatuses] = useState<ProviderStatusDTO[]>([]);
  const [transactions, setTransactions] = useState<TransactionDTO[]>([]);
  const [checking, setChecking] = useState(false);
  const [applying, setApplying] = useState(false);
  const [preview, setPreview] = useState<TransactionPreviewDTO | null>(null);
  const [batchResult, setBatchResult] = useState<UpdateBatchResultDTO | null>(null);
  const [activeTab, setActiveTab] = useState<'updates' | 'providers' | 'journal'>('updates');

  const loadData = async () => {
    const goApp = (window as any).go?.main?.App;
    if (!goApp) {
      setSummary({
        totalCount: 4,
        providerCount: 3,
        securityCount: 2,
        requiresFullSysUpgrade: true,
        byProvider: { pacman: 2, flatpak: 1, aur: 1 },
        notes: [
          'Arch Linux does not support partial upgrades: applying Pacman updates performs a full official system upgrade (pacman -Syu) before any AUR upgrades.',
        ],
        checkedAt: new Date().toISOString(),
        providerErrors: { snap: 'Snap is supported by Arjax, but snapd is not installed on host' },
        updates: [
          {
            offerId: 'pacman:extra:firefox',
            appId: 'org.mozilla.firefox',
            packageName: 'firefox',
            displayName: 'Firefox',
            provider: 'pacman',
            repository: 'extra',
            installScope: 'system',
            installedVersion: '124.0.1-1',
            availableVersion: '125.0.1-1',
            downloadSize: '78 MB',
            isSecurity: true,
            partialUpgradeSafe: false,
            upgradeGroupPolicy: 'full_system_upgrade',
            trustLevel: 'official',
            summary: 'Requires full Arch system upgrade (pacman -Syu)',
          },
          {
            offerId: 'pacman:core:openssl',
            appId: 'openssl',
            packageName: 'openssl',
            displayName: 'OpenSSL',
            provider: 'pacman',
            repository: 'core',
            installScope: 'system',
            installedVersion: '3.2.1-1',
            availableVersion: '3.3.0-1',
            downloadSize: '6 MB',
            isSecurity: true,
            partialUpgradeSafe: false,
            upgradeGroupPolicy: 'full_system_upgrade',
            trustLevel: 'official',
            summary: 'Requires full Arch system upgrade (pacman -Syu)',
          },
          {
            offerId: 'flatpak:flathub:org.blender.blender:scope=user:branch=stable',
            appId: 'org.blender.Blender',
            packageName: 'org.blender.Blender',
            displayName: 'Blender',
            provider: 'flatpak',
            repository: 'flathub',
            installScope: 'user',
            branch: 'stable',
            installedVersion: '4.2.1',
            availableVersion: '4.2.2',
            downloadSize: '310 MB',
            isSecurity: false,
            partialUpgradeSafe: true,
            upgradeGroupPolicy: 'scoped_ref',
            trustLevel: 'verified',
            summary: 'Flatpak update (user · flathub/stable)',
          },
          {
            offerId: 'aur:aur:visual-studio-code-bin',
            appId: 'com.visualstudio.code',
            packageName: 'visual-studio-code-bin',
            displayName: 'Visual Studio Code',
            provider: 'aur',
            repository: 'aur',
            installScope: 'user',
            installedVersion: '1.89.0-1',
            availableVersion: '1.90.0-1',
            downloadSize: 'PKGBUILD',
            isSecurity: false,
            partialUpgradeSafe: false,
            upgradeGroupPolicy: 'aur_helper_upgrade',
            trustLevel: 'community',
            summary: 'AUR community upgrade (requires official Pacman repos updated first; applies via paru -Sua)',
          },
        ],
      });
      return;
    }

    setChecking(true);
    try {
      const [upd, statuses, txs] = await Promise.all([
        goApp.CheckAllUpdates ? goApp.CheckAllUpdates() : Promise.resolve(null),
        goApp.GetProviderStatuses ? goApp.GetProviderStatuses() : Promise.resolve([]),
        goApp.GetTransactions ? goApp.GetTransactions() : Promise.resolve([]),
      ]);
      if (upd) {
        setSummary(upd);
        setSelectedIds(new Set((upd.updates || []).map((u: AvailableUpdateDTO) => u.offerId)));
      }
      if (Array.isArray(statuses)) setProviderStatuses(statuses);
      if (Array.isArray(txs)) setTransactions(txs);
    } finally {
      setChecking(false);
    }
  };

  useEffect(() => {
    loadData();
    const runtime = (window as any).runtime;
    if (!runtime || !runtime.EventsOn) return;
    const unsubTx = runtime.EventsOn('transaction:update', () => {
      const goApp = (window as any).go?.main?.App;
      if (goApp?.GetTransactions) {
        goApp.GetTransactions().then((txs: TransactionDTO[]) => {
          if (Array.isArray(txs)) setTransactions(txs);
        });
      }
    });
    const unsubRec = runtime.EventsOn('transactions:recovered', (rec: TransactionDTO[]) => {
      if (Array.isArray(rec) && rec.length > 0) {
        setTransactions((prev) => [...rec, ...prev]);
      }
    });
    return () => {
      if (typeof unsubTx === 'function') unsubTx();
      if (typeof unsubRec === 'function') unsubRec();
    };
  }, []);

  const toggleSelect = (item: AvailableUpdateDTO) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      const isSelected = next.has(item.offerId);
      if (item.provider.toLowerCase() === 'pacman') {
        const allPacman = (summary?.updates || []).filter((u) => u.provider.toLowerCase() === 'pacman');
        if (isSelected) {
          allPacman.forEach((u) => next.delete(u.offerId));
        } else {
          allPacman.forEach((u) => next.add(u.offerId));
        }
      } else {
        if (isSelected) next.delete(item.offerId);
        else next.add(item.offerId);
      }
      return next;
    });
  };

  // Explicit preview step BEFORE applying updates (protects against blind --noconfirm)
  const handleRequestPreview = async (ids: string[]) => {
    if (ids.length === 0 || applying) return;
    setBatchResult(null);
    const goApp = (window as any).go?.main?.App;
    if (goApp?.PreviewUpdates) {
      const prevDTO: TransactionPreviewDTO = await goApp.PreviewUpdates(ids);
      setPreview(prevDTO);
    } else if (goApp?.ApplySelectedUpdates) {
      const res = await goApp.ApplySelectedUpdates(ids);
      setBatchResult(res);
    } else {
      // Offline preview modal demonstration
      const pkgs = (summary?.updates || []).filter((u) => ids.includes(u.offerId));
      const hasPac = pkgs.some((u) => u.provider === 'pacman');
      const hasAUR = pkgs.some((u) => u.provider === 'aur');
      const pacTotal = (summary?.updates || []).filter((u) => u.provider === 'pacman').length;
      setPreview({
        previewId: 'prev_demo',
        requestedOfferIds: ids,
        effectiveOfferIds: ids,
        packagesToUpgrade: pkgs,
        packagesToInstall: [],
        packagesToRemove: [],
        estimatedDownloadSize: `${pkgs.length} package(s)`,
        requiresOfficialFirst: hasAUR && !hasPac && pacTotal > 0,
        includesFullSysUpgrade: hasPac,
        includesAurBuild: hasAUR,
        warnings: [
          ...(hasAUR && !hasPac && pacTotal > 0
            ? [`Blocked: ${pacTotal} official Pacman updates are pending. Official \`pacman -Syu\` must run first before upgrading AUR packages.`]
            : []),
          ...(hasPac ? ['Executes full official upgrade (`pacman -Syu`) with explicit approval instead of blind `--noconfirm`.'] : []),
        ],
        requiresExplicitApproval: true,
        createdAt: new Date().toISOString(),
        expiresAt: new Date(Date.now() + 600000).toISOString(),
      });
    }
  };

  const handleConfirmApprovedPreview = async (includeOfficialFirst = false) => {
    if (!preview || applying) return;
    setApplying(true);
    const goApp = (window as any).go?.main?.App;
    try {
      if (includeOfficialFirst && summary) {
        const pacIds = summary.updates.filter((u) => u.provider === 'pacman').map((u) => u.offerId);
        const combined = Array.from(new Set([...pacIds, ...preview.requestedOfferIds]));
        if (goApp?.PreviewUpdates && goApp?.ExecuteApprovedUpdates) {
          const freshPrev: TransactionPreviewDTO = await goApp.PreviewUpdates(combined);
          const res = await goApp.ExecuteApprovedUpdates(freshPrev.previewId, true);
          setBatchResult(res);
        }
      } else if (goApp?.ExecuteApprovedUpdates) {
        const res = await goApp.ExecuteApprovedUpdates(preview.previewId, true);
        setBatchResult(res);
      }
      setPreview(null);
      await loadData();
    } finally {
      setApplying(false);
    }
  };

  const updatesList = summary?.updates || [];
  const filteredUpdates = updatesList.filter((u) =>
    providerFilter === 'ALL' ? true : u.provider.toUpperCase() === providerFilter
  );
  const providerCounts = Object.entries(summary?.byProvider || {});
  const providerErrors = Object.entries(summary?.providerErrors || {});
  const hasPacmanPending = updatesList.some((u) => u.provider.toLowerCase() === 'pacman');
  const hasPacmanSelected = updatesList.some((u) => u.provider.toLowerCase() === 'pacman' && selectedIds.has(u.offerId));
  const hasAURSelected = updatesList.some((u) => u.provider.toLowerCase() === 'aur' && selectedIds.has(u.offerId));
  const aurOrderBlocked = hasAURSelected && hasPacmanPending && !hasPacmanSelected;

  return (
    <div className="p-8 space-y-6 max-w-6xl mx-auto font-sans">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2.5">
            <span className="material-symbols-outlined text-fluent-blue text-2xl">system_update</span>
            <h1 className="text-2xl font-bold tracking-tight text-fluent-text">Arjax Update Center</h1>
          </div>
          <p className="text-xs text-fluent-muted">
            Explicit transaction previews, Arch Pacman-before-AUR sequencing, Flatpak scope isolation, and crash recovery.
          </p>
        </div>

        <div className="flex items-center gap-2.5">
          <button
            type="button"
            onClick={loadData}
            disabled={checking || applying}
            className="px-3.5 py-2 rounded-lg bg-white/[0.06] hover:bg-white/[0.1] border border-mica-border text-xs font-medium text-fluent-text flex items-center gap-1.5 cursor-pointer"
          >
            <span className={`material-symbols-outlined text-[16px] ${checking ? 'animate-spin' : ''}`}>sync</span>
            <span>{checking ? 'Checking Providers...' : 'Refresh Updates'}</span>
          </button>

          <button
            type="button"
            onClick={() => handleRequestPreview(Array.from(selectedIds))}
            disabled={applying || selectedIds.size === 0}
            className="px-3.5 py-2 rounded-lg bg-surface-container-high hover:bg-surface-bright border border-white/10 text-xs font-medium text-fluent-text disabled:opacity-50 cursor-pointer"
          >
            Review &amp; Update Selected ({selectedIds.size})
          </button>

          <button
            type="button"
            onClick={() => handleRequestPreview(updatesList.map((u) => u.offerId))}
            disabled={applying || updatesList.length === 0}
            className="px-4 py-2 rounded-lg bg-fluent-blue hover:bg-fluent-blueHover text-white text-xs font-semibold shadow-sm disabled:opacity-50 flex items-center gap-1.5 cursor-pointer"
          >
            <span className="material-symbols-outlined text-[16px]">fact_check</span>
            <span>{applying ? 'Updating...' : `Review & Update All (${updatesList.length})`}</span>
          </button>
        </div>
      </div>

      {/* Arch Full Upgrade & Pacman-Before-AUR Policy Notices */}
      {summary?.requiresFullSysUpgrade && (
        <div className="rounded-xl bg-fluent-blue/10 border border-fluent-blue/25 p-3.5 text-xs flex items-start gap-2.5">
          <span className="material-symbols-outlined text-[18px] text-fluent-blue shrink-0 mt-0.5">policy</span>
          <div className="space-y-0.5">
            <strong className="text-white block">Arch Linux Full-Upgrade &amp; Sequencing Policy</strong>
            <span className="text-fluent-muted block">
              Official Pacman updates run together via <code className="text-fluent-blue font-mono">pacman -Syu</code> before AUR upgrades (<code className="text-fluent-blue font-mono">paru -Sua</code>), with an explicit approval preview replacing unconditional <code className="font-mono">--noconfirm</code>.
            </span>
          </div>
        </div>
      )}

      {aurOrderBlocked && (
        <div className="rounded-xl bg-amber-500/15 border border-amber-500/30 p-3.5 text-xs flex items-start gap-2.5">
          <span className="material-symbols-outlined text-[18px] text-amber-300 shrink-0 mt-0.5">gpp_maybe</span>
          <div className="space-y-0.5">
            <strong className="text-amber-200 block">Pacman-Before-AUR Guard Active</strong>
            <span className="text-amber-100/80 block">
              Official repository updates are pending. Arjax will block standalone AUR builds until official Pacman repositories are upgraded cleanly first.
            </span>
          </div>
        </div>
      )}

      {/* Transaction Approval Preview Modal/Panel */}
      {preview && (
        <div className="rounded-2xl bg-surface-container-high border-2 border-fluent-blue/50 p-5 space-y-4 shadow-2xl">
          <div className="flex items-center justify-between border-b border-white/10 pb-3">
            <div className="flex items-center gap-2">
              <span className="material-symbols-outlined text-fluent-blue">verified_user</span>
              <h2 className="text-base font-bold text-white">Transaction Preview &amp; Explicit Approval</h2>
            </div>
            <span className="text-xs font-mono text-fluent-muted">{preview.estimatedDownloadSize}</span>
          </div>

          {preview.warnings && preview.warnings.length > 0 && (
            <div className="space-y-1.5">
              {preview.warnings.map((w, idx) => (
                <div key={idx} className="p-2.5 rounded-lg bg-amber-500/10 border border-amber-500/20 text-xs text-amber-200">
                  {w}
                </div>
              ))}
            </div>
          )}

          <div className="max-h-56 overflow-y-auto space-y-1.5 pr-1">
            {preview.packagesToUpgrade.map((pkg) => (
              <div key={pkg.offerId} className="flex items-center justify-between text-xs p-2 rounded bg-black/20 border border-white/5">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-white">{pkg.displayName || pkg.packageName}</span>
                  <span className="font-mono text-[10px] px-1.5 py-0.5 rounded bg-white/5 uppercase">
                    {pkg.provider}{pkg.installScope ? `:${pkg.installScope}` : ''}
                  </span>
                </div>
                <span className="font-mono text-fluent-muted">
                  {pkg.installedVersion} → <strong className="text-emerald-400">{pkg.availableVersion}</strong>
                </span>
              </div>
            ))}
          </div>

          <div className="flex items-center justify-end gap-2.5 pt-2 border-t border-white/10">
            <button
              type="button"
              onClick={() => setPreview(null)}
              disabled={applying}
              className="px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-xs font-medium text-fluent-text cursor-pointer"
            >
              Cancel
            </button>
            {preview.requiresOfficialFirst ? (
              <button
                type="button"
                onClick={() => handleConfirmApprovedPreview(true)}
                disabled={applying}
                className="px-4 py-2 rounded-lg bg-amber-500 hover:bg-amber-600 text-black text-xs font-bold cursor-pointer"
              >
                {applying ? 'Applying Sequence...' : 'Approve Full pacman -Syu First, Then AUR'}
              </button>
            ) : (
              <button
                type="button"
                onClick={() => handleConfirmApprovedPreview(false)}
                disabled={applying}
                className="px-4 py-2 rounded-lg bg-fluent-blue hover:bg-fluent-blueHover text-white text-xs font-semibold cursor-pointer"
              >
                {applying ? 'Executing...' : 'Approve & Execute Transaction'}
              </button>
            )}
          </div>
        </div>
      )}

      {/* Top Metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="fluent-card p-4 rounded-xl bg-surface-container border border-white/5">
          <span className="text-xs text-fluent-muted block">Available Updates</span>
          <span className="text-2xl font-bold text-white mt-1 block">{summary?.totalCount ?? 0}</span>
          <div className="flex flex-wrap gap-1.5 mt-2">
            {providerCounts.map(([prov, cnt]) => (
              <span key={prov} className="text-[10.5px] font-mono px-2 py-0.5 rounded bg-white/5 text-fluent-text border border-white/10 uppercase">
                {prov}: {cnt}
              </span>
            ))}
          </div>
        </div>

        <div className="fluent-card p-4 rounded-xl bg-surface-container border border-white/5">
          <span className="text-xs text-fluent-muted block">Active Providers with Updates</span>
          <span className="text-2xl font-bold text-fluent-blue mt-1 block">{summary?.providerCount ?? 0}</span>
          <span className="text-[11px] text-fluent-dim mt-2 block">
            Pacman-before-AUR · No partial upgrades
          </span>
        </div>

        <div className="fluent-card p-4 rounded-xl bg-surface-container border border-white/5">
          <span className="text-xs text-fluent-muted block">Security-Sensitive Updates</span>
          <span className="text-2xl font-bold text-amber-400 mt-1 block">{summary?.securityCount ?? 0}</span>
          <span className="text-[11px] text-fluent-dim mt-2 block">
            Core runtime, TLS, kernel &amp; browser packages
          </span>
        </div>
      </div>

      {providerErrors.length > 0 && (
        <div className="rounded-xl bg-amber-500/10 border border-amber-500/20 p-3.5 text-xs space-y-1">
          <div className="font-semibold text-amber-300 flex items-center gap-1.5">
            <span className="material-symbols-outlined text-[16px]">warning</span>
            <span>Partial Provider Notices</span>
          </div>
          {providerErrors.map(([prov, msg]) => (
            <p key={prov} className="text-amber-200/80 font-mono text-[11px]">
              [{prov.toUpperCase()}] {msg}
            </p>
          ))}
        </div>
      )}

      {batchResult && (
        <div className="rounded-xl bg-surface-container p-4 border border-fluent-blue/30 text-xs space-y-2">
          <div className="flex items-center justify-between flex-wrap gap-2">
            <strong className="text-fluent-text">
              Batch Result: {batchResult.succeeded}/{batchResult.totalRequested} updated ({batchResult.failed} failed)
            </strong>
            <span className="text-fluent-dim">{batchResult.note}</span>
          </div>
          {batchResult.results.filter((r) => !r.success || r.skippedUpToDate).map((r) => (
            <div
              key={r.offerId}
              className={`font-mono text-[11px] ${r.skippedUpToDate ? 'text-emerald-300' : 'text-red-300'}`}
            >
              [{r.provider}{r.installScope ? `:${r.installScope}` : ''}] {r.packageName || r.offerId}: {r.errorMessage}
            </div>
          ))}
        </div>
      )}

      {/* Sub-navigation Tabs */}
      <div className="flex items-center gap-2 border-b border-white/10 pb-2">
        <button
          type="button"
          onClick={() => setActiveTab('updates')}
          className={`px-3.5 py-1.5 rounded-lg text-xs font-medium cursor-pointer ${
            activeTab === 'updates' ? 'bg-fluent-blue text-white' : 'text-fluent-muted hover:text-white'
          }`}
        >
          Pending Updates ({updatesList.length})
        </button>
        <button
          type="button"
          onClick={() => setActiveTab('providers')}
          className={`px-3.5 py-1.5 rounded-lg text-xs font-medium cursor-pointer ${
            activeTab === 'providers' ? 'bg-fluent-blue text-white' : 'text-fluent-muted hover:text-white'
          }`}
        >
          Provider Runtime Health ({providerStatuses.length})
        </button>
        <button
          type="button"
          onClick={() => setActiveTab('journal')}
          className={`px-3.5 py-1.5 rounded-lg text-xs font-medium cursor-pointer ${
            activeTab === 'journal' ? 'bg-fluent-blue text-white' : 'text-fluent-muted hover:text-white'
          }`}
        >
          Transaction Journal &amp; Recovery ({transactions.length})
        </button>
      </div>

      {/* Tab 1: Pending Updates */}
      {activeTab === 'updates' && (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            {['ALL', 'PACMAN', 'FLATPAK', 'AUR', 'SNAP'].map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => setProviderFilter(p)}
                className={`px-2.5 py-1 rounded text-[11px] font-mono uppercase cursor-pointer ${
                  providerFilter === p ? 'bg-primary/20 text-primary border border-primary/30' : 'bg-white/5 text-fluent-muted'
                }`}
              >
                {p}
              </button>
            ))}
          </div>

          {filteredUpdates.length === 0 ? (
            <div className="p-8 rounded-xl border border-dashed border-white/10 text-center text-fluent-muted text-sm">
              System is up to date for selected providers.
            </div>
          ) : (
            filteredUpdates.map((item) => {
              const checked = selectedIds.has(item.offerId);
              return (
                <div
                  key={item.offerId}
                  onClick={() => toggleSelect(item)}
                  className="p-4 rounded-xl bg-surface-container border border-white/5 hover:border-fluent-blue/30 flex items-center justify-between gap-4 cursor-pointer"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={() => toggleSelect(item)}
                      onClick={(e) => e.stopPropagation()}
                      className="w-4 h-4 accent-fluent-blue"
                    />
                    <div className="space-y-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-semibold text-sm text-white">{item.displayName || item.packageName}</span>
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-white/5 border border-white/10 uppercase">
                          {item.provider} ({item.repository})
                        </span>
                        {item.installScope && (
                          <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-purple-500/15 text-purple-300 border border-purple-500/25">
                            scope:{item.installScope}
                          </span>
                        )}
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-white/5 text-fluent-dim">
                          trust:{item.trustLevel}
                        </span>
                        {item.partialUpgradeSafe === false && item.provider === 'pacman' && (
                          <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-fluent-blue/15 text-fluent-blue border border-fluent-blue/30">
                            Full Upgrade Group (-Syu)
                          </span>
                        )}
                        {item.isSecurity && (
                          <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30">
                            Security
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-fluent-muted truncate">{item.summary}</p>
                    </div>
                  </div>

                  <div className="text-right shrink-0 font-mono text-xs">
                    <div className="text-fluent-text">
                      <span className="text-fluent-dim">{item.installedVersion}</span>
                      <span className="mx-1.5 text-fluent-blue">→</span>
                      <span className="text-emerald-400 font-semibold">{item.availableVersion}</span>
                    </div>
                    <span className="text-[11px] text-fluent-dim">{item.downloadSize}</span>
                  </div>
                </div>
              );
            })
          )}
        </div>
      )}

      {/* Tab 2: Provider Runtime Health */}
      {activeTab === 'providers' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {providerStatuses.map((st) => (
            <div key={st.provider} className="p-4 rounded-xl bg-surface-container border border-white/5 space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-sm text-white">{st.displayName}</span>
                <span
                  className={`text-[10px] font-mono px-2 py-0.5 rounded-full border ${
                    st.healthy
                      ? 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30'
                      : st.installed
                      ? 'bg-amber-500/15 text-amber-300 border-amber-500/30'
                      : 'bg-white/5 text-fluent-dim border-white/10'
                  }`}
                >
                  {st.healthy ? 'HEALTHY' : st.installed ? 'UNHEALTHY' : st.supported ? 'NOT INSTALLED' : 'PLANNED'}
                </span>
              </div>
              <p className="text-xs text-fluent-muted">{st.statusMessage}</p>
              <div className="flex flex-wrap gap-1.5 text-[10px] font-mono text-fluent-dim pt-1">
                <span>supported:{String(st.supported)}</span>
                <span>• installed:{String(st.installed)}</span>
                <span>• search:{st.capabilities.search}</span>
                <span>• install:{st.capabilities.install}</span>
                <span>• update:{st.capabilities.update}</span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Tab 3: Transaction Journal & Crash Recovery */}
      {activeTab === 'journal' && (
        <div className="space-y-2.5">
          {transactions.length === 0 ? (
            <div className="p-8 rounded-xl border border-dashed border-white/10 text-center text-fluent-muted text-sm">
              No journaled transactions recorded yet.
            </div>
          ) : (
            transactions.map((tx) => (
              <div key={tx.id} className="p-3.5 rounded-xl bg-surface-container border border-white/5 space-y-1 text-xs">
                <div className="flex items-center justify-between flex-wrap gap-2">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-semibold text-white">
                      {tx.operation.toUpperCase()} {tx.packageName}
                    </span>
                    <span className="font-mono text-[10px] px-2 py-0.5 rounded bg-white/5">{tx.provider}</span>
                    {tx.recoveryStatus && tx.recoveryStatus !== 'none' && (
                      <span
                        className={`font-mono text-[10px] px-2 py-0.5 rounded border ${
                          tx.recoveryStatus === 'outcome_unknown' || tx.recoveryStatus === 'recovery_required'
                            ? 'bg-amber-500/20 text-amber-300 border-amber-500/30'
                            : 'bg-fluent-blue/20 text-fluent-blue border-fluent-blue/30'
                        }`}
                      >
                        recovery:{tx.recoveryStatus}
                      </span>
                    )}
                  </div>
                  <span
                    className={`font-mono text-[11px] font-semibold uppercase ${
                      tx.state === 'completed'
                        ? 'text-emerald-400'
                        : tx.state === 'failed'
                        ? 'text-red-400'
                        : 'text-amber-300'
                    }`}
                  >
                    {tx.state}
                  </span>
                </div>
                <p className="text-fluent-muted">{tx.errorMessage || tx.message}</p>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
};
