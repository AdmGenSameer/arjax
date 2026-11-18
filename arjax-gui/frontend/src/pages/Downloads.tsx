import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

interface QueuedItem {
  id: string;
  name: string;
  version: string;
  size: string;
  icon: string;
  iconBg: string;
  iconColor: string;
  tag?: string;
  tagColor?: string;
  desc: string;
  techDetail: string;
}

interface CompletedItem {
  id: string;
  name: string;
  version: string;
  size: string;
  timeAgo: string;
  icon: string;
  iconBg: string;
  iconColor: string;
  tag?: string;
  hasPatchNotes?: boolean;
}

export const DownloadsPage: React.FC = () => {
  const navigate = useNavigate();

  // State
  const [isPausedAll, setIsPausedAll] = useState(false);
  const [downloadSpeed, setDownloadSpeed] = useState(18.4);
  const [isCheckingUpdates, setIsCheckingUpdates] = useState(false);
  const [activeDownloadPaused, setActiveDownloadPaused] = useState(false);
  const [activeProgress, setActiveProgress] = useState(70);

  // Toggles
  const [autoUpdates, setAutoUpdates] = useState(true);
  const [meteredShield, setMeteredShield] = useState(true);

  // Toast
  const [toastVisible, setToastVisible] = useState(false);
  const [toastMessage, setToastMessage] = useState({ title: '', desc: '' });

  const triggerToast = (title: string, desc: string, duration = 2400) => {
    setToastMessage({ title, desc });
    setToastVisible(true);
    setTimeout(() => {
      setToastVisible(false);
    }, duration);
  };

  // Up Next Queue
  const [queue, setQueue] = useState<QueuedItem[]>([
    {
      id: 'brave',
      name: 'Brave Browser',
      version: 'v1.65.130',
      size: '84.2 MB',
      icon: 'public',
      iconBg: 'bg-on-secondary-container/20',
      iconColor: 'text-secondary-fixed',
      desc: 'Starts automatically after Blender LTS',
      techDetail: 'Flatpak delta update'
    },
    {
      id: 'discord',
      name: 'Discord',
      version: 'v0.0.48',
      size: '18.5 MB',
      icon: 'forum',
      iconBg: 'bg-primary-container/20',
      iconColor: 'text-primary',
      tag: 'Security Patch',
      tagColor: 'bg-error-container/40 text-error',
      desc: 'Pending slot in transfer pipe',
      techDetail: 'Electron runtime patch'
    }
  ]);

  // Completed Today
  const [completed, setCompleted] = useState<CompletedItem[]>([
    {
      id: 'vscode',
      name: 'Visual Studio Code',
      version: '1.89.1',
      size: '68.2 MB',
      timeAgo: 'Updated 25 mins ago',
      icon: 'terminal',
      iconBg: 'bg-primary-container/20',
      iconColor: 'text-primary',
      hasPatchNotes: true
    },
    {
      id: 'vlc',
      name: 'VLC Media Player',
      version: '3.0.21',
      size: '24.1 MB',
      timeAgo: 'Updated 2 hours ago',
      icon: 'play_circle',
      iconBg: 'bg-secondary-container/20',
      iconColor: 'text-secondary',
      hasPatchNotes: true
    },
    {
      id: 'obs',
      name: 'OBS Studio',
      version: '30.1.2',
      size: '112.0 MB',
      timeAgo: 'Updated 5 hours ago',
      icon: 'videocam',
      iconBg: 'bg-surface-bright',
      iconColor: 'text-white'
    },
    {
      id: 'gnome',
      name: 'GNOME 46 Platform Runtime',
      version: '46.2',
      size: '420 MB',
      timeAgo: 'Updated 6 hours ago',
      icon: 'layers',
      iconBg: 'bg-surface-bright',
      iconColor: 'text-tertiary',
      tag: 'Core Dependency'
    }
  ]);

  // Progress simulation & real-time Wails event binding
  useEffect(() => {
    // Listen to real-time events from Go provider engine
    const wailsRuntime = (window as any).runtime;
    if (wailsRuntime && wailsRuntime.EventsOn) {
      const cancelSub = wailsRuntime.EventsOn('install-progress', (data: any) => {
        if (data) {
          if (data.progress) setActiveProgress(data.progress);
          if (data.stage) {
            triggerToast(data.stage, data.packageName ? `Target: ${data.packageName}` : '');
          }
          if (data.completed && data.success) {
            setCompleted(prev => [
              {
                id: data.packageName || 'app',
                name: data.packageName || 'Application',
                version: 'Latest',
                size: 'Installed',
                timeAgo: 'Just now',
                icon: 'check_circle',
                iconBg: 'bg-emerald-500/20',
                iconColor: 'text-emerald-400',
              },
              ...prev
            ]);
          }
        }
      });
      return () => {
        if (typeof cancelSub === 'function') cancelSub();
      };
    }
  }, []);

  useEffect(() => {
    if (isPausedAll || activeDownloadPaused) return;

    const timer = setInterval(() => {
      setActiveProgress((prev) => {
        if (prev >= 98) return 98;
        return prev + 1;
      });
    }, 1800);

    return () => clearInterval(timer);
  }, [isPausedAll, activeDownloadPaused]);

  // Handlers
  const handleTogglePauseAll = () => {
    const nextState = !isPausedAll;
    setIsPausedAll(nextState);
    if (nextState) {
      setDownloadSpeed(0.0);
      triggerToast('All Transfers Paused', 'Network bandwidth released for other local processes.');
    } else {
      setDownloadSpeed(18.4);
      triggerToast('Transfers Resumed', 'Resuming active transfer pipelines.');
    }
  };

  const handleCheckUpdates = async () => {
    if (isCheckingUpdates) return;
    setIsCheckingUpdates(true);
    triggerToast('Checking remote repositories...', 'Querying Flathub and Arch mirror indexes.');

    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.CheckForUpdates) {
      try {
        const updates = await goApp.CheckForUpdates();
        if (Array.isArray(updates) && updates.length > 0) {
          triggerToast('Repositories Synchronized', `Found ${updates.length} updates ready.`);
        }
      } catch (e) {
        console.warn(e);
      }
    }
    setTimeout(() => {
      setIsCheckingUpdates(false);
    }, 1200);
  };

  const handleMoveToTop = (id: string) => {
    setQueue((prev) => {
      const idx = prev.findIndex((item) => item.id === id);
      if (idx <= 0) return prev;
      const item = prev[idx];
      const next = [...prev];
      next.splice(idx, 1);
      next.unshift(item);
      return next;
    });
    triggerToast('Queue Priority Updated', 'Moved package to head of download queue.');
  };

  const handleDownloadNow = async (item: QueuedItem) => {
    setQueue((prev) => prev.filter((q) => q.id !== item.id));
    triggerToast(`Starting download: ${item.name}`, 'Allocating high-priority transfer slot.');

    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.InstallPackage) {
      try {
        await goApp.InstallPackage(item.id, 'flatpak');
      } catch (e) {
        console.warn(e);
      }
    }
  };

  const handleRemoveFromQueue = (id: string, name: string) => {
    setQueue((prev) => prev.filter((item) => item.id !== id));
    triggerToast(`Cancelled: ${name}`, 'Removed package from pending queue.');
  };

  const handleLaunch = (name: string) => {
    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.LaunchApp) {
      goApp.LaunchApp(name);
    }
    triggerToast(`Launching ${name}...`, 'Initializing containerized application.');
  };

  return (
    <div className="flex flex-col w-full font-sans">
      <div className="p-space-lg max-w-[1360px] mx-auto w-full space-y-space-xl">

        {/* Top Action & Title Bar */}
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-space-lg">
          <div className="space-y-space-xs">
            <div className="flex items-center gap-space-sm">
              <h1 className="font-headline-lg text-headline-lg text-white tracking-tight">Downloads &amp; Updates</h1>
              <span className="px-2 py-0.5 rounded-full font-label-sm text-label-sm bg-primary-container text-white shadow-sm font-medium">
                3 Active
              </span>
            </div>
            <p className="font-body-md text-body-md text-on-surface-variant max-w-2xl">
              Manage package transfers, silent background security updates, and isolated sandboxed installations.
            </p>
          </div>

          {/* Global Actions */}
          <div className="flex items-center flex-wrap gap-space-sm">
            <button
              onClick={handleCheckUpdates}
              disabled={isCheckingUpdates}
              className="flex items-center gap-space-xs px-space-md py-2 rounded-lg bg-surface-container-high hover:bg-surface-bright text-on-surface font-label-md text-label-md transition shadow-sm active:scale-95 cursor-pointer"
              id="btn-check-updates"
            >
              <span className={`material-symbols-outlined text-[17px] transition-transform duration-500 ${isCheckingUpdates ? 'rotate-180 animate-spin' : ''}`}>
                sync
              </span>
              <span>{isCheckingUpdates ? 'Checking...' : 'Check for Updates'}</span>
            </button>

            <button
              onClick={handleTogglePauseAll}
              className="flex items-center gap-space-xs px-space-md py-2 rounded-lg bg-surface-container-high hover:bg-surface-bright text-on-surface font-label-md text-label-md transition shadow-sm active:scale-95 cursor-pointer"
              id="btn-pause-all"
            >
              <span className="material-symbols-outlined text-[17px]">
                {isPausedAll ? 'play_arrow' : 'pause'}
              </span>
              <span>{isPausedAll ? 'Resume All' : 'Pause All'}</span>
            </button>

            <button
              onClick={() => triggerToast('Updating all packages...', 'Downloading 2 pending updates in background.')}
              className="flex items-center gap-space-xs px-space-lg py-2 rounded-lg bg-primary-container hover:bg-[#1084d9] text-white font-label-md text-label-md shadow-md transition active:scale-95 cursor-pointer"
            >
              <span className="material-symbols-outlined text-[17px]">download</span>
              <span>Update All (2 Pending)</span>
            </button>
          </div>
        </div>

        {/* Stat Strip (4 Metric Tiles) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-space-md">
          {/* Tile 1: Speed */}
          <div className="p-space-md rounded-xl bg-surface-container/60 backdrop-blur-md shadow-sm flex flex-col justify-between">
            <div className="flex items-center justify-between text-on-surface-variant mb-space-xs">
              <span className="font-label-sm text-label-sm">Download Speed</span>
              <span className="material-symbols-outlined text-primary text-[18px]">arrow_circle_up</span>
            </div>
            <div className="flex items-baseline gap-space-xs">
              <span className="font-headline-md text-headline-md text-white font-semibold">
                {downloadSpeed.toFixed(1)}
              </span>
              <span className="font-label-md text-label-md text-on-surface-variant">MB/s</span>
            </div>
            <div className="mt-space-xs flex items-center justify-between">
              <span className="font-body-sm text-body-sm text-on-surface-variant">Peak: 24.2 MB/s</span>
              {/* Inline Sparkline Chart */}
              <svg className="w-16 h-4 text-primary overflow-visible" fill="none" stroke="currentColor" strokeWidth="1.8">
                <polyline points="0,14 10,12 20,13 30,7 40,9 50,3 64,4"></polyline>
              </svg>
            </div>
          </div>

          {/* Tile 2: Queue Overview */}
          <div className="p-space-md rounded-xl bg-surface-container/60 backdrop-blur-md shadow-sm flex flex-col justify-between">
            <div className="flex items-center justify-between text-on-surface-variant mb-space-xs">
              <span className="font-label-sm text-label-sm">Queue Status</span>
              <span className="material-symbols-outlined text-tertiary text-[18px]">list_alt</span>
            </div>
            <div className="font-title text-title text-white">
              1 Downloading • {queue.length} Queued
            </div>
            <div className="mt-space-xs flex items-center gap-space-xs text-on-surface-variant font-body-sm text-body-sm">
              <span className="w-2 h-2 rounded-full bg-tertiary"></span>
              <span>{completed.length} Completed today</span>
            </div>
          </div>

          {/* Tile 3: Bandwidth Mode */}
          <div className="p-space-md rounded-xl bg-surface-container/60 backdrop-blur-md shadow-sm flex flex-col justify-between">
            <div className="flex items-center justify-between text-on-surface-variant mb-space-xs">
              <span className="font-label-sm text-label-sm">Bandwidth Allocation</span>
              <span className="material-symbols-outlined text-secondary text-[18px]">speed</span>
            </div>
            <div className="font-title text-title text-white truncate">
              Unlimited
            </div>
            <div className="mt-space-xs flex items-center justify-between">
              <span className="font-body-sm text-body-sm text-on-surface-variant">Low Latency Mode</span>
              <button
                onClick={() => triggerToast('Bandwidth Throttle', 'Bandwidth allocation is managed automatically.')}
                className="font-label-sm text-label-sm text-primary hover:underline cursor-pointer"
              >
                Adjust
              </button>
            </div>
          </div>

          {/* Tile 4: Disk Activity */}
          <div className="p-space-md rounded-xl bg-surface-container/60 backdrop-blur-md shadow-sm flex flex-col justify-between">
            <div className="flex items-center justify-between text-on-surface-variant mb-space-xs">
              <span className="font-label-sm text-label-sm">Disk I/O Write</span>
              <span className="material-symbols-outlined text-outline text-[18px]">hard_drive</span>
            </div>
            <div className="flex items-baseline gap-space-xs">
              <span className="font-headline-md text-headline-md text-white font-semibold">42.1</span>
              <span className="font-label-md text-label-md text-on-surface-variant">MB/s</span>
            </div>
            <div className="mt-space-xs text-on-surface-variant font-body-sm text-body-sm truncate">
              Direct NVMe Stream (/var/lib/flatpak)
            </div>
          </div>
        </div>

        {/* Active Transfers (In Progress) */}
        <div className="space-y-space-md">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-space-sm">
              <span className="material-symbols-outlined text-primary text-[20px]">cloud_sync</span>
              <h2 className="font-title text-title text-white">In Progress</h2>
            </div>
            <span className="font-label-sm text-label-sm text-on-surface-variant">High Priority Channel</span>
          </div>

          {/* Active Card: Blender 4.2.2 LTS */}
          <div className="rounded-xl bg-surface-container/80 backdrop-blur-xl p-space-lg shadow-md transition-all hover:bg-surface-container-high/80 relative overflow-hidden group">
            {/* Ambient background soft accent */}
            <div className="absolute -right-16 -top-16 w-56 h-56 bg-primary-container/10 rounded-full blur-3xl pointer-events-none"></div>

            <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-space-lg">
              <div className="flex items-start gap-space-md min-w-0">
                {/* App Icon with Squircle Frame */}
                <div className="w-14 h-14 rounded-xl bg-secondary-container/20 flex items-center justify-center shrink-0 shadow-sm relative overflow-hidden">
                  <span className="material-symbols-outlined text-secondary text-[32px]">view_in_ar</span>
                </div>
                <div className="min-w-0 space-y-1">
                  <div className="flex items-center flex-wrap gap-space-xs">
                    <span
                      onClick={() => navigate('/package/blender')}
                      className="font-title text-title text-white hover:text-primary transition cursor-pointer"
                    >
                      Blender 4.2.2 LTS
                    </span>
                    <span className="px-2 py-0.5 rounded font-label-sm text-label-sm bg-surface-bright text-on-surface-variant">
                      New Installation
                    </span>
                  </div>
                  <p className="font-body-sm text-body-sm text-on-surface-variant truncate">
                    Professional 3D computer graphics software, modeling, animation, rendering, and compositing pipeline.
                  </p>
                  {/* Safe, plain-language reassurance badges */}
                  <div className="flex items-center flex-wrap gap-space-xs pt-1">
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded font-label-sm text-label-sm bg-primary-container/15 text-primary">
                      <span className="material-symbols-outlined text-[13px]">verified_user</span>
                      Verified Publisher
                    </span>
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded font-label-sm text-label-sm bg-white/[0.05] text-on-surface-variant">
                      <span className="material-symbols-outlined text-[13px]">shield</span>
                      Flatpak Sandboxed
                    </span>
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded font-label-sm text-label-sm bg-white/[0.05] text-on-surface-variant">
                      <span className="material-symbols-outlined text-[13px]">cloud</span>
                      Direct Flathub CDN
                    </span>
                  </div>
                </div>
              </div>

              {/* Action Controls for Active Item */}
              <div className="flex items-center gap-space-xs shrink-0 self-end md:self-center">
                <button
                  onClick={() => setActiveDownloadPaused(!activeDownloadPaused)}
                  className="p-2 rounded-lg bg-surface-bright hover:bg-surface-variant text-on-surface hover:text-white transition cursor-pointer"
                  title={activeDownloadPaused ? "Resume Download" : "Pause Download"}
                >
                  <span className="material-symbols-outlined text-[18px]">
                    {activeDownloadPaused ? 'play_arrow' : 'pause'}
                  </span>
                </button>
                <button
                  onClick={() => triggerToast('Cancelled Download', 'Removed Blender from downloads.')}
                  className="p-2 rounded-lg bg-surface-bright hover:bg-error-container hover:text-error text-on-surface transition cursor-pointer"
                  title="Cancel Download"
                >
                  <span className="material-symbols-outlined text-[18px]">close</span>
                </button>
              </div>
            </div>

            {/* Progress Details & Visual Fluent Bar */}
            <div className="mt-space-lg space-y-space-xs">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-1 font-body-sm text-body-sm">
                <span className="text-white font-medium">
                  {activeDownloadPaused ? (
                    <span className="text-secondary">Download paused (70%)</span>
                  ) : (
                    <>
                      Downloading packages... <span className="text-primary font-code">218.4 MB</span> of 312.0 MB ({activeProgress}%)
                    </>
                  )}
                </span>
                <div className="flex items-center gap-space-md text-on-surface-variant">
                  <span>{activeDownloadPaused ? '0.0 MB/s' : '18.4 MB/s'}</span>
                  <span className="w-1 h-1 rounded-full bg-outline"></span>
                  <span>{activeDownloadPaused ? 'Paused' : '~5 seconds remaining'}</span>
                </div>
              </div>

              {/* Fluent Progress Bar */}
              <div className="w-full h-2 rounded-full bg-surface-bright overflow-hidden relative">
                <div
                  className="h-full bg-gradient-to-r from-primary-container via-primary to-primary-container rounded-full transition-all duration-300"
                  style={{ width: `${activeProgress}%` }}
                ></div>
              </div>

              {/* Three Stage Micro-Timeline (Human Readable Pipeline) */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-space-sm pt-space-xs">
                <div className="flex items-center gap-space-xs font-label-sm text-label-sm text-tertiary">
                  <span className="material-symbols-outlined text-[15px]">check_circle</span>
                  <span>1. Cryptographic Checksum Verified</span>
                </div>
                <div className="flex items-center gap-space-xs font-label-sm text-label-sm text-primary font-medium animate-pulse">
                  <span className="material-symbols-outlined text-[15px]">downloading</span>
                  <span>2. Fetching FreeDesktop Runtime</span>
                </div>
                <div className="flex items-center gap-space-xs font-label-sm text-label-sm text-on-surface-variant">
                  <span className="material-symbols-outlined text-[15px]">hourglass_empty</span>
                  <span>3. Sandboxed Container Setup</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Up Next Queue */}
        <div className="space-y-space-md">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-space-sm">
              <span className="material-symbols-outlined text-secondary text-[20px]">queue_music</span>
              <h2 className="font-title text-title text-white">Up Next • {queue.length} Queued</h2>
            </div>
            <button
              onClick={() => triggerToast('Queue auto-balanced', 'Priority queue is optimized for sequential execution.')}
              className="font-label-sm text-label-sm text-on-surface-variant hover:text-white transition cursor-pointer"
            >
              Reorder Priority
            </button>
          </div>

          <div className="space-y-space-xs">
            {queue.map((item) => (
              <div
                key={item.id}
                className="p-space-md rounded-xl bg-surface-container/60 backdrop-blur-md flex flex-col md:flex-row md:items-center justify-between gap-space-md hover:bg-surface-container-high/60 transition group"
              >
                <div className="flex items-center gap-space-md min-w-0">
                  <div className={`w-11 h-11 rounded-xl ${item.iconBg} flex items-center justify-center shrink-0`}>
                    <span className={`material-symbols-outlined ${item.iconColor} text-[26px]`}>{item.icon}</span>
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-space-xs">
                      <span className="font-title text-title text-white">{item.name}</span>
                      <span className="font-code text-code text-on-surface-variant">{item.version}</span>
                      <span className="px-1.5 py-0.5 rounded font-label-sm text-label-sm bg-white/[0.05] text-on-surface-variant">
                        {item.size}
                      </span>
                      {item.tag && (
                        <span className={`px-2 py-0.5 rounded font-label-sm text-label-sm ${item.tagColor} font-medium`}>
                          {item.tag}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-space-sm text-on-surface-variant font-body-sm text-body-sm mt-0.5">
                      <span className="text-secondary font-medium">Queued</span>
                      <span>•</span>
                      <span className="truncate">{item.desc}</span>
                      <span>•</span>
                      <span className="hidden sm:inline text-outline">{item.techDetail}</span>
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-space-xs shrink-0 self-end md:self-center">
                  <button
                    onClick={() => handleMoveToTop(item.id)}
                    className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-surface-bright hover:bg-surface-variant text-on-surface hover:text-white font-label-md text-label-md transition cursor-pointer"
                  >
                    <span className="material-symbols-outlined text-[16px]">arrow_upward</span>
                    <span className="hidden sm:inline">Move to Top</span>
                  </button>
                  <button
                    onClick={() => handleDownloadNow(item)}
                    className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-primary-container/20 hover:bg-primary-container text-primary hover:text-white font-label-md text-label-md transition cursor-pointer"
                  >
                    <span className="material-symbols-outlined text-[16px]">play_arrow</span>
                    <span>Download Now</span>
                  </button>
                  <button
                    onClick={() => handleRemoveFromQueue(item.id, item.name)}
                    className="p-1.5 rounded-lg text-on-surface-variant hover:text-error hover:bg-error-container/20 transition cursor-pointer"
                    title="Remove from queue"
                  >
                    <span className="material-symbols-outlined text-[18px]">close</span>
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Recently Completed Today */}
        <div className="space-y-space-md">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-space-sm">
              <span className="material-symbols-outlined text-tertiary text-[20px]">task_alt</span>
              <h2 className="font-title text-title text-white">Completed Today ({completed.length} Items)</h2>
            </div>
            <button
              onClick={() => {
                setCompleted([]);
                triggerToast('History Cleared', 'Download history cleared for today.');
              }}
              className="font-label-sm text-label-sm text-on-surface-variant hover:text-white transition flex items-center gap-1 cursor-pointer"
            >
              <span className="material-symbols-outlined text-[15px]">clear_all</span>
              <span>Clear History</span>
            </button>
          </div>

          <div className="rounded-xl bg-surface-container/60 backdrop-blur-md overflow-hidden divide-y divide-white/[0.04]">
            {completed.map((item) => (
              <div
                key={item.id}
                className="p-space-md flex flex-col sm:flex-row sm:items-center justify-between gap-space-md hover:bg-surface-bright/40 transition"
              >
                <div className="flex items-center gap-space-md min-w-0">
                  <div className={`w-10 h-10 rounded-lg ${item.iconBg} flex items-center justify-center shrink-0`}>
                    <span className={`material-symbols-outlined ${item.iconColor} text-[24px]`}>{item.icon}</span>
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-space-xs">
                      <span className="font-title text-title text-white">{item.name}</span>
                      <span className="font-code text-code text-on-surface-variant">{item.version}</span>
                      {item.tag && (
                        <span className="px-2 py-0.5 rounded font-label-sm text-label-sm bg-surface-variant text-on-surface-variant">
                          {item.tag}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-space-sm font-body-sm text-body-sm text-on-surface-variant mt-0.5">
                      <span className="inline-flex items-center gap-1 text-tertiary">
                        <span className="material-symbols-outlined text-[14px]">check</span>
                        Installed Successfully
                      </span>
                      <span>•</span>
                      <span>{item.timeAgo}</span>
                      <span>•</span>
                      <span>{item.size}</span>
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-space-xs self-end sm:self-center shrink-0">
                  {item.hasPatchNotes && (
                    <button
                      onClick={() => triggerToast(`Patch Notes: ${item.name}`, `Version ${item.version} bugfixes & improvements.`)}
                      className="px-3 py-1.5 rounded-lg bg-surface-bright hover:bg-surface-variant text-on-surface font-label-md text-label-md transition cursor-pointer"
                    >
                      Patch Notes
                    </button>
                  )}
                  {item.tag ? (
                    <span className="font-body-sm text-body-sm text-outline px-space-sm">Zero action needed</span>
                  ) : (
                    <button
                      onClick={() => handleLaunch(item.name)}
                      className="px-3.5 py-1.5 rounded-lg bg-primary-container hover:bg-[#1084d9] text-white font-label-md text-label-md transition shadow-sm cursor-pointer"
                    >
                      Launch
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Beginner Peace-of-Mind & Atomic Rollback Banner */}
        <div className="rounded-xl bg-gradient-to-r from-surface-container via-surface-container-high to-surface-container p-space-lg shadow-md relative overflow-hidden">
          {/* Shield accent graphic */}
          <div className="absolute right-4 -bottom-6 opacity-10 pointer-events-none">
            <span className="material-symbols-outlined text-[160px] text-tertiary">verified</span>
          </div>

          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-space-lg relative z-10">
            <div className="flex items-start gap-space-md max-w-3xl">
              <div className="w-12 h-12 rounded-xl bg-tertiary/15 flex items-center justify-center shrink-0 shadow-sm mt-0.5">
                <span className="material-symbols-outlined text-tertiary text-[28px]">security</span>
              </div>
              <div className="space-y-1">
                <div className="flex items-center gap-space-sm">
                  <h3 className="font-title text-title text-white">Seamless Background Updates &amp; Atomic Rollbacks</h3>
                  <span className="px-2 py-0.5 rounded font-label-sm text-label-sm bg-tertiary/20 text-tertiary">
                    Safety Shield Active
                  </span>
                </div>
                <p className="font-body-md text-body-md text-on-surface-variant">
                  Arjax applications run in isolated containers. You never need to reboot your system or close your work while updates apply. If an update ever behaves unexpectedly, 1-Click Rollback restores the previous stable version in under two seconds.
                </p>
              </div>
            </div>

            {/* Quick Toggles */}
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-space-sm shrink-0">
              <div className="flex items-center justify-between sm:justify-start gap-space-sm px-space-md py-2 rounded-lg bg-black/20">
                <span className="font-label-sm text-label-sm text-on-surface">Auto-Updates</span>
                <button
                  onClick={() => {
                    const next = !autoUpdates;
                    setAutoUpdates(next);
                    triggerToast('Auto-Updates', next ? 'Background updates enabled.' : 'Background updates paused.');
                  }}
                  aria-label="Toggle Auto-Updates"
                  className={`w-9 h-5 rounded-full relative p-0.5 transition cursor-pointer ${
                    autoUpdates ? 'bg-primary-container' : 'bg-surface-variant'
                  }`}
                  type="button"
                >
                  <div
                    className={`w-4 h-4 rounded-full bg-white transition-transform ${
                      autoUpdates ? 'translate-x-4' : 'translate-x-0'
                    }`}
                  ></div>
                </button>
              </div>

              <div className="flex items-center justify-between sm:justify-start gap-space-sm px-space-md py-2 rounded-lg bg-black/20">
                <span className="font-label-sm text-label-sm text-on-surface">Metered Data Shield</span>
                <button
                  onClick={() => {
                    const next = !meteredShield;
                    setMeteredShield(next);
                    triggerToast('Metered Data Shield', next ? 'Transfers paused over metered networks.' : 'Metered shield disabled.');
                  }}
                  aria-label="Toggle Metered Data Shield"
                  className={`w-9 h-5 rounded-full relative p-0.5 transition cursor-pointer ${
                    meteredShield ? 'bg-primary-container' : 'bg-surface-variant'
                  }`}
                  type="button"
                >
                  <div
                    className={`w-4 h-4 rounded-full bg-white transition-transform ${
                      meteredShield ? 'translate-x-4' : 'translate-x-0'
                    }`}
                  ></div>
                </button>
              </div>
            </div>
          </div>
        </div>

      </div>

      {/* FLUENT NOTIFICATION / INSTALL FLYOUT */}
      <div
        className={`fixed bottom-5 right-5 bg-surface-container-high border border-white/[0.12] text-white px-4 py-3 rounded-lg shadow-fluent-flyout z-50 flex items-center gap-3 transition-all duration-300 ${
          toastVisible ? 'translate-y-0 opacity-100' : 'translate-y-24 opacity-0 pointer-events-none'
        }`}
      >
        <div className="w-7 h-7 rounded-md bg-primary-container/20 text-primary flex items-center justify-center">
          <span className="material-symbols-outlined text-[18px]">downloading</span>
        </div>
        <div>
          <div className="text-[12.5px] font-semibold text-white">{toastMessage.title}</div>
          <div className="text-[11px] text-on-surface-variant">{toastMessage.desc}</div>
        </div>
      </div>
    </div>
  );
};

