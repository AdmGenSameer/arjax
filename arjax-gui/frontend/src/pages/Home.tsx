import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { HERO_SLIDES, APPS_DATA } from '../data/appsData';
import { AppIcon } from '../components/AppIcon';

interface AppItem {
  id: string;
  name: string;
  category: string;
  desc: string;
  rating: number;
  reviews?: string;
  size: string;
  provider: string;
  iconBg: string;
  iconName: string;
}

export const Home: React.FC = () => {
  const navigate = useNavigate();

  // Toast flyout feedback
  const [toastVisible, setToastVisible] = useState(false);
  const [toastTitle, setToastTitle] = useState('Starting installation...');
  const [toastSubtitle, setToastSubtitle] = useState('Configuring Flatpak sandbox cleanly in background.');
  const [installingId, setInstallingId] = useState<string | null>(null);

  // Installed tracking
  const [installedMap, setInstalledMap] = useState<Record<string, boolean>>({
    'steam': true,
  });

  // Welcome banner dismiss state
  const [welcomeDismissed, setWelcomeDismissed] = useState(false);

  // Active Hero Slide Index
  const [activeHeroIdx, setActiveHeroIdx] = useState(0);

  // Trigger app install with toast and Go backend integration
  const handleInstall = async (appId: string, appName: string, provider: string = 'flatpak') => {
    setInstallingId(appId);
    setToastTitle(`Downloading ${appName}...`);
    setToastSubtitle('Configuring Flatpak sandbox cleanly in background.');
    setToastVisible(true);

    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.InstallPackage) {
      try {
        await goApp.InstallPackage(appId, provider);
        setInstalledMap((prev) => ({ ...prev, [appId]: true }));
        setInstallingId(null);
        setToastTitle('Installed and ready to launch!');
        setToastSubtitle(`${appName} desktop entry created.`);
        setTimeout(() => setToastVisible(false), 2400);
      } catch (err: any) {
        setInstallingId(null);
        setToastTitle(`Installation failed: ${appName}`);
        setToastSubtitle(err?.message || 'Could not fetch package.');
        setTimeout(() => setToastVisible(false), 3200);
      }
    } else {
      setTimeout(() => {
        setInstalledMap((prev) => ({ ...prev, [appId]: true }));
        setInstallingId(null);
        setToastTitle('Installed and ready to launch!');
        setToastSubtitle(`${appName} desktop entry created.`);
        setTimeout(() => setToastVisible(false), 2200);
      }, 1500);
    }
  };

  const topFreeApps: AppItem[] = [
    {
      id: 'brave-bin',
      name: 'Brave Browser',
      category: 'Web',
      desc: 'Fast, private web browsing with built-in ad and tracker blocking.',
      rating: 4.8,
      size: '98 MB',
      provider: 'aur',
      iconBg: 'from-[#fb542b] to-[#ff9e1b]',
      iconName: 'shield',
    },
    {
      id: 'spotify',
      name: 'Spotify',
      category: 'Music',
      desc: 'Stream your favorite podcasts, music tracks, and curated audio.',
      rating: 4.7,
      size: '115 MB',
      provider: 'aur',
      iconBg: 'bg-[#1ed760]',
      iconName: 'headphones',
    },
    {
      id: 'discord',
      name: 'Discord',
      category: 'Social',
      desc: 'Voice, video, and group text chat to hang out with friends & teams.',
      rating: 4.6,
      size: '92 MB',
      provider: 'flatpak',
      iconBg: 'bg-[#5865F2]',
      iconName: 'chat',
    },
    {
      id: 'steam',
      name: 'Steam',
      category: 'Gaming',
      desc: 'Play thousands of Windows & Linux games effortlessly via Proton.',
      rating: 4.9,
      size: '54 MB',
      provider: 'pacman',
      iconBg: 'from-[#171a21] to-[#2a475e]',
      iconName: 'sports_esports',
    },
    {
      id: 'onlyoffice-bin',
      name: 'OnlyOffice Suite',
      category: 'Office',
      desc: 'Full Microsoft Office compatibility for docs, sheets, and slides.',
      rating: 4.7,
      size: '285 MB',
      provider: 'aur',
      iconBg: 'from-[#ff6f3d] to-[#e74c3c]',
      iconName: 'description',
    },
  ];

  const categories = [
    { name: 'Creativity', count: '120+ apps', icon: 'brush', iconBg: 'bg-fluent-blue/15 text-fluent-blue', route: '/creativity' },
    { name: 'Productivity', count: '180+ tools', icon: 'work', iconBg: 'bg-indigo-500/15 text-indigo-400', route: '/apps' },
    { name: 'Gaming', count: '450+ titles', icon: 'sports_esports', iconBg: 'bg-emerald-500/15 text-emerald-400', route: '/gaming' },
    { name: 'Media & Audio', count: '95+ players', icon: 'music_note', iconBg: 'bg-amber-500/15 text-amber-400', route: '/entertainment' },
    { name: 'Developer Tools', count: '230+ utilities', icon: 'terminal', iconBg: 'bg-sky-500/15 text-sky-400', route: '/apps' },
  ];

  const topCharts = [
    {
      rank: 1,
      id: 'vlc',
      name: 'VLC Media Player',
      desc: 'Plays all video formats out of the box',
      rating: 4.9,
      size: '38 MB',
      provider: 'pacman',
      iconBg: 'from-[#ea580c] to-[#f97316]',
      iconName: 'smart_display',
    },
    {
      rank: 2,
      id: 'obs-studio',
      name: 'OBS Studio',
      desc: 'Live streaming & screen recording',
      rating: 4.8,
      size: '142 MB',
      provider: 'pacman',
      iconBg: 'bg-slate-800 border border-white/10',
      iconName: 'videocam',
    },
    {
      rank: 3,
      id: 'visual-studio-code-bin',
      name: 'Visual Studio Code',
      desc: 'Streamlined code editing & extensions',
      rating: 4.9,
      size: '95 MB',
      provider: 'aur',
      iconBg: 'bg-[#007acc]',
      iconName: 'code',
    },
    {
      rank: 4,
      id: 'telegram-desktop',
      name: 'Telegram Desktop',
      desc: 'Fast, cloud-based private messenger',
      rating: 4.8,
      size: '65 MB',
      provider: 'pacman',
      iconBg: 'bg-[#229ed9]',
      iconName: 'send',
    },
    {
      rank: 5,
      id: 'gimp',
      name: 'GIMP 2.10',
      desc: 'Professional photo editing & graphics',
      rating: 4.6,
      size: '124 MB',
      provider: 'pacman',
      iconBg: 'bg-[#5c5543]',
      iconName: 'draw',
    },
    {
      rank: 6,
      id: 'bitwarden',
      name: 'Bitwarden',
      desc: 'Zero-knowledge password manager',
      rating: 4.9,
      size: '72 MB',
      provider: 'flatpak',
      iconBg: 'bg-[#175ddc]',
      iconName: 'key',
    },
  ];

  return (
    <div className="p-6 md:p-8 max-w-[1360px] mx-auto space-y-8 select-none">
      
      {/* WELCOMING BANNER FOR LINUX FIRST-TIMERS */}
      {!welcomeDismissed && (
        <div className="relative overflow-hidden rounded-xl border border-white/[0.08] bg-gradient-to-r from-mica-elevated/70 via-mica-surface/50 to-transparent p-4 sm:p-5 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 transition-all">
          <div className="flex items-center gap-3.5">
            <div className="w-10 h-10 rounded-lg bg-fluent-blue/20 border border-fluent-blue/30 text-fluent-blue flex items-center justify-center shrink-0">
              <span className="material-symbols-outlined text-[22px]">waving_hand</span>
            </div>
            <div>
              <h2 className="text-[14px] font-semibold text-white flex items-center gap-2">
                New to Linux? No terminal required.
                <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-[#107c41]/20 text-[#34d399] border border-[#107c41]/30">
                  Verified &amp; Safe
                </span>
              </h2>
              <p className="text-[12px] text-fluent-muted mt-0.5">
                Every application here installs with 1-click sandboxed isolation. Your system stays protected and updates happen automatically.
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2.5 shrink-0 self-end sm:self-center">
            <button
              type="button"
              onClick={() => navigate('/settings')}
              className="px-3 py-1.5 rounded-md text-[12px] font-medium text-fluent-muted hover:text-white bg-white/[0.05] hover:bg-white/[0.08] border border-white/[0.08] transition"
            >
              Learn basics
            </button>
            <button
              type="button"
              onClick={() => setWelcomeDismissed(true)}
              className="px-3.5 py-1.5 rounded-md text-[12px] font-medium bg-fluent-blue hover:bg-fluent-blueHover text-white shadow-fluent-btn transition active:scale-[0.98]"
            >
              Got it
            </button>
          </div>
        </div>
      )}

      {/* 1. HERO SHOWCASE SPOTLIGHT: WINDOWS 11 STORE STYLE CAROUSEL */}
      {(() => {
        const currentHero = HERO_SLIDES[activeHeroIdx] || HERO_SLIDES[0];
        const isHeroInstalled = !!installedMap[currentHero.id];
        const isHeroBusy = installingId === currentHero.id;

        return (
          <section className="relative rounded-2xl overflow-hidden border border-white/[0.09] shadow-fluent-elevation bg-mica-base">
            <div className="relative min-h-[380px] p-6 sm:p-8 lg:p-10 flex flex-col lg:flex-row items-center justify-between gap-8 bg-gradient-to-r from-[#141822] via-[#121620] to-[#181d28]">
              
              {/* Left: App Details Card */}
              <div className="max-w-xl space-y-4 z-10">
                <div className="flex items-center gap-2">
                  <span className="px-2.5 py-0.5 rounded text-[11px] font-medium bg-fluent-blue/20 text-[#60a5fa] border border-fluent-blue/30">
                    {currentHero.badge || "Editor's Choice Spotlight"}
                  </span>
                  <span className="text-[12px] text-fluent-dim">•</span>
                  <span className="text-[12px] text-fluent-muted">{currentHero.category}</span>
                </div>

                <div className="flex items-start gap-4">
                  <div className="w-16 h-16 rounded-xl bg-transparent p-1 flex items-center justify-center shrink-0 overflow-hidden">
                    <AppIcon
                      appId={currentHero.id}
                      pkgName={currentHero.id}
                      provider={currentHero.provider}
                      iconUrl={currentHero.iconUrl}
                      fallbackIcon={currentHero.iconName}
                      fallbackBg="bg-transparent"
                      size={128}
                      className="w-14 h-14"
                      imgClassName="w-14 h-14 object-contain drop-shadow-md"
                      alt={currentHero.name}
                    />
                  </div>
                  <div>
                    <h1 
                      onClick={() => navigate(`/package/${currentHero.id}`)}
                      className="text-2xl sm:text-3xl font-bold text-white tracking-tight cursor-pointer hover:text-fluent-blue transition"
                    >
                      {currentHero.name}
                    </h1>
                    <p className="text-fluent-muted text-[13px] mt-0.5">
                      {currentHero.publisher} • Official {currentHero.provider.toUpperCase()}
                    </p>
                    <div className="flex items-center gap-2 mt-2 text-[12px]">
                      <div className="flex text-[#f59e0b]">
                        <span className="material-symbols-outlined text-[15px] fill-current">star</span>
                        <span className="material-symbols-outlined text-[15px] fill-current">star</span>
                        <span className="material-symbols-outlined text-[15px] fill-current">star</span>
                        <span className="material-symbols-outlined text-[15px] fill-current">star</span>
                        <span className="material-symbols-outlined text-[15px] fill-current">star</span>
                      </div>
                      <span className="font-semibold text-white">{currentHero.rating}</span>
                      <span className="text-fluent-dim">({currentHero.reviews} reviews)</span>
                      <span className="text-fluent-dim">•</span>
                      <span className="text-[#34d399] font-medium">Free &amp; Open Source</span>
                    </div>
                  </div>
                </div>

                <p className="text-fluent-muted text-[13px] leading-relaxed max-w-lg">
                  {currentHero.description}
                </p>

                <div className="pt-2 flex items-center gap-3 flex-wrap">
                  {isHeroInstalled ? (
                    <button
                      type="button"
                      className="px-6 py-2.5 bg-white/[0.05] text-[#34d399] border border-white/[0.09] font-medium text-[13px] rounded-md flex items-center gap-2 cursor-default"
                    >
                      <span className="material-symbols-outlined text-[18px]">check</span>
                      <span>Installed</span>
                    </button>
                  ) : (
                    <button
                      type="button"
                      disabled={isHeroBusy}
                      onClick={() => handleInstall(currentHero.id, currentHero.name, currentHero.provider)}
                      className="app-install-btn px-6 py-2.5 bg-fluent-blue hover:bg-fluent-blueHover text-white font-medium text-[13px] rounded-md flex items-center gap-2 shadow-fluent-btn transition active:scale-[0.98] disabled:opacity-75"
                    >
                      {isHeroBusy ? (
                        <>
                          <span className="material-symbols-outlined text-[18px] animate-spin">refresh</span>
                          <span>Installing...</span>
                        </>
                      ) : (
                        <>
                          <span className="material-symbols-outlined text-[18px]">download</span>
                          <span>Install Free</span>
                        </>
                      )}
                    </button>
                  )}

                  <button
                    type="button"
                    onClick={() => navigate(`/package/${currentHero.id}`)}
                    className="px-4 py-2.5 bg-white/[0.06] hover:bg-white/[0.1] text-white border border-white/[0.09] font-medium text-[13px] rounded-md transition flex items-center gap-1.5 cursor-pointer"
                  >
                    <span className="material-symbols-outlined text-[18px]">info</span>
                    <span>App Details</span>
                  </button>

                  <div className="flex items-center gap-1.5 text-[11px] text-fluent-dim ml-2">
                    <span className="material-symbols-outlined text-[15px] text-[#34d399]">verified_user</span>
                    <span>Sandboxed {currentHero.provider.toUpperCase()} Package</span>
                  </div>
                </div>
              </div>

              {/* Right: Screen / Cinematic Preview Strip */}
              <div className="w-full lg:w-[460px] bg-black/40 border border-white/[0.08] rounded-xl p-3 shadow-xl backdrop-blur-md space-y-3 z-10">
                <div className="flex items-center justify-between text-[11.5px] px-1 text-fluent-muted">
                  <span className="flex items-center gap-1 text-white font-medium">
                    <span className="material-symbols-outlined text-[16px] text-fluent-blue">smart_display</span>
                    <span>Feature Demonstration</span>
                  </span>
                  <span className="text-fluent-dim">{currentHero.version} • Latest Build</span>
                </div>

                {/* Video / Screenshot Mockup */}
                <div 
                  onClick={() => navigate(`/package/${currentHero.id}`)}
                  style={currentHero.screenshots?.[0]?.url ? {
                    backgroundImage: `linear-gradient(to bottom, rgba(15, 23, 42, 0.65), rgba(15, 23, 42, 0.85)), url('${currentHero.screenshots[0].url}')`,
                    backgroundSize: 'cover',
                    backgroundPosition: 'center'
                  } : undefined}
                  className="h-48 rounded-lg bg-gradient-to-tr from-slate-900 via-slate-800 to-indigo-950 border border-white/[0.06] relative overflow-hidden flex flex-col justify-between p-3.5 group cursor-pointer"
                >
                  <div className="flex items-center justify-between">
                    <span className="px-2 py-0.5 rounded text-[10px] bg-black/60 text-white border border-white/10 backdrop-blur">
                      {currentHero.heroPreview?.tagline || currentHero.headline}
                    </span>
                    <span className="px-2 py-0.5 rounded text-[10px] bg-black/60 text-fluent-muted border border-white/10 backdrop-blur">
                      {currentHero.category}
                    </span>
                  </div>

                  <div className="flex items-center justify-center">
                    <div className="w-12 h-12 rounded-full bg-white/15 border border-white/30 backdrop-blur-md flex items-center justify-center text-white group-hover:scale-110 group-hover:bg-fluent-blue transition duration-200">
                      <span className="material-symbols-outlined text-[24px]">play_arrow</span>
                    </div>
                  </div>

                  <div className="flex items-center justify-between text-[11px] text-white/80 bg-black/50 backdrop-blur px-2.5 py-1 rounded">
                    <span>{currentHero.heroPreview?.subtext || 'Verified Linux Package'}</span>
                    <span className="text-fluent-blue font-semibold">{currentHero.heroPreview?.stats || '1-Click'}</span>
                  </div>
                </div>

                {/* Carousel Dots */}
                <div className="flex items-center justify-center gap-1.5 pt-1">
                  {HERO_SLIDES.map((slide, idx) => (
                    <button 
                      key={slide.id}
                      type="button"
                      onClick={() => setActiveHeroIdx(idx)}
                      title={slide.name}
                      className={`transition ${activeHeroIdx === idx ? 'w-6 h-1 rounded-full bg-fluent-blue' : 'w-1.5 h-1 rounded-full bg-white/20 hover:bg-white/40'}`} 
                    />
                  ))}
                </div>
              </div>
            </div>
          </section>
        );
      })()}

      {/* 2. TOP FREE APPS FOR YOUR NEW PC (Fluent Curated Ribbon) */}
      <section className="space-y-3.5">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-[17px] font-semibold text-white">Top Free Apps for Your New PC</h2>
            <p className="text-[12px] text-fluent-muted">Everyday essentials to browse, listen, game, and communicate</p>
          </div>
          <button
            onClick={() => navigate('/search')}
            className="text-[12px] font-medium text-fluent-blue hover:text-fluent-blueHover flex items-center gap-1 transition"
          >
            <span>See all</span>
            <span className="material-symbols-outlined text-[15px]">arrow_forward</span>
          </button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
          {topFreeApps.map((app) => {
            const isInstalled = !!installedMap[app.id];
            const isBusy = installingId === app.id;

            return (
              <div key={app.id} className="fluent-card rounded-xl p-3.5 flex flex-col justify-between group">
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <div 
                      onClick={() => navigate(`/package/${app.id}`)}
                      className="w-12 h-12 rounded-xl bg-transparent flex items-center justify-center cursor-pointer hover:scale-110 transition overflow-hidden p-0.5 shrink-0"
                    >
                      <AppIcon
                        appId={app.id}
                        pkgName={app.name}
                        provider={app.provider}
                        iconUrl={APPS_DATA[app.id]?.iconUrl}
                        fallbackIcon={app.iconName}
                        fallbackBg="bg-transparent"
                        size={128}
                        className="w-11 h-11"
                        imgClassName="w-11 h-11 object-contain drop-shadow"
                        alt={app.name}
                      />
                    </div>
                    <span className="text-[10px] font-medium text-fluent-dim px-2 py-0.5 rounded bg-white/[0.04]">
                      {app.category}
                    </span>
                  </div>

                  <div>
                    <h4 
                      onClick={() => navigate(`/package/${app.id}`)}
                      className="font-medium text-[13.5px] text-white group-hover:text-fluent-blue transition cursor-pointer"
                    >
                      {app.name}
                    </h4>
                    <p className="text-[11.5px] text-fluent-muted mt-0.5 line-clamp-2">
                      {app.desc}
                    </p>
                  </div>

                  <div className="flex items-center gap-1 text-[11px] text-fluent-muted">
                    <span className="text-[#f59e0b]">★ {app.rating}</span>
                    <span className="text-fluent-dim">•</span>
                    <span className="text-fluent-dim">{app.size}</span>
                  </div>
                </div>

                <div className="pt-3 mt-3 border-t border-white/[0.06] flex items-center justify-between">
                  <span className={`text-[11px] font-medium ${isInstalled ? 'text-[#34d399] flex items-center gap-1' : 'text-fluent-muted'}`}>
                    {isInstalled ? (
                      <>
                        <span className="material-symbols-outlined text-[13px]">check_circle</span>
                        <span>Installed</span>
                      </>
                    ) : (
                      'Free'
                    )}
                  </span>

                  {isInstalled ? (
                    <button
                      type="button"
                      onClick={() => navigate(`/package/${app.id}`)}
                      className="px-3 py-1 bg-white/[0.05] hover:bg-white/[0.1] text-fluent-muted hover:text-white font-medium text-[11.5px] rounded-md transition"
                    >
                      Open
                    </button>
                  ) : (
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => handleInstall(app.id, app.name, app.provider)}
                      className="app-install-btn px-3 py-1 bg-white/[0.08] hover:bg-fluent-blue hover:text-white text-fluent-text font-medium text-[11.5px] rounded-md transition flex items-center gap-1 disabled:opacity-75"
                    >
                      {isBusy ? (
                        <>
                          <span className="material-symbols-outlined text-[13px] animate-spin">refresh</span>
                          <span>...</span>
                        </>
                      ) : (
                        <span>Get</span>
                      )}
                    </button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </section>

      {/* 3. FEATURED CATEGORIES (Fluent Tile Grid) */}
      <section className="space-y-3">
        <h2 className="text-[17px] font-semibold text-white">Browse Categories</h2>
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-3">
          {categories.map((cat) => (
            <div
              key={cat.name}
              onClick={() => navigate(cat.route)}
              className="fluent-card-subtle p-3 rounded-xl flex items-center gap-3 group transition cursor-pointer"
            >
              <div className={`w-9 h-9 rounded-lg ${cat.iconBg} flex items-center justify-center group-hover:bg-fluent-blue group-hover:text-white transition`}>
                <span className="material-symbols-outlined text-[20px]">{cat.icon}</span>
              </div>
              <div>
                <div className="text-[12.5px] font-medium text-white group-hover:text-fluent-blue transition">
                  {cat.name}
                </div>
                <div className="text-[11px] text-fluent-dim">{cat.count}</div>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* 4. TRENDING CHARTS & EDITOR'S CHOICE (Ranked Asymmetric List View) */}
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-[17px] font-semibold text-white">Top Charts &amp; Recommended</h2>
            <p className="text-[12px] text-fluent-muted">Most installed programs on Linux this month</p>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-[11px] text-fluent-dim">Sorted by:</span>
            <select className="bg-black/30 border border-white/[0.08] text-[11.5px] text-fluent-muted py-1 px-2.5 rounded-md focus:outline-none focus:border-fluent-blue cursor-pointer">
              <option>Most Popular</option>
              <option>Top Rated</option>
              <option>New Additions</option>
            </select>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3.5">
          {topCharts.map((pick) => {
            const isInstalled = !!installedMap[pick.id];
            const isBusy = installingId === pick.id;

            return (
              <div key={pick.id} className="fluent-card rounded-xl p-3 flex items-center justify-between gap-3 group">
                <div
                  onClick={() => navigate(`/package/${pick.id}`)}
                  className="flex items-center gap-3 min-w-0 cursor-pointer flex-1"
                >
                  <span className="font-semibold text-[14px] text-fluent-dim w-4 text-center shrink-0">
                    {pick.rank}
                  </span>
                  <div className="w-11 h-11 rounded-lg bg-transparent flex items-center justify-center shrink-0 overflow-hidden p-0.5">
                    <AppIcon
                      appId={pick.id}
                      pkgName={pick.name}
                      provider={pick.provider}
                      iconUrl={APPS_DATA[pick.id]?.iconUrl}
                      fallbackIcon={pick.iconName}
                      fallbackBg="bg-transparent"
                      size={128}
                      className="w-10 h-10"
                      imgClassName="w-10 h-10 object-contain drop-shadow"
                      alt={pick.name}
                    />
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                      <h4 className="font-medium text-[13px] text-white group-hover:text-fluent-blue transition truncate">
                        {pick.name}
                      </h4>
                    </div>
                    <p className="text-[11px] text-fluent-dim truncate">{pick.desc}</p>
                    <div className="flex items-center gap-1.5 text-[10.5px] text-fluent-muted mt-0.5">
                      <span className="text-[#f59e0b]">★ {pick.rating}</span>
                      <span>•</span>
                      <span>{pick.size}</span>
                    </div>
                  </div>
                </div>

                <div className="shrink-0">
                  {isInstalled ? (
                    <button
                      type="button"
                      onClick={() => navigate(`/package/${pick.id}`)}
                      className="px-3 py-1 bg-white/[0.05] hover:bg-white/[0.1] text-fluent-muted hover:text-white font-medium text-[11.5px] rounded-md transition"
                    >
                      Open
                    </button>
                  ) : (
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => handleInstall(pick.id, pick.name, pick.provider)}
                      className="app-install-btn px-3 py-1 bg-white/[0.08] hover:bg-fluent-blue hover:text-white text-fluent-text font-medium text-[11.5px] rounded-md transition flex items-center gap-1 disabled:opacity-75"
                    >
                      {isBusy ? (
                        <>
                          <span className="material-symbols-outlined text-[13px] animate-spin">refresh</span>
                          <span>...</span>
                        </>
                      ) : (
                        <span>Get</span>
                      )}
                    </button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </section>

      {/* 5. FOOTER WITH SAFE SYSTEM INFORMATION */}
      <footer className="pt-6 pb-4 border-t border-white/[0.06] flex flex-col sm:flex-row items-center justify-between gap-4 text-[11.5px] text-fluent-dim">
        <div className="flex items-center gap-4">
          <span>Arjax Software Store v3.4.1</span>
          <span>•</span>
          <a className="hover:text-white transition" href="#/settings">Security Policy</a>
          <span>•</span>
          <a className="hover:text-white transition" href="#/settings">Flathub Verified</a>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-[#107c41]"></span>
          <span>Sandboxed App Environment Active</span>
        </div>
      </footer>

      {/* FLUENT NOTIFICATION / INSTALL FLYOUT */}
      <div 
        id="fluentToast"
        className={`fixed bottom-5 right-5 bg-mica-elevated border border-white/[0.12] text-fluent-text px-4 py-3 rounded-lg shadow-fluent-flyout z-50 flex items-center gap-3 transition-all duration-300 ${
          toastVisible ? 'translate-y-0 opacity-100' : 'translate-y-24 opacity-0 pointer-events-none'
        }`}
      >
        <div className="w-7 h-7 rounded-md bg-fluent-blue/20 text-fluent-blue flex items-center justify-center">
          <span className="material-symbols-outlined text-[18px]">downloading</span>
        </div>
        <div>
          <div className="text-[12.5px] font-semibold text-white">{toastTitle}</div>
          <div className="text-[11px] text-fluent-muted">{toastSubtitle}</div>
        </div>
      </div>

    </div>
  );
};
