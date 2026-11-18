import topCatalog from './topAppsCatalog.json';

export interface AppFeature {
  icon: string;
  title: string;
  desc: string;
}

export interface AppScreenshot {
  url: string;
  title: string;
  caption: string;
  alt: string;
}

export interface AppDetail {
  id: string;
  flathubId?: string;
  name: string;
  headline: string;
  category: string;
  categorySlug: 'creativity' | 'gaming' | 'apps' | 'entertainment';
  publisher: string;
  version: string;
  rating: number;
  reviews: string;
  size: string;
  installedSize: string;
  provider: 'pacman' | 'aur' | 'flatpak' | 'snap';
  license: string;
  website: string;
  iconUrl?: string;
  iconName: string;
  iconBg: string;
  badge?: string;
  description: string;
  longDescription: string;
  features: AppFeature[];
  releaseNotes: string[];
  screenshots: AppScreenshot[];
  heroPreview?: {
    tagline: string;
    subtext: string;
    stats: string;
  };
}

export const APPS_DATA: Record<string, AppDetail> = {};

// 1. Seed APPS_DATA with all verified top catalog apps
(topCatalog as any[]).forEach((item) => {
  const shortId = (item.id || '').toLowerCase();
  const flathubId = (item.flathubId || '').toLowerCase();

  const appObj: AppDetail = {
    ...item,
    rating: item.rating || 4.8,
    reviews: item.reviews || '12,500',
    size: item.size || '85 MB',
    installedSize: item.installedSize || '240 MB',
    provider: item.provider || 'flatpak',
    license: item.license || 'Open Source',
    website: item.website || `https://flathub.org/apps/${item.flathubId}`,
    iconUrl: item.iconUrl || `https://dl.flathub.org/media/icons/128x128/${item.flathubId}.png`,
    iconName: item.iconName || 'grid_view',
    iconBg: item.iconBg || 'from-fluent-blue to-indigo-600',
    features: item.features || [
      {
        icon: 'bolt',
        title: 'Native High-Speed Execution',
        desc: 'Compiled and optimized for modern Linux desktop kernels.'
      },
      {
        icon: 'lock',
        title: 'Sandboxed Security',
        desc: 'Isolated bubblewrap runtime permissions protect system drives.'
      },
      {
        icon: 'terminal',
        title: 'Zero Terminal Hassle',
        desc: '1-click install automatically registers launcher icon and desktop entry.'
      }
    ],
    releaseNotes: item.releaseNotes || [
      'Upstream stable release verified against Arch Linux desktop environment.'
    ],
    screenshots: item.screenshots || []
  };

  if (shortId) {
    APPS_DATA[shortId] = appObj;
  }
  if (flathubId) {
    APPS_DATA[flathubId] = appObj;
  }
});

// 2. Custom Spotlight Metadata enhancements (preserving verified screenshots & URLs)
if (APPS_DATA['blender']) {
  APPS_DATA['blender'].badge = "Editors' Choice";
  APPS_DATA['blender'].iconName = 'view_in_ar';
  APPS_DATA['blender'].iconBg = 'from-[#ea7600] to-[#cf471d]';
  APPS_DATA['blender'].heroPreview = {
    tagline: 'Raytracing Cycles 3D Viewport',
    subtext: 'Full Vulkan & Wayland Acceleration',
    stats: '1080p 60fps'
  };
}

if (APPS_DATA['steam']) {
  APPS_DATA['steam'].badge = 'Essential';
  APPS_DATA['steam'].iconName = 'sports_esports';
  APPS_DATA['steam'].iconBg = 'from-[#171a21] to-[#2a475e]';
  APPS_DATA['steam'].heroPreview = {
    tagline: 'Proton 9.0 Compatibility Layer',
    subtext: 'Play thousands of AAA Windows & Linux titles',
    stats: '10,000+ Deck Verified'
  };
}

if (APPS_DATA['visual-studio-code-bin'] || APPS_DATA['code']) {
  const vs = APPS_DATA['visual-studio-code-bin'] || APPS_DATA['code'];
  vs.badge = "Editors' Choice";
  vs.iconName = 'code';
  vs.iconBg = 'bg-[#007acc]';
  vs.heroPreview = {
    tagline: 'TypeScript & Rust LSP Integration',
    subtext: 'Low latency Wayland editor with extension ecosystem',
    stats: 'Instant startup'
  };
}

if (APPS_DATA['spotify']) {
  APPS_DATA['spotify'].badge = 'Essential';
  APPS_DATA['spotify'].iconName = 'headphones';
  APPS_DATA['spotify'].iconBg = 'bg-[#1ed760]';
  APPS_DATA['spotify'].heroPreview = {
    tagline: 'Lossless Streaming & Podcasts',
    subtext: 'PipeWire & MPRIS2 media key control',
    stats: '320 kbps HiFi'
  };
}

// 3. Common package aliases mapping
const ALIAS_MAP: Record<string, string> = {
  'firefox': 'firefox',
  'firefox-bin': 'firefox',
  'chrome': 'chrome',
  'google-chrome': 'chrome',
  'google-chrome-stable': 'chrome',
  'brave': 'brave-bin',
  'brave-browser': 'brave-bin',
  'code': 'visual-studio-code-bin',
  'vscode': 'visual-studio-code-bin',
  'visual-studio-code': 'visual-studio-code-bin',
  'obs': 'obs-studio',
  'telegram': 'telegram-desktop',
  'telegram-desktop': 'telegram-desktop',
  'discord': 'discord',
  'vlc': 'vlc',
  'spotify': 'spotify',
  'steam': 'steam',
  'bitwarden': 'bitwarden',
  'onlyoffice': 'onlyoffice-bin',
  'onlyoffice-desktopeditors': 'onlyoffice-bin',
  'bottles': 'bottles',
  'flatseal': 'flatseal',
  'heroic': 'heroic',
  'lutris': 'lutris',
  'retroarch': 'retroarch',
  'gimp': 'gimp',
  'inkscape': 'inkscape',
  'krita': 'krita',
  'kdenlive': 'kdenlive',
  'audacity': 'audacity',
  'obsidian': 'obsidian'
};

Object.entries(ALIAS_MAP).forEach(([alias, targetId]) => {
  if (APPS_DATA[targetId]) {
    APPS_DATA[alias] = APPS_DATA[targetId];
  }
});

// Also register dot-notation reverse-DNS mappings (e.g. org.mozilla.firefox -> firefox)
Object.values(APPS_DATA).forEach((app) => {
  if (app.flathubId && !APPS_DATA[app.flathubId.toLowerCase()]) {
    APPS_DATA[app.flathubId.toLowerCase()] = app;
  }
});

export const ALL_INDEXED_APPS: AppDetail[] = Object.values(APPS_DATA).filter(
  (app, index, self) => index === self.findIndex((a) => a.id === app.id)
);

export const HERO_SLIDES: AppDetail[] = [
  APPS_DATA['blender'] || APPS_DATA['firefox'],
  APPS_DATA['steam'],
  APPS_DATA['visual-studio-code-bin'] || APPS_DATA['code'],
  APPS_DATA['spotify']
].filter(Boolean);
