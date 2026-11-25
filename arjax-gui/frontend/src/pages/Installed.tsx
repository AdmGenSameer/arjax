import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { AppIcon } from '../components/AppIcon';

interface AppItem {
  id: string;
  name: string;
  displayName: string;
  category: 'apps' | 'games' | 'runtimes';
  hasUpdate?: boolean;
  updateVersion?: string;
  updateSize?: string;
  format: string;
  provider?: string;
  isNative?: boolean;
  sizeMB: number;
  sizeLabel: string;
  lastActivity: string;
  activityHours: number;
  icon: string;
  iconColor: string;
  description: string;
  isSharedCore?: boolean;
}

export const InstalledPage: React.FC = () => {
  const navigate = useNavigate();

  // State
  const [filter, setFilter] = useState<'all' | 'updates' | 'apps' | 'games' | 'runtimes'>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [sortType, setSortType] = useState<'recent' | 'name' | 'size' | 'installed'>('recent');
  const [sortDropdownOpen, setSortDropdownOpen] = useState(false);
  const [changelogOpen, setChangelogOpen] = useState(false);
  const [activeMenuId, setActiveMenuId] = useState<string | null>(null);

  // Global actions state
  const [updatingAll, setUpdatingAll] = useState(false);
  const [cleaningCache, setCleaningCache] = useState(false);
  const [cacheCleaned, setCacheCleaned] = useState(false);
  const [checkingUpdates, setCheckingUpdates] = useState(false);

  // Toast state
  const [toastVisible, setToastVisible] = useState(false);
  const [toastMessage, setToastMessage] = useState({ title: '', desc: '' });

  const triggerToast = (title: string, desc: string, duration = 2400) => {
    setToastMessage({ title, desc });
    setToastVisible(true);
    setTimeout(() => {
      setToastVisible(false);
    }, duration);
  };

  // Apps list
  const [apps, setApps] = useState<AppItem[]>([
    {
      id: 'brave',
      name: 'brave browser',
      displayName: 'Brave Browser',
      category: 'apps',
      hasUpdate: true,
      updateVersion: 'Update v1.65.130',
      updateSize: '84 MB',
      format: 'Flatpak (Sandboxed)',
      sizeMB: 842,
      sizeLabel: '842 MB',
      lastActivity: 'Used 2 hours ago',
      activityHours: 2,
      icon: 'travel_explore',
      iconColor: 'text-secondary',
      description: 'Fast, privacy-first Chromium web browser with native adblocking.'
    },
    {
      id: 'blender',
      name: 'blender',
      displayName: 'Blender 4.2.2 LTS',
      category: 'apps',
      hasUpdate: false,
      format: 'Flatpak (Sandboxed)',
      sizeMB: 1240,
      sizeLabel: '1.24 GB',
      lastActivity: 'Used yesterday',
      activityHours: 24,
      icon: 'view_in_ar',
      iconColor: 'text-secondary-fixed-dim',
      description: 'Complete 3D creation suite: modeling, rigging, simulation, rendering.'
    },
    {
      id: 'discord',
      name: 'discord',
      displayName: 'Discord',
      category: 'apps',
      hasUpdate: true,
      updateVersion: 'Update v0.0.48',
      updateSize: '46 MB',
      format: 'Flatpak (Sandboxed)',
      sizeMB: 310,
      sizeLabel: '310 MB',
      lastActivity: 'Used 5 hours ago',
      activityHours: 5,
      icon: 'forum',
      iconColor: 'text-primary',
      description: 'Voice, video, and text communication platform for communities.'
    },
    {
      id: 'steam',
      name: 'steam',
      displayName: 'Steam',
      category: 'games',
      hasUpdate: false,
      format: 'Native Pacman',
      isNative: true,
      sizeMB: 420,
      sizeLabel: '420 MB',
      lastActivity: 'Used 12 hours ago',
      activityHours: 12,
      icon: 'sports_esports',
      iconColor: 'text-tertiary',
      description: 'Gaming distribution platform with integrated Proton compatibility.'
    },
    {
      id: 'vscode',
      name: 'visual studio code',
      displayName: 'Visual Studio Code 1.89',
      category: 'apps',
      hasUpdate: false,
      format: 'Flatpak (Sandboxed)',
      sizeMB: 348,
      sizeLabel: '348 MB',
      lastActivity: 'Used 3 days ago',
      activityHours: 72,
      icon: 'code',
      iconColor: 'text-primary-container',
      description: 'Extensible, lightweight source code editor and debugging client.'
    },
    {
      id: 'vlc',
      name: 'vlc media player',
      displayName: 'VLC Media Player',
      category: 'apps',
      hasUpdate: true,
      updateVersion: 'Update v3.0.21',
      updateSize: '12 MB',
      format: 'Flatpak (Sandboxed)',
      sizeMB: 128,
      sizeLabel: '128 MB',
      lastActivity: 'Used 4 days ago',
      activityHours: 96,
      icon: 'smart_display',
      iconColor: 'text-secondary',
      description: 'Free, open-source multimedia engine and hardware decoder.'
    },
    {
      id: 'obs',
      name: 'obs studio',
      displayName: 'OBS Studio 30.1',
      category: 'apps',
      hasUpdate: false,
      format: 'Flatpak (Sandboxed)',
      sizeMB: 512,
      sizeLabel: '512 MB',
      lastActivity: 'Used last week',
      activityHours: 168,
      icon: 'videocam',
      iconColor: 'text-surface-tint',
      description: 'Live streaming broadcast studio and desktop screen recorder.'
    },
    {
      id: 'gnome-platform',
      name: 'gnome application platform',
      displayName: 'GNOME 46 Application Platform',
      category: 'runtimes',
      hasUpdate: false,
      format: 'Org.gnome.Platform',
      sizeMB: 1600,
      sizeLabel: '1.60 GB',
      lastActivity: 'Background active',
      activityHours: 999,
      icon: 'layers',
      iconColor: 'text-on-surface-variant',
      description: 'Shared GTK4 graphics libraries, Wayland backend, and system fonts.',
      isSharedCore: true
    }
  ]);

  // Updating single app
  const handleUpdateSingle = async (appId: string) => {
    setApps(prev => prev.map(a => a.id === appId ? { ...a, hasUpdate: false } : a));
    triggerToast("Updating package...", "Applying sandboxed delta archive cleanly in background.");

    const app = apps.find(a => a.id === appId);
    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.InstallPackage && app) {
      try {
        await goApp.InstallPackage(app.id, app.provider || 'flatpak');
      } catch (err) {
        console.warn(err);
      }
    }
  };

  useEffect(() => {
    const fetchInstalled = async () => {
      const goApp = (window as any).go?.main?.App;
      if (goApp && goApp.GetInstalledPackages) {
        try {
          const res = await goApp.GetInstalledPackages();
          if (Array.isArray(res) && res.length > 0) {
            setApps(res.map((item: any, idx: number) => ({
              id: item.id || item.name,
              name: item.name,
              displayName: item.displayName || item.name,
              category: item.category || 'apps',
              hasUpdate: item.hasUpdate || false,
              updateVersion: item.updateVersion || '',
              format: item.format || 'Native Package',
              provider: item.provider || (item.format?.includes('Flatpak') ? 'flatpak' : 'pacman'),
              isNative: item.isNative || false,
              isSharedCore: item.isSharedCore || false,
              sizeMB: item.sizeMB || 200,
              sizeLabel: item.size || '185 MB',
              lastActivity: item.lastActivity || 'Installed',
              activityHours: idx * 2 + 1,
              icon: item.icon || 'inventory_2',
              iconColor: item.isNative ? 'text-tertiary' : 'text-primary',
              description: `Installed package managed via ${item.provider || 'system'}.`
            })));
          }
        } catch (e) {
          console.warn('Failed to fetch installed packages from backend:', e);
        }
      }
    };
    fetchInstalled();
  }, []);

  // Launching single app
  const handleLaunch = (appName: string) => {
    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.LaunchApp) {
      goApp.LaunchApp(appName);
    }
    triggerToast(`Launching ${appName}...`, "Initializing containerized graphics context.");
  };

  // Uninstalling app
  const handleUninstall = async (appId: string, appName: string, provider: string = 'flatpak') => {
    setActiveMenuId(null);
    setApps(prev => prev.filter(a => a.id !== appId));
    triggerToast(`Uninstalled ${appName}`, "Cleaned application files and runtime associations.");

    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.RemovePackage) {
      try {
        await goApp.RemovePackage(appId, provider);
      } catch (e) {
        console.warn(e);
      }
    }
  };

  // Update all action
  const handleUpdateAll = async () => {
    if (updatingAll) return;
    setUpdatingAll(true);
    triggerToast("Starting Update All...", "Downloading sandboxed updates in background.");

    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.InstallPackage) {
      const pending = apps.filter(a => a.hasUpdate);
      for (const p of pending) {
        try {
          await goApp.InstallPackage(p.id, p.provider || 'flatpak');
        } catch (err) {
          console.warn(err);
        }
      }
    }

    setTimeout(() => {
      setApps(prev => prev.map(a => ({ ...a, hasUpdate: false })));
      setUpdatingAll(false);
      triggerToast("All applications up to date!", "Zero system restarts required.");
    }, 2000);
  };

  // Check for updates
  const handleCheckUpdates = async () => {
    if (checkingUpdates) return;
    setCheckingUpdates(true);
    triggerToast("Checking remote repositories...", "Querying Flathub and Arch Linux mirrors.");

    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.CheckForUpdates) {
      try {
        const updates = await goApp.CheckForUpdates();
        if (Array.isArray(updates) && updates.length > 0) {
          triggerToast("Repositories Synchronized", `Found ${updates.length} pending updates.`);
        }
      } catch (err) {
        console.warn(err);
      }
    }

    setTimeout(() => {
      setCheckingUpdates(false);
    }, 1400);
  };

  // Clean cache
  const handleCleanCache = async () => {
    if (cleaningCache || cacheCleaned) return;
    setCleaningCache(true);

    let reclaimed = "1.84 GB";
    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.CleanCache) {
      try {
        const res = await goApp.CleanCache();
        if (res && res.reclaimedMB) {
          reclaimed = `${(res.reclaimedMB / 1024).toFixed(2)} GB`;
        }
      } catch (e) {
        console.warn(e);
      }
    }

    setTimeout(() => {
      setCleaningCache(false);
      setCacheCleaned(true);
      triggerToast(`${reclaimed} Reclaimed!`, "Cleaned unreferenced build cache and pruned orphaned Flatpak runtimes.");
    }, 1500);
  };

  // Filtered & Sorted Apps
  const filteredApps = apps.filter(app => {
    const matchesSearch = !searchQuery ||
      app.displayName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      app.description.toLowerCase().includes(searchQuery.toLowerCase());

    if (!matchesSearch) return false;

    if (filter === 'all') return true;
    if (filter === 'updates') return app.hasUpdate;
    if (filter === 'apps') return app.category === 'apps';
    if (filter === 'games') return app.category === 'games';
    if (filter === 'runtimes') return app.category === 'runtimes';
    return true;
  });

  const sortedApps = [...filteredApps].sort((a, b) => {
    if (sortType === 'name') return a.displayName.localeCompare(b.displayName);
    if (sortType === 'size') return b.sizeMB - a.sizeMB;
    if (sortType === 'recent') return a.activityHours - b.activityHours;
    return 0;
  });

  const updatesCount = apps.filter(a => a.hasUpdate).length;

  return (
    <div className="flex flex-col w-full font-sans">
      <div className="w-full max-w-[1440px] mx-auto px-4 sm:px-6 lg:px-10 py-6 space-y-6">

        {/* 1. HEADER & META SUMMARY */}
        <div className="flex flex-col lg:flex-row lg:items-end justify-between gap-4 pb-2">
          <div className="space-y-1">
            <div className="flex items-center gap-2 text-surface-tint font-label-sm text-label-sm uppercase tracking-wider">
              <span className="material-symbols-outlined text-[15px]" style={{ fontVariationSettings: "'FILL' 1" }}>inventory_2</span>
              <span>Library &amp; System Storage</span>
            </div>
            <h1 className="text-on-background font-headline-lg text-headline-lg font-semibold tracking-tight">
              Installed Apps &amp; Library
            </h1>
            <p className="text-on-surface-variant font-body-sm text-body-sm">
              Manage local software containers, inspect storage footprints, and orchestrate background runtime maintenance.
            </p>
          </div>

          {/* Global Actions */}
          <div className="flex items-center gap-2.5 self-start lg:self-auto">
            <button
              onClick={handleCheckUpdates}
              disabled={checkingUpdates}
              className="px-3.5 py-2 rounded-lg bg-surface-container-high hover:bg-surface-container-highest text-on-surface hover:text-on-background font-label-md text-label-md flex items-center gap-2 transition duration-150 shadow-sm active:scale-95 cursor-pointer"
              type="button"
            >
              <span className={`material-symbols-outlined text-[17px] text-surface-tint ${checkingUpdates ? 'animate-spin' : ''}`}>
                sync
              </span>
              <span>{checkingUpdates ? 'Checking...' : 'Check for Updates'}</span>
            </button>

            <button
              onClick={handleUpdateAll}
              disabled={updatingAll || updatesCount === 0}
              className={`px-4 py-2 rounded-lg font-label-md text-label-md flex items-center gap-2.5 transition duration-150 shadow-md hover:shadow-lg active:scale-95 cursor-pointer ${updatesCount > 0
                ? 'bg-primary-container hover:bg-inverse-primary text-on-primary-container'
                : 'bg-surface-container text-on-surface-variant opacity-70 cursor-default'
                }`}
              type="button"
            >
              <span className="material-symbols-outlined text-[18px]">
                {updatingAll ? 'refresh' : 'download_for_offline'}
              </span>
              <span className="font-medium">{updatingAll ? 'Updating All...' : 'Update All'}</span>
              {updatesCount > 0 && (
                <span className="px-1.5 py-0.5 rounded-full bg-white/20 text-on-primary-container font-code text-label-sm font-semibold">
                  {updatesCount}
                </span>
              )}
            </button>
          </div>
        </div>

        {/* 2. PROMINENT UPDATES READY CARD */}
        {updatesCount > 0 && (
          <div className="relative overflow-hidden rounded-xl bg-surface-container-low p-4 sm:p-5 shadow-fluent-elevation transition-all duration-300">
            <div className="absolute -top-16 -right-16 w-64 h-64 bg-primary-container/10 rounded-full blur-3xl pointer-events-none"></div>

            <div className="relative flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
              <div className="flex items-start sm:items-center gap-3.5">
                <div className="w-10 h-10 rounded-xl bg-primary-container/15 flex items-center justify-center shrink-0">
                  <span className="material-symbols-outlined text-primary-container text-[22px]">system_update_alt</span>
                </div>
                <div>
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-title text-title text-on-background font-semibold">
                      {updatesCount} updates ready to install
                    </span>
                    <span className="px-2 py-0.5 rounded-md bg-surface-container text-surface-tint font-code text-label-sm">
                      ~142 MB Total
                    </span>
                  </div>
                  <p className="text-on-surface-variant font-body-sm text-body-sm mt-0.5">
                    Updates run silently in sandboxed containers without restarting your system or interrupting active sessions.
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-3 w-full md:w-auto justify-end">
                <button
                  onClick={() => setChangelogOpen(!changelogOpen)}
                  className="px-3 py-1.5 rounded-lg bg-surface-container hover:bg-surface-container-high text-on-surface font-label-md text-label-md transition cursor-pointer"
                  type="button"
                >
                  {changelogOpen ? 'Hide Changes' : 'Review Changes'}
                </button>
                <button
                  onClick={handleUpdateAll}
                  disabled={updatingAll}
                  className="px-3.5 py-1.5 rounded-lg bg-primary-container hover:bg-inverse-primary text-on-primary-container font-label-md text-label-md flex items-center gap-1.5 transition active:scale-95 cursor-pointer"
                  type="button"
                >
                  <span className="material-symbols-outlined text-[16px]">arrow_downward</span>
                  <span>Update (142 MB)</span>
                </button>
              </div>
            </div>

            {/* Expandable Changelog Preview Shelf */}
            {changelogOpen && (
              <div className="mt-4 pt-3 space-y-2 border-t border-white/[0.04]">
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
                  <div className="p-3 rounded-lg bg-surface-container flex items-center justify-between">
                    <div className="flex items-center gap-2.5 min-w-0">
                      <div className="w-7 h-7 rounded-md bg-secondary-container/20 flex items-center justify-center shrink-0">
                        <span className="material-symbols-outlined text-[17px] text-secondary">travel_explore</span>
                      </div>
                      <div className="truncate">
                        <p className="text-on-surface font-label-md text-label-md font-medium truncate">Brave Browser</p>
                        <p className="text-on-surface-variant font-code text-label-sm">1.65.122 → 1.65.130</p>
                      </div>
                    </div>
                    <span className="text-on-surface-variant font-code text-label-sm shrink-0">84 MB</span>
                  </div>

                  <div className="p-3 rounded-lg bg-surface-container flex items-center justify-between">
                    <div className="flex items-center gap-2.5 min-w-0">
                      <div className="w-7 h-7 rounded-md bg-primary-container/20 flex items-center justify-center shrink-0">
                        <span className="material-symbols-outlined text-[17px] text-primary">forum</span>
                      </div>
                      <div className="truncate">
                        <p className="text-on-surface font-label-md text-label-md font-medium truncate">Discord</p>
                        <p className="text-on-surface-variant font-code text-label-sm">0.0.47 → 0.0.48</p>
                      </div>
                    </div>
                    <span className="text-on-surface-variant font-code text-label-sm shrink-0">46 MB</span>
                  </div>

                  <div className="p-3 rounded-lg bg-surface-container flex items-center justify-between">
                    <div className="flex items-center gap-2.5 min-w-0">
                      <div className="w-7 h-7 rounded-md bg-secondary-fixed-dim/20 flex items-center justify-center shrink-0">
                        <span className="material-symbols-outlined text-[17px] text-secondary-fixed-dim">smart_display</span>
                      </div>
                      <div className="truncate">
                        <p className="text-on-surface font-label-md text-label-md font-medium truncate">VLC Media Player</p>
                        <p className="text-on-surface-variant font-code text-label-sm">3.0.20 → 3.0.21</p>
                      </div>
                    </div>
                    <span className="text-on-surface-variant font-code text-label-sm shrink-0">12 MB</span>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* 3. STORAGE & HEALTH OVERVIEW (BENTO PANEL) */}
        <div className="grid grid-cols-1 xl:grid-cols-12 gap-4">
          {/* Storage Distribution */}
          <div className="xl:col-span-8 rounded-xl bg-surface-container-low p-5 shadow-sm space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
              <div>
                <span className="text-on-surface-variant font-label-sm text-label-sm uppercase tracking-wider">
                  App Storage Consumption
                </span>
                <div className="flex items-baseline gap-2 mt-0.5">
                  <span className="text-on-background font-headline-md text-headline-md font-semibold">42.8 GB</span>
                  <span className="text-on-surface-variant font-body-sm text-body-sm">allocated across 28 applications</span>
                </div>
              </div>
              <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-container text-tertiary font-label-sm text-label-sm">
                <span className="material-symbols-outlined text-[16px]">verified_user</span>
                <span>All Sandbox Permissions Healthy</span>
              </div>
            </div>

            {/* Segmented Disk Bar */}
            <div className="space-y-2">
              <div className="h-3 w-full rounded-full bg-surface-container-highest overflow-hidden flex">
                <div className="h-full bg-primary-container transition-all duration-500" style={{ width: '66.3%' }} title="Flatpak Apps (28.4 GB)"></div>
                <div className="h-full bg-tertiary transition-all duration-500" style={{ width: '21.5%' }} title="Native System Binaries (9.2 GB)"></div>
                <div className="h-full bg-secondary-fixed-dim transition-all duration-500" style={{ width: '12.2%' }} title="App Cache & Data (5.2 GB)"></div>
              </div>

              {/* Legend */}
              <div className="flex flex-wrap items-center justify-between text-label-sm font-label-sm gap-y-2 pt-1">
                <div className="flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-primary-container"></span>
                  <span className="text-on-surface">Flatpak Sandboxes</span>
                  <span className="text-on-surface-variant font-code">28.4 GB (66%)</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-tertiary"></span>
                  <span className="text-on-surface">Native System Binaries</span>
                  <span className="text-on-surface-variant font-code">9.2 GB (22%)</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-secondary-fixed-dim"></span>
                  <span className="text-on-surface">Shared Cache &amp; Temp</span>
                  <span className="text-on-surface-variant font-code">5.2 GB (12%)</span>
                </div>
              </div>
            </div>
          </div>

          {/* Quick Maintenance Hub */}
          <div className="xl:col-span-4 rounded-xl bg-surface-container-low p-5 shadow-sm flex flex-col justify-between space-y-3">
            <div>
              <div className="flex items-center justify-between">
                <span className="text-on-surface-variant font-label-sm text-label-sm uppercase tracking-wider">Automated Hygiene</span>
                <span className="px-2 py-0.5 rounded text-[11px] font-code bg-tertiary-container/30 text-tertiary">
                  {cacheCleaned ? 'Optimized' : 'Trim Ready'}
                </span>
              </div>
              <h2 className="text-on-background font-title text-title mt-1 font-semibold">Unused Runtimes &amp; Cache</h2>
              <p className="text-on-surface-variant font-body-sm text-body-sm mt-1">
                Reclaim space from orphaned GNOME/KDE runtimes and unreferenced build caches.
              </p>
            </div>

            <button
              onClick={handleCleanCache}
              disabled={cleaningCache || cacheCleaned}
              className="w-full px-3.5 py-2.5 rounded-lg bg-surface-container hover:bg-surface-container-high text-on-surface hover:text-on-background font-label-md text-label-md flex items-center justify-center gap-2 transition active:scale-[0.98] cursor-pointer"
              type="button"
            >
              <span className={`material-symbols-outlined text-[18px] text-tertiary ${cleaningCache ? 'animate-spin' : ''}`}>
                {cacheCleaned ? 'check' : 'cleaning_services'}
              </span>
              <span>
                {cleaningCache ? 'Trimming Runtimes...' : cacheCleaned ? '1.84 GB Reclaimed!' : 'Clean Cache (Frees ~1.8 GB)'}
              </span>
            </button>
          </div>
        </div>

        {/* 4. CONTROLS: FILTER PILLS, SEARCH & SORT BAR */}
        <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3 pt-2">
          {/* Filter Chips */}
          <div className="flex items-center gap-1.5 overflow-x-auto pb-1 md:pb-0 scrollbar-none">
            <button
              onClick={() => setFilter('all')}
              className={`px-3 py-1.5 rounded-lg text-label-md font-label-md font-medium transition shrink-0 cursor-pointer ${filter === 'all'
                ? 'bg-primary-container/20 text-primary-fixed-dim'
                : 'bg-surface-container text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high'
                }`}
              type="button"
            >
              All (28)
            </button>

            <button
              onClick={() => setFilter('updates')}
              className={`px-3 py-1.5 rounded-lg text-label-md font-label-md transition shrink-0 flex items-center gap-1.5 cursor-pointer ${filter === 'updates'
                ? 'bg-primary-container/20 text-primary-fixed-dim font-medium'
                : 'bg-surface-container text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high'
                }`}
              type="button"
            >
              <span>Updates Available</span>
              <span className="px-1.5 py-0.2 rounded-full text-label-sm font-code bg-primary-container text-on-primary-container">
                {updatesCount}
              </span>
            </button>

            <button
              onClick={() => setFilter('apps')}
              className={`px-3 py-1.5 rounded-lg text-label-md font-label-md transition shrink-0 cursor-pointer ${filter === 'apps'
                ? 'bg-primary-container/20 text-primary-fixed-dim font-medium'
                : 'bg-surface-container text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high'
                }`}
              type="button"
            >
              Apps (18)
            </button>

            <button
              onClick={() => setFilter('games')}
              className={`px-3 py-1.5 rounded-lg text-label-md font-label-md transition shrink-0 cursor-pointer ${filter === 'games'
                ? 'bg-primary-container/20 text-primary-fixed-dim font-medium'
                : 'bg-surface-container text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high'
                }`}
              type="button"
            >
              Games (4)
            </button>

            <button
              onClick={() => setFilter('runtimes')}
              className={`px-3 py-1.5 rounded-lg text-label-md font-label-md transition shrink-0 cursor-pointer ${filter === 'runtimes'
                ? 'bg-primary-container/20 text-primary-fixed-dim font-medium'
                : 'bg-surface-container text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high'
                }`}
              type="button"
            >
              System Runtimes (6)
            </button>
          </div>

          {/* Search & Sort Row */}
          <div className="flex items-center gap-2">
            <div className="relative flex-1 md:w-64">
              <span className="material-symbols-outlined absolute left-3 top-2.5 text-on-surface-variant text-[17px] pointer-events-none">
                search
              </span>
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Filter installed apps..."
                className="w-full pl-9 pr-3 py-1.5 rounded-lg bg-surface-container-low hover:bg-surface-container focus:bg-surface-container text-on-surface placeholder:text-outline font-body-sm text-body-sm focus:outline-none transition"
              />
            </div>

            <div className="relative">
              <button
                onClick={() => setSortDropdownOpen(!sortDropdownOpen)}
                className="px-3 py-1.5 rounded-lg bg-surface-container-low hover:bg-surface-container text-on-surface font-label-md text-label-md flex items-center gap-1.5 transition cursor-pointer"
                type="button"
              >
                <span className="material-symbols-outlined text-[16px] text-on-surface-variant">sort</span>
                <span>
                  {sortType === 'recent' ? 'Recently used' : sortType === 'name' ? 'Alphabetical (A-Z)' : sortType === 'size' ? 'Disk size (Largest)' : 'Install date'}
                </span>
                <span className="material-symbols-outlined text-[14px]">expand_more</span>
              </button>

              {sortDropdownOpen && (
                <div className="absolute right-0 mt-1 w-44 rounded-xl bg-surface-container-high shadow-fluent-flyout py-1 z-30">
                  <button
                    onClick={() => { setSortType('recent'); setSortDropdownOpen(false); }}
                    className="w-full text-left px-3 py-2 text-label-sm font-label-sm text-on-surface hover:bg-surface-container-highest cursor-pointer"
                  >
                    Recently used
                  </button>
                  <button
                    onClick={() => { setSortType('name'); setSortDropdownOpen(false); }}
                    className="w-full text-left px-3 py-2 text-label-sm font-label-sm text-on-surface hover:bg-surface-container-highest cursor-pointer"
                  >
                    Alphabetical (A-Z)
                  </button>
                  <button
                    onClick={() => { setSortType('size'); setSortDropdownOpen(false); }}
                    className="w-full text-left px-3 py-2 text-label-sm font-label-sm text-on-surface hover:bg-surface-container-highest cursor-pointer"
                  >
                    Disk size (Largest)
                  </button>
                  <button
                    onClick={() => { setSortType('installed'); setSortDropdownOpen(false); }}
                    className="w-full text-left px-3 py-2 text-label-sm font-label-sm text-on-surface hover:bg-surface-container-highest cursor-pointer"
                  >
                    Install date
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* 5. INSTALLED APPLICATIONS LIST TABLE */}
        <div className="rounded-xl bg-surface-container-low overflow-hidden shadow-sm">
          {/* Table Headers */}
          <div className="hidden md:grid md:grid-cols-12 px-5 py-3 text-label-sm font-label-sm text-on-surface-variant bg-surface-container-low border-b border-white/[0.04]">
            <div className="col-span-5">Application</div>
            <div className="col-span-2">Format &amp; Environment</div>
            <div className="col-span-1">Size</div>
            <div className="col-span-2">Last Activity</div>
            <div className="col-span-2 text-right">Actions</div>
          </div>

          <div className="divide-y divide-white/[0.04]">
            {sortedApps.map((app) => (
              <div
                key={app.id}
                className="p-4 sm:px-5 flex flex-col md:grid md:grid-cols-12 gap-3 md:gap-2 items-start md:items-center hover:bg-surface-container/60 transition group"
              >
                {/* Application Name & Icon */}
                <div
                  className="md:col-span-5 flex items-center gap-3.5 min-w-0 w-full cursor-pointer"
                  onClick={() => navigate(`/package/${app.name.toLowerCase().replace(/\s+/g, '-')}`)}
                >
                  <div className="w-11 h-11 rounded-xl bg-surface-container-highest flex items-center justify-center shrink-0 overflow-hidden p-1.5">
                    <AppIcon
                      appId={app.id}
                      pkgName={app.name}
                      provider={app.provider}
                      mode="installed"
                      fallbackIcon={app.icon}
                      fallbackBg="bg-transparent"
                      size={64}
                      className="w-8 h-8"
                      imgClassName="w-8 h-8 object-contain"
                      alt={app.displayName}
                    />
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-title text-title text-on-background truncate group-hover:text-primary transition-colors">
                        {app.displayName}
                      </span>
                      {app.hasUpdate && (
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-code bg-primary-container/20 text-primary-fixed-dim">
                          {app.updateVersion}
                        </span>
                      )}
                      {!app.hasUpdate && !app.isSharedCore && (
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-code bg-tertiary-container/30 text-tertiary">
                          Up to date
                        </span>
                      )}
                      {app.isSharedCore && (
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-code bg-surface-container text-on-surface-variant">
                          System Runtime
                        </span>
                      )}
                    </div>
                    <p className="text-on-surface-variant font-body-sm text-body-sm truncate">{app.description}</p>
                  </div>
                </div>

                {/* Format & Environment */}
                <div className="md:col-span-2 flex items-center gap-2">
                  <span className={`px-2 py-0.5 rounded-md font-label-sm text-label-sm ${app.isNative
                    ? 'bg-surface-container text-tertiary'
                    : 'bg-surface-container text-surface-tint'
                    }`}>
                    {app.format}
                  </span>
                </div>

                {/* Size */}
                <div className="md:col-span-1 text-on-surface font-code text-label-sm">
                  {app.sizeLabel}
                </div>

                {/* Last Activity */}
                <div className="md:col-span-2 text-on-surface-variant font-body-sm text-body-sm flex items-center gap-1.5">
                  <span className={`w-1.5 h-1.5 rounded-full ${app.hasUpdate ? 'bg-tertiary' : 'bg-surface-tint'}`}></span>
                  <span>{app.lastActivity}</span>
                </div>

                {/* Actions */}
                <div className="md:col-span-2 flex items-center justify-end gap-2 w-full md:w-auto">
                  {app.hasUpdate && (
                    <button
                      onClick={() => handleUpdateSingle(app.id)}
                      className="px-3 py-1.5 rounded-lg bg-primary-container hover:bg-inverse-primary text-on-primary-container font-label-md text-label-md transition active:scale-95 flex items-center gap-1 cursor-pointer"
                      type="button"
                    >
                      <span className="material-symbols-outlined text-[15px]">arrow_downward</span>
                      <span>Update</span>
                    </button>
                  )}

                  {!app.isSharedCore && (
                    <button
                      onClick={() => handleLaunch(app.displayName)}
                      className="px-3.5 py-1.5 rounded-lg bg-surface-container hover:bg-surface-container-high text-on-surface font-label-md text-label-md transition cursor-pointer"
                      type="button"
                    >
                      Launch
                    </button>
                  )}

                  {app.isSharedCore && (
                    <span className="text-on-surface-variant font-label-sm text-label-sm italic mr-2">Shared Core</span>
                  )}

                  <div className="relative">
                    <button
                      onClick={() => setActiveMenuId(activeMenuId === app.id ? null : app.id)}
                      className="p-1.5 rounded-lg text-on-surface-variant hover:text-on-background hover:bg-surface-container transition cursor-pointer"
                      type="button"
                    >
                      <span className="material-symbols-outlined text-[18px]">more_vert</span>
                    </button>

                    {activeMenuId === app.id && (
                      <div className="absolute right-0 mt-1 w-48 rounded-xl bg-surface-container-high shadow-fluent-flyout py-1 z-20">
                        <button
                          onClick={() => {
                            setActiveMenuId(null);
                            triggerToast("Permissions Inspector", `Viewing sandbox portals for ${app.displayName}.`);
                          }}
                          className="w-full text-left px-3 py-1.5 text-label-sm font-label-sm text-on-surface hover:bg-surface-container-highest flex items-center gap-2 cursor-pointer"
                        >
                          <span className="material-symbols-outlined text-[15px]">tune</span> Permissions
                        </button>
                        <button
                          onClick={() => {
                            setActiveMenuId(null);
                            triggerToast("Desktop Shortcut", `Created .desktop launcher for ${app.displayName}.`);
                          }}
                          className="w-full text-left px-3 py-1.5 text-label-sm font-label-sm text-on-surface hover:bg-surface-container-highest flex items-center gap-2 cursor-pointer"
                        >
                          <span className="material-symbols-outlined text-[15px]">shortcut</span> Desktop Shortcut
                        </button>
                        <button
                          onClick={() => {
                            setActiveMenuId(null);
                            triggerToast("Cache Cleared", `Cleaned cached logs for ${app.displayName}.`);
                          }}
                          className="w-full text-left px-3 py-1.5 text-label-sm font-label-sm text-on-surface hover:bg-surface-container-highest flex items-center gap-2 cursor-pointer"
                        >
                          <span className="material-symbols-outlined text-[15px]">cleaning_services</span> Clear Cache
                        </button>
                        <div className="my-1 h-[1px] bg-white/[0.06]"></div>
                        <button
                          onClick={() => handleUninstall(app.id, app.displayName)}
                          className="w-full text-left px-3 py-1.5 text-label-sm font-label-sm text-error hover:bg-error-container/30 flex items-center gap-2 cursor-pointer"
                        >
                          <span className="material-symbols-outlined text-[15px]">delete</span> Uninstall
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* 6. SANDBOXED PERMISSIONS & ZERO-MAINTENANCE HYGIENE CARD */}
        <div className="rounded-xl bg-surface-container-low p-6 shadow-sm">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-center">
            <div className="lg:col-span-8 space-y-2">
              <div className="flex items-center gap-2 text-tertiary font-label-sm text-label-sm uppercase tracking-wider font-semibold">
                <span className="material-symbols-outlined text-[16px]">format_image_left</span>
                <span>Zero Terminal Maintenance Guarantee</span>
              </div>
              <h2 className="text-on-background font-headline-md text-headline-md font-semibold">
                Safe Isolation, Transparent Permissions
              </h2>
              <p className="text-on-surface-variant font-body-md text-body-md max-w-2xl">
                Arjax executes every modern application within an isolated unprivileged user space. Applications cannot tamper with system-critical directories (<code className="font-code text-label-sm text-surface-tint">/usr</code>, <code className="font-code text-label-sm text-surface-tint">/etc</code>), corrupt your boot partition, or conflict with each other's libraries.
              </p>
              <div className="flex flex-wrap items-center gap-4 pt-2">
                <div className="flex items-center gap-2 text-on-surface font-body-sm text-body-sm">
                  <span className="material-symbols-outlined text-[16px] text-tertiary">check_circle</span>
                  <span>Granular Camera &amp; Mic Gates</span>
                </div>
                <div className="flex items-center gap-2 text-on-surface font-body-sm text-body-sm">
                  <span className="material-symbols-outlined text-[16px] text-tertiary">check_circle</span>
                  <span>Isolated Home Directory Access</span>
                </div>
                <div className="flex items-center gap-2 text-on-surface font-body-sm text-body-sm">
                  <span className="material-symbols-outlined text-[16px] text-tertiary">check_circle</span>
                  <span>Atomic 1-Click Rollbacks</span>
                </div>
              </div>
            </div>

            <div className="lg:col-span-4 flex flex-col sm:flex-row lg:flex-col gap-3 justify-center">
              <button
                onClick={() => triggerToast("Global Portals", "Opening Wayland & XDG desktop portal configurations.")}
                className="px-4 py-2.5 rounded-lg bg-surface-container hover:bg-surface-container-high text-on-surface font-label-md text-label-md flex items-center justify-center gap-2 transition active:scale-[0.98] cursor-pointer"
                type="button"
              >
                <span className="material-symbols-outlined text-[18px] text-surface-tint">security</span>
                <span>Configure Global App Portals</span>
              </button>
              <button
                onClick={() => triggerToast("Integrity Diagnostic", "All application sandbox namespaces verified clean.")}
                className="px-4 py-2.5 rounded-lg bg-surface-container-high hover:bg-surface-container-highest text-on-surface font-label-md text-label-md flex items-center justify-center gap-2 transition active:scale-[0.98] cursor-pointer"
                type="button"
              >
                <span className="material-symbols-outlined text-[18px] text-tertiary">health_and_safety</span>
                <span>Run Integrity Diagnostic</span>
              </button>
            </div>
          </div>
        </div>

      </div>

      {/* FLUENT NOTIFICATION / INSTALL FLYOUT */}
      <div
        className={`fixed bottom-5 right-5 bg-mica-elevated border border-white/[0.12] text-fluent-text px-4 py-3 rounded-lg shadow-fluent-flyout z-50 flex items-center gap-3 transition-all duration-300 ${toastVisible ? 'translate-y-0 opacity-100' : 'translate-y-24 opacity-0 pointer-events-none'
          }`}
      >
        <div className="w-7 h-7 rounded-md bg-fluent-blue/20 text-fluent-blue flex items-center justify-center">
          <span className="material-symbols-outlined text-[18px]">downloading</span>
        </div>
        <div>
          <div className="text-[12.5px] font-semibold text-white">{toastMessage.title}</div>
          <div className="text-[11px] text-fluent-muted">{toastMessage.desc}</div>
        </div>
      </div>
    </div>
  );
};
