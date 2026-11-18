import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ALL_INDEXED_APPS } from '../data/appsData';
import { AppIcon } from '../components/AppIcon';

interface AppCard {
  id: string;
  name: string;
  category: string;
  version: string;
  rating: string;
  reviews: string;
  description: string;
  icon: string;
  iconUrl?: string;
  iconColor: string;
  badge?: string;
  provider: string;
  size: string;
}

interface CategoryConfig {
  title: string;
  subtitle: string;
  badge: string;
  hero: {
    title: string;
    tagline: string;
    description: string;
    badge: string;
    version: string;
    category: string;
    icon: string;
    iconColor: string;
    ctaPackage: string;
    features: string[];
  };
  apps: AppCard[];
}

const CATEGORY_DATA: Record<string, CategoryConfig> = {
  apps: {
    title: 'Featured Apps & Tools',
    subtitle: 'Essential native and containerized applications for everyday productivity, development, and system utilities.',
    badge: 'Productivity & Tools',
    hero: {
      title: 'Visual Studio Code',
      tagline: 'Code editing. Redefined.',
      description: 'Free, built on open source, runs everywhere. Complete code editor with integrated terminal, debugging, and thousands of extensions.',
      badge: 'Editor of the Year',
      version: 'v1.89.1',
      category: 'Developer Tools',
      icon: 'code',
      iconColor: 'text-primary',
      ctaPackage: 'visual-studio-code-bin',
      features: ['Integrated Terminal', 'Git Lens Built-in', 'Wayland Accelerated', 'Sandboxed Flatpak'],
    },
    apps: [
      { id: 'brave', name: 'Brave Browser', category: 'Web Browser', version: 'v1.65.130', rating: '4.8', reviews: '18,400', description: 'Privacy-focused Chromium browser with native ad blocking.', icon: 'travel_explore', iconColor: 'text-secondary', badge: 'Popular', provider: 'flatpak', size: '185 MB' },
      { id: 'alacritty', name: 'Alacritty', category: 'Terminal', version: 'v0.13.2', rating: '4.9', reviews: '6,200', description: 'GPU-accelerated terminal emulator written in Rust.', icon: 'terminal', iconColor: 'text-tertiary', provider: 'pacman', size: '14 MB' },
      { id: 'btop', name: 'btop', category: 'System Monitor', version: 'v1.3.2', rating: '4.9', reviews: '9,800', description: 'Modern, beautiful terminal resource monitor showing CPU and RAM.', icon: 'query_stats', iconColor: 'text-primary', badge: 'Essential', provider: 'pacman', size: '4.2 MB' },
      { id: 'onlyoffice', name: 'OnlyOffice Desktop', category: 'Office Suite', version: 'v8.0.1', rating: '4.7', reviews: '5,100', description: 'Full office suite highly compatible with Microsoft Office formats.', icon: 'description', iconColor: 'text-secondary', provider: 'flatpak', size: '340 MB' },
      { id: 'obsidian', name: 'Obsidian', category: 'Knowledge Base', version: 'v1.5.12', rating: '4.9', reviews: '14,300', description: 'Markdown-based connected note-taking and knowledge graph.', icon: 'account_tree', iconColor: 'text-tertiary', badge: 'Top Pick', provider: 'flatpak', size: '92 MB' },
      { id: 'thunderbird', name: 'Mozilla Thunderbird', category: 'Email Client', version: 'v115.10', rating: '4.6', reviews: '8,200', description: 'Open-source email, calendar, and address book client.', icon: 'mail', iconColor: 'text-primary', provider: 'pacman', size: '78 MB' },
    ],
  },
  gaming: {
    title: 'Gaming on Linux & Proton',
    subtitle: 'Discover games, compatibility layers, and launchers. Run thousands of Windows games seamlessly with Proton and Vulkan.',
    badge: 'Proton Verified & Native',
    hero: {
      title: 'Steam',
      tagline: 'The Ultimate Entertainment Platform',
      description: 'Play thousands of native and Windows titles seamlessly on Linux through Valve Proton with hardware shader pre-caching.',
      badge: 'Proton 9.0 Enabled',
      version: 'v1.0.0.79',
      category: 'Gaming Platform',
      icon: 'sports_esports',
      iconColor: 'text-tertiary',
      ctaPackage: 'steam',
      features: ['Proton GE Support', 'Vulkan Shader Caching', 'Steam Deck Verified', 'Direct NVMe I/O'],
    },
    apps: [
      { id: 'heroic', name: 'Heroic Games Launcher', category: 'Epic & GOG Client', version: 'v2.14.1', rating: '4.8', reviews: '7,300', description: 'Native GUI launcher for Epic Games, GOG, and Amazon Prime gaming.', icon: 'rocket_launch', iconColor: 'text-primary', badge: 'Top Rated', provider: 'flatpak', size: '120 MB' },
      { id: 'lutris', name: 'Lutris', category: 'Game Manager', version: 'v0.5.17', rating: '4.7', reviews: '11,200', description: 'Open gaming platform unifying Steam, GOG, Battle.net, and emulators.', icon: 'videogame_asset', iconColor: 'text-secondary', provider: 'pacman', size: '48 MB' },
      { id: 'retroarch', name: 'RetroArch', category: 'Emulation Hub', version: 'v1.18.0', rating: '4.9', reviews: '15,600', description: 'Universal multi-system frontend for classic game consoles and arcade cores.', icon: 'sports_esports', iconColor: 'text-tertiary', badge: 'Classic', provider: 'flatpak', size: '210 MB' },
      { id: 'protonup-qt', name: 'ProtonUp-Qt', category: 'Compatibility Tool', version: 'v2.9.2', rating: '4.9', reviews: '8,900', description: 'Install and manage Proton-GE, Luxtorpeda, and Wine-GE with one click.', icon: 'upgrade', iconColor: 'text-primary', badge: 'Must-Have', provider: 'flatpak', size: '34 MB' },
      { id: 'mangohud', name: 'MangoHud', category: 'Performance Overlay', version: 'v0.7.1', rating: '4.8', reviews: '6,400', description: 'Vulkan and OpenGL overlay for monitoring FPS, temperatures, and CPU/GPU load.', icon: 'speed', iconColor: 'text-secondary', provider: 'pacman', size: '8 MB' },
      { id: 'prism-launcher', name: 'Prism Launcher', category: 'Minecraft Client', version: 'v8.3', rating: '4.9', reviews: '12,100', description: 'High performance custom Minecraft launcher with modpack automation.', icon: 'deployed_code', iconColor: 'text-tertiary', provider: 'flatpak', size: '52 MB' },
    ],
  },
  creativity: {
    title: 'Creative & 3D Production',
    subtitle: 'Professional 3D animation, digital painting, vector illustration, and cinema audio/video suites.',
    badge: 'Pro Creator Suite',
    hero: {
      title: 'Blender 4.2 LTS',
      tagline: 'Open Source 3D Creation Pipeline',
      description: 'The world premier free 3D creation suite. Modeling, rigging, sculpting, EEVEE Next real-time rendering, and Cycles GPU raytracing.',
      badge: "Editors' Choice",
      version: 'v4.2.2 LTS',
      category: '3D & VFX',
      icon: 'view_in_ar',
      iconColor: 'text-secondary',
      ctaPackage: 'blender',
      features: ['Cycles GPU Raytracing', 'EEVEE Next Engine', 'Geometry Nodes', '100% Free Forever'],
    },
    apps: [
      { id: 'krita', name: 'Krita', category: 'Digital Painting', version: 'v5.2.2', rating: '4.9', reviews: '16,200', description: 'Professional concept art, digital painting, and comic book illustration software.', icon: 'brush', iconColor: 'text-secondary', badge: 'Best for Art', provider: 'flatpak', size: '280 MB' },
      { id: 'inkscape', name: 'Inkscape', category: 'Vector Illustration', version: 'v1.3.2', rating: '4.8', reviews: '14,100', description: 'Professional vector graphics editor for SVG illustration and logo design.', icon: 'draw', iconColor: 'text-tertiary', provider: 'flatpak', size: '140 MB' },
      { id: 'gimp', name: 'GIMP', category: 'Photo Editing', version: 'v2.10.38', rating: '4.7', reviews: '22,400', description: 'Extensible image retouching and composition program with plugin ecosystem.', icon: 'photo_library', iconColor: 'text-primary', badge: 'Standard', provider: 'pacman', size: '115 MB' },
      { id: 'kdenlive', name: 'Kdenlive', category: 'Video Editor', version: 'v24.02.2', rating: '4.7', reviews: '9,300', description: 'Non-linear video editor with multi-track timeline, color grading, and effects.', icon: 'movie', iconColor: 'text-secondary', provider: 'flatpak', size: '230 MB' },
      { id: 'audacity', name: 'Audacity', category: 'Audio Editor', version: 'v3.5.1', rating: '4.8', reviews: '19,500', description: 'Multi-track audio editor and recorder with audio normalization and noise reduction.', icon: 'graphic_eq', iconColor: 'text-tertiary', provider: 'pacman', size: '42 MB' },
      { id: 'darktable', name: 'Darktable', category: 'RAW Photography', version: 'v4.6.1', rating: '4.8', reviews: '6,700', description: 'Virtual lighttable and darkroom for RAW image processing and camera tethering.', icon: 'camera_alt', iconColor: 'text-primary', provider: 'pacman', size: '68 MB' },
    ],
  },
  entertainment: {
    title: 'Entertainment & Streaming',
    subtitle: 'Music players, home theater hubs, high-fidelity audio streams, and communication networks.',
    badge: 'Media & Streaming',
    hero: {
      title: 'VLC Media Player',
      tagline: 'Plays everything. Anywhere.',
      description: 'Free, open source cross-platform multimedia player and framework that plays most multimedia files, DVDs, Audio CDs, VCDs, and various streaming protocols.',
      badge: 'Universal Decoder',
      version: 'v3.0.21',
      category: 'Media Player',
      icon: 'smart_display',
      iconColor: 'text-secondary',
      ctaPackage: 'vlc',
      features: ['Hardware Acceleration', 'Plays All Codecs', 'No Spyware/Ads', 'Audio Normalization'],
    },
    apps: [
      { id: 'spotify', name: 'Spotify', category: 'Music Streaming', version: 'v1.2.37', rating: '4.8', reviews: '26,500', description: 'Stream millions of songs, podcasts, and curated playlists in high fidelity.', icon: 'headphones', iconColor: 'text-tertiary', badge: 'Essential', provider: 'flatpak', size: '160 MB' },
      { id: 'discord', name: 'Discord', category: 'Voice & Community', version: 'v0.0.48', rating: '4.7', reviews: '34,200', description: 'Voice, video, and text communication platform for gamers and communities.', icon: 'forum', iconColor: 'text-primary', badge: 'Top Chat', provider: 'flatpak', size: '85 MB' },
      { id: 'kodi', name: 'Kodi', category: 'Home Theater', version: 'v21.0', rating: '4.8', reviews: '11,400', description: 'Award-winning media center software for TVs, remote controls, and movies.', icon: 'tv', iconColor: 'text-secondary', provider: 'flatpak', size: '210 MB' },
      { id: 'stremio', name: 'Stremio', category: 'Video Aggregator', version: 'v4.4.168', rating: '4.9', reviews: '14,800', description: 'Modern video aggregator allowing discovery, organization, and streaming.', icon: 'theaters', iconColor: 'text-tertiary', badge: 'Popular', provider: 'flatpak', size: '95 MB' },
      { id: 'telegram', name: 'Telegram Desktop', category: 'Fast Messaging', version: 'v5.0.1', rating: '4.9', reviews: '21,100', description: 'Cloud-based mobile and desktop messaging app with encryption and large channels.', icon: 'send', iconColor: 'text-primary', provider: 'pacman', size: '54 MB' },
      { id: 'mpv', name: 'mpv', category: 'Minimalist Player', version: 'v0.38.0', rating: '4.9', reviews: '12,900', description: 'Ultra-fast, minimalist command-line and GUI media player with GPU shaders.', icon: 'play_arrow', iconColor: 'text-secondary', provider: 'pacman', size: '18 MB' },
    ],
  },
};

export const CategoryPage: React.FC<{ type: 'apps' | 'gaming' | 'creativity' | 'entertainment' }> = ({ type }) => {
  const navigate = useNavigate();
  const config = CATEGORY_DATA[type] || CATEGORY_DATA.apps;

  const [installedMap, setInstalledMap] = useState<Record<string, boolean>>({});
  const [toastVisible, setToastVisible] = useState(false);
  const [toastMessage, setToastMessage] = useState({ title: '', desc: '' });

  // Merge static hero apps with full indexed top apps catalog
  const catalogApps = ALL_INDEXED_APPS.filter((a) => a.categorySlug === type);
  const displayedApps: AppCard[] = [...config.apps];
  const seenIds = new Set(config.apps.map((a) => a.id.toLowerCase()));

  catalogApps.forEach((catApp) => {
    if (!seenIds.has(catApp.id.toLowerCase())) {
      seenIds.add(catApp.id.toLowerCase());
      displayedApps.push({
        id: catApp.id,
        name: catApp.name,
        category: catApp.category,
        version: catApp.version,
        rating: String(catApp.rating),
        reviews: catApp.reviews,
        description: catApp.headline,
        icon: catApp.iconName,
        iconUrl: catApp.iconUrl,
        iconColor: type === 'gaming' ? 'text-secondary' : type === 'creativity' ? 'text-tertiary' : 'text-fluent-blue',
        badge: catApp.badge,
        provider: catApp.provider,
        size: catApp.size,
      });
    }
  });

  const triggerToast = (title: string, desc: string, duration = 2400) => {
    setToastMessage({ title, desc });
    setToastVisible(true);
    setTimeout(() => setToastVisible(false), duration);
  };

  const handleInstall = async (pkgName: string, provider: string) => {
    setInstalledMap((prev) => ({ ...prev, [pkgName]: true }));
    triggerToast(`Installing ${pkgName}...`, `Downloading verified container via ${provider}.`);

    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.InstallPackage) {
      try {
        await goApp.InstallPackage(pkgName, provider);
      } catch (err) {
        console.warn(err);
      }
    }
  };

  return (
    <div className="flex flex-col w-full font-sans">
      <div className="w-full max-w-[1440px] mx-auto px-4 sm:px-6 lg:px-10 py-6 space-y-6">

        {/* 1. Category Header Banner */}
        <div className="space-y-1 pb-2 border-b border-white/[0.06]">
          <div className="flex items-center gap-2 text-fluent-blue font-label-sm text-label-sm uppercase tracking-wider">
            <span className="material-symbols-outlined text-[16px]">category</span>
            <span>{config.badge}</span>
          </div>
          <h1 className="text-2xl font-bold text-fluent-text tracking-tight">{config.title}</h1>
          <p className="text-xs text-fluent-muted max-w-2xl">{config.subtitle}</p>
        </div>

        {/* 2. Master Category Hero Spotlight */}
        <div className="fluent-card fluent-acrylic p-6 sm:p-8 rounded-2xl relative overflow-hidden shadow-fluent-elevation">
          <div className="absolute -right-20 -top-20 w-80 h-80 rounded-full bg-fluent-blue/15 blur-3xl pointer-events-none"></div>

          <div className="relative z-10 flex flex-col lg:flex-row items-start lg:items-center justify-between gap-6">
            <div className="flex items-start gap-5 min-w-0">
              <div className="w-20 h-20 rounded-2xl bg-white/[0.06] border border-white/[0.08] flex items-center justify-center shrink-0 shadow-lg">
                <span className={`material-symbols-outlined text-4xl ${config.hero.iconColor}`}>{config.hero.icon}</span>
              </div>
              <div className="space-y-2">
                <div className="flex items-center gap-2.5 flex-wrap">
                  <h2 className="text-2xl font-bold text-white tracking-tight">{config.hero.title}</h2>
                  <span className="px-2 py-0.5 rounded-full bg-white/[0.06] text-fluent-muted font-mono text-[11px] border border-white/[0.08]">
                    {config.hero.version}
                  </span>
                  <span className="px-2.5 py-0.5 rounded-full bg-fluent-blue/20 text-[#60a5fa] text-[11px] font-medium">
                    {config.hero.badge}
                  </span>
                </div>
                <p className="text-xs text-fluent-muted max-w-2xl leading-relaxed">{config.hero.description}</p>
                <div className="flex items-center flex-wrap gap-2 pt-1">
                  {config.hero.features.map((feat) => (
                    <span key={feat} className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-md bg-white/[0.04] border border-white/[0.06] text-[11px] text-fluent-text">
                      <span className="material-symbols-outlined text-[13px] text-emerald-400">check</span>
                      {feat}
                    </span>
                  ))}
                </div>
              </div>
            </div>

            <div className="flex flex-col sm:flex-row lg:flex-col gap-2 shrink-0 self-stretch sm:self-auto justify-center">
              <button
                onClick={() => handleInstall(config.hero.ctaPackage, 'flatpak')}
                disabled={installedMap[config.hero.ctaPackage]}
                className={`px-6 py-2.5 rounded-xl text-xs font-semibold flex items-center justify-center gap-2 transition-all shadow-fluent-btn active:scale-95 cursor-pointer ${
                  installedMap[config.hero.ctaPackage]
                    ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
                    : 'bg-fluent-blue hover:bg-fluent-blueHover text-white'
                }`}
                type="button"
              >
                <span className="material-symbols-outlined text-[16px]">
                  {installedMap[config.hero.ctaPackage] ? 'check' : 'download'}
                </span>
                <span>{installedMap[config.hero.ctaPackage] ? 'Installed' : 'Install Free'}</span>
              </button>
              <button
                onClick={() => navigate(`/package/${config.hero.ctaPackage}`)}
                className="px-4 py-2 rounded-xl bg-white/[0.04] hover:bg-white/[0.08] border border-white/[0.08] text-fluent-muted hover:text-white text-xs font-medium text-center transition cursor-pointer"
                type="button"
              >
                View Details
              </button>
            </div>
          </div>
        </div>

        {/* 3. Curated Apps Grid */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold uppercase tracking-wider text-fluent-dim">
              Curated Selection ({displayedApps.length})
            </h2>
            <span className="text-xs text-fluent-muted">Tested for Linux compatibility</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {displayedApps.map((app) => (
              <div
                key={app.id}
                className="fluent-card p-4 rounded-xl flex items-center justify-between gap-4 group transition-all"
              >
                <div
                  onClick={() => navigate(`/package/${app.id}`)}
                  className="flex items-center gap-3.5 min-w-0 cursor-pointer flex-1"
                >
                  <div className="w-12 h-12 rounded-xl bg-white/[0.05] border border-white/[0.08] flex items-center justify-center shrink-0 group-hover:scale-105 transition-transform overflow-hidden p-1.5">
                    <AppIcon
                      appId={app.id}
                      pkgName={app.name}
                      provider={app.provider}
                      iconUrl={app.iconUrl}
                      fallbackIcon={app.icon}
                      fallbackBg="bg-transparent"
                      size={128}
                      className="w-9 h-9"
                      imgClassName="w-9 h-9 object-contain"
                      alt={app.name}
                    />
                  </div>
                  <div className="min-w-0 space-y-0.5">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-fluent-text text-sm truncate group-hover:text-fluent-blue transition-colors">
                        {app.name}
                      </span>
                      {app.badge && (
                        <span className="px-1.5 py-0.2 rounded text-[10px] font-medium bg-fluent-blue/20 text-[#60a5fa]">
                          {app.badge}
                        </span>
                      )}
                    </div>
                    <p className="text-[11.5px] text-fluent-muted truncate">{app.description}</p>
                    <div className="flex items-center gap-2 text-[11px] text-fluent-dim pt-0.5">
                      <span className="flex items-center text-secondary">
                        ★ <span className="text-fluent-text ml-0.5 font-medium">{app.rating}</span>
                      </span>
                      <span>•</span>
                      <span className="font-mono">{app.size}</span>
                      <span>•</span>
                      <span className="capitalize">{app.provider}</span>
                    </div>
                  </div>
                </div>

                <div className="shrink-0">
                  <button
                    onClick={() => handleInstall(app.id, app.provider)}
                    disabled={installedMap[app.id]}
                    className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer ${
                      installedMap[app.id]
                        ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
                        : 'bg-white/[0.06] hover:bg-fluent-blue hover:text-white text-fluent-text border border-white/[0.08] shadow-sm active:scale-95'
                    }`}
                    type="button"
                  >
                    {installedMap[app.id] ? 'Installed' : 'Get'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>

      </div>

      {/* FLUENT NOTIFICATION / INSTALL FLYOUT */}
      <div
        className={`fixed bottom-5 right-5 bg-mica-elevated border border-white/[0.12] text-fluent-text px-4 py-3 rounded-lg shadow-fluent-flyout z-50 flex items-center gap-3 transition-all duration-300 ${
          toastVisible ? 'translate-y-0 opacity-100' : 'translate-y-24 opacity-0 pointer-events-none'
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

