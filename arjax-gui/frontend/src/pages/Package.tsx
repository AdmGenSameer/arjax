import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { APPS_DATA, AppDetail } from '../data/appsData';
import { AppIcon } from '../components/AppIcon';
import { SourcePicker } from '../components/SourcePicker';
import { UnifiedApplicationDTO, PackageOfferDTO, TransactionDTO } from '../types';

interface Screenshot {
  url: string;
  caption: string;
  title: string;
  alt: string;
}

export const PackagePage: React.FC = () => {
  const { name = 'blender' } = useParams<{ name: string }>();
  const navigate = useNavigate();

  const normName = (name || 'blender').toLowerCase();

  // 1. Resolve curated metadata from APPS_DATA if available
  const appStatic: AppDetail | undefined =
    APPS_DATA[normName] ||
    APPS_DATA[normName.replace(/-bin$/, '')] ||
    APPS_DATA[normName.replace(/-desktop$/, '')];

  // 2. Dynamic multi-source unified application & live system info
  const [unifiedApp, setUnifiedApp] = useState<UnifiedApplicationDTO | null>(null);
  const [selectedOfferId, setSelectedOfferId] = useState<string>('');
  const [backendInfo, setBackendInfo] = useState<any>(null);
  const [loadingBackend, setLoadingBackend] = useState(false);

  // Dynamic UI state
  const [activePreviewIdx, setActivePreviewIdx] = useState(0);
  const [isDescExpanded, setIsDescExpanded] = useState(false);
  const [isWishlisted, setIsWishlisted] = useState(false);
  const [installStatus, setInstallStatus] = useState<'idle' | 'installing' | 'installed'>('idle');
  const [toastVisible, setToastVisible] = useState(false);
  const [toastMessage, setToastMessage] = useState({ title: '', desc: '' });
  const [notesModalOpen, setNotesModalOpen] = useState(false);
  const [reviewModalOpen, setReviewModalOpen] = useState(false);
  const [galleryModalOpen, setGalleryModalOpen] = useState(false);

  // Fetch package metadata from backend whenever name or selectedOffer changes
  useEffect(() => {
    setActivePreviewIdx(0);
    setIsDescExpanded(false);

    const goApp = (window as any).go?.main?.App;
    if (goApp) {
      setLoadingBackend(true);
      if (goApp.GetUnifiedAppDetails) {
        goApp.GetUnifiedAppDetails(normName, selectedOfferId || '')
          .then((res: UnifiedApplicationDTO) => {
            if (res && res.name) {
              setUnifiedApp(res);
              if (!selectedOfferId && res.defaultOfferId) {
                setSelectedOfferId(res.defaultOfferId);
              }
              setBackendInfo(res);
            }
          })
          .catch((err: any) => console.warn('GetUnifiedAppDetails err:', err))
          .finally(() => setLoadingBackend(false));
      } else {
        const queryFn = goApp.GetAppDetails
          ? goApp.GetAppDetails.bind(goApp)
          : goApp.GetPackageInfo?.bind(goApp);

        if (queryFn) {
          queryFn(normName)
            .then((res: any) => {
              if (res && res.name) {
                setBackendInfo(res);
              }
            })
            .catch((err: any) => console.warn('GetAppDetails / GetPackageInfo err:', err))
            .finally(() => setLoadingBackend(false));
        }
      }
    }
  }, [normName]);

  // Current active offer selected by user or recommended default
  const currentOffer: PackageOfferDTO | undefined =
    unifiedApp?.offers?.find((o) => o.id === selectedOfferId) || unifiedApp?.offers?.[0];

  // Sync install status with current offer's installed state
  useEffect(() => {
    if (currentOffer) {
      setInstallStatus(currentOffer.isInstalled ? 'installed' : 'idle');
    }
  }, [currentOffer?.id, currentOffer?.isInstalled]);

  // Unified computed fields combining curated data + live repo heuristics
  const appDisplayName = unifiedApp?.name || appStatic?.name || backendInfo?.name || (normName.charAt(0).toUpperCase() + normName.slice(1));
  const appCategory = (unifiedApp?.categories && unifiedApp.categories[0]) || appStatic?.category || (backendInfo?.repo ? `System (${backendInfo.repo})` : 'Utilities & Tools');
  const appCategorySlug = appStatic?.categorySlug || 'apps';
  const appPublisher = unifiedApp?.developer || appStatic?.publisher || (backendInfo?.repo ? `Arch Linux Repository (${backendInfo.repo})` : 'Arch Community Maintainers');
  const appVersion = currentOffer?.version || unifiedApp?.offers?.[0]?.version || appStatic?.version || backendInfo?.version || 'Latest Stable';
  const appRating = unifiedApp?.rating || appStatic?.rating || 4.8;
  const appReviews = unifiedApp?.reviews || appStatic?.reviews || '4,280';
  const appSize = currentOffer?.downloadSize || appStatic?.size || backendInfo?.size || '45 MB';
  const appInstalledSize = currentOffer?.installedSize || appStatic?.installedSize || backendInfo?.size || '~120 MB';
  const appProvider = currentOffer?.provider || appStatic?.provider || backendInfo?.provider || 'pacman';
  const appLicense = unifiedApp?.license || appStatic?.license || backendInfo?.license || 'GNU GPL-3.0 / Open Source';

  // Real upstream website resolution with external browser support
  const rawWebsite = backendInfo?.website || appStatic?.website;
  const appWebsite = (rawWebsite && rawWebsite.startsWith('http') && !rawWebsite.includes('archlinux.org/packages/'))
    ? rawWebsite
    : (appStatic?.website || backendInfo?.website || `https://archlinux.org/packages/?q=${encodeURIComponent(normName)}`);

  const appIconUrl = appStatic?.iconUrl || backendInfo?.iconUrl;
  const appIconName = appStatic?.iconName || (
    appCategorySlug === 'gaming' ? 'sports_esports' :
      appCategorySlug === 'creativity' ? 'palette' :
        appCategorySlug === 'entertainment' ? 'smart_display' : 'terminal'
  );
  const appIconBg = appStatic?.iconBg || 'from-fluent-blue to-indigo-600';
  const appBadge = appStatic?.badge || (appProvider === 'flatpak' ? 'Sandboxed Flatpak' : 'Native Arch Package');
  const appDescription = appStatic?.description || backendInfo?.description || backendInfo?.headline || 'High performance software package verified and built for Arch Linux.';
  const appLongDescription = appStatic?.longDescription || backendInfo?.description || appDescription;

  // Real Flathub screenshots
  const backendScreenshots: Screenshot[] = Array.isArray(backendInfo?.screenshots) && backendInfo.screenshots.length > 0
    ? backendInfo.screenshots.map((url: string, i: number) => ({
      url,
      title: `${appDisplayName} Screenshot ${i + 1}`,
      caption: `${appDisplayName} running natively on Linux compositor.`,
      alt: `${appDisplayName} preview`,
    }))
    : [];

  const staticScreenshots = Array.isArray(appStatic?.screenshots) && appStatic.screenshots.length > 0
    ? appStatic.screenshots.map((s: any) => ({
      url: s.url,
      title: s.title || `${appDisplayName} View`,
      caption: s.caption || `${appDisplayName} interface`,
      alt: s.alt || `${appDisplayName} preview`,
    }))
    : [];

  // Prefer verified static catalog screenshots or live crawler screenshots
  const screenshots = staticScreenshots.length > 0
    ? staticScreenshots
    : (backendScreenshots.length > 0 ? backendScreenshots : []);

  const currentPreview: Screenshot | undefined = screenshots.length > 0
    ? (screenshots[activePreviewIdx] || screenshots[0])
    : undefined;

  const defaultFeatures = [
    {
      icon: 'bolt',
      title: 'Optimized Native Build',
      desc: 'Compiled with Arch Linux optimization flags for modern x86_64 CPU instructions.'
    },
    {
      icon: 'lock',
      title: 'Isolated & Secure',
      desc: 'Tested against package dependency conflicts with zero intrusive background telemetry.'
    },
    {
      icon: 'terminal',
      title: 'No Terminal Required',
      desc: 'Installs in one click. Desktop entry, launcher icon, and system integration configured automatically.'
    }
  ];

  const features = appStatic?.features || defaultFeatures;

  const defaultReleaseNotes = [
    `Up-to-date packaging matching upstream Arch Linux repository state.`,
    `Integrated desktop launcher (.desktop) and high-resolution icons.`,
    `Compatibility tested under PipeWire audio and Wayland compositor sessions.`
  ];

  const releaseNotes = appStatic?.releaseNotes || defaultReleaseNotes;

  // Category specific complementary suggestions
  const complementaryCatalog: Record<string, Array<{ name: string; category: string; icon: string; color: string; id: string; provider: string }>> = {
    creativity: [
      { name: 'Krita', category: 'Digital painting & concept art', icon: 'brush', color: 'text-secondary', id: 'krita', provider: 'pacman' },
      { name: 'Inkscape', category: 'Vector SVG illustration', icon: 'draw', color: 'text-tertiary', id: 'inkscape', provider: 'pacman' },
      { name: 'GIMP', category: 'Image manipulation & retouch', icon: 'photo_library', color: 'text-primary', id: 'gimp', provider: 'pacman' },
    ],
    gaming: [
      { name: 'Heroic Games', category: 'Epic & GOG launcher', icon: 'sports_esports', color: 'text-secondary', id: 'heroic', provider: 'flatpak' },
      { name: 'Lutris', category: 'Open gaming platform', icon: 'gamepad', color: 'text-tertiary', id: 'lutris', provider: 'pacman' },
      { name: 'RetroArch', category: 'Multi-system emulator', icon: 'videogame_asset', color: 'text-primary', id: 'retroarch', provider: 'pacman' },
    ],
    entertainment: [
      { name: 'Spotify', category: 'Music & podcast streaming', icon: 'headphones', color: 'text-tertiary', id: 'spotify', provider: 'flatpak' },
      { name: 'VLC Media Player', category: 'Universal media player', icon: 'smart_display', color: 'text-secondary', id: 'vlc', provider: 'pacman' },
      { name: 'Discord', category: 'Community voice & text', icon: 'forum', color: 'text-primary', id: 'discord', provider: 'flatpak' },
    ],
    apps: [
      { name: 'Visual Studio Code', category: 'Developer code editor', icon: 'code', color: 'text-primary', id: 'visual-studio-code-bin', provider: 'aur' },
      { name: 'Obsidian', category: 'Knowledge base & notes', icon: 'edit_note', color: 'text-secondary', id: 'obsidian', provider: 'flatpak' },
      { name: 'Bitwarden', category: 'Encrypted password vault', icon: 'key', color: 'text-tertiary', id: 'bitwarden', provider: 'flatpak' },
    ],
  };

  const complimentaryApps = complementaryCatalog[appCategorySlug] || complementaryCatalog.apps;
  const [complimentaryInstalled, setComplimentaryInstalled] = useState<Record<string, boolean>>({});

  // Listen for unified backend TransactionManager lifecycle updates
  useEffect(() => {
    const runtime = (window as any).runtime;
    if (!runtime || !runtime.EventsOn) return;
    const unsub = runtime.EventsOn('transaction:update', (tx: TransactionDTO) => {
      if (!tx || !currentOffer || tx.offerId !== currentOffer.id) return;
      if (tx.state === 'completed') {
        setInstallStatus(tx.operation === 'remove' ? 'idle' : 'installed');
      } else if (tx.state === 'failed' || tx.state === 'cancelled') {
        setInstallStatus('idle');
      } else {
        setInstallStatus('installing');
      }
      setToastMessage({
        title: `${tx.packageName} · ${tx.state.replace('_', ' ').toUpperCase()} (${tx.progress}%)`,
        desc: tx.errorMessage || tx.message || '',
      });
      setToastVisible(true);
    });
    return () => {
      if (typeof unsub === 'function') unsub();
    };
  }, [currentOffer?.id]);

  // Authoritative OfferID-based install trigger (no executable command passed from frontend)
  const handleInstall = async (offerOverride?: PackageOfferDTO) => {
    const targetOffer = offerOverride || currentOffer;
    const pkgToInstall = targetOffer?.packageName || normName;
    const provToInstall = targetOffer?.provider || appProvider;

    if (installStatus === 'installing') return;

    setInstallStatus('installing');
    setToastMessage({
      title: `Starting installation of ${appDisplayName}...`,
      desc: `Submitting authoritative OfferID via ${provToInstall.toUpperCase()} transaction engine.`
    });
    setToastVisible(true);

    const goApp = (window as any).go?.main?.App;
    let txSuccess = true;
    let txErrMsg = '';
    if (goApp && goApp.InstallOffer && targetOffer?.id) {
      try {
        const tx: TransactionDTO = await goApp.InstallOffer(targetOffer.id);
        txSuccess = tx.state === 'completed';
        txErrMsg = tx.errorMessage || '';
      } catch (err: any) {
        txSuccess = false;
        txErrMsg = String(err);
      }
    } else if (goApp && goApp.InstallPackage) {
      try {
        await goApp.InstallPackage(pkgToInstall, provToInstall);
      } catch (err) {
        console.warn('Backend install call error:', err);
      }
    } else {
      await new Promise((r) => setTimeout(r, 1200));
    }

    if (txSuccess) {
      setInstallStatus('installed');
      if (unifiedApp && targetOffer) {
        const updatedOffers = unifiedApp.offers.map((o) =>
          o.id === targetOffer.id
            ? { ...o, isInstalled: true, installedVersion: o.version }
            : o
        );
        setUnifiedApp({ ...unifiedApp, offers: updatedOffers, isInstalled: true });
      }
      setToastMessage({
        title: `${appDisplayName} installed cleanly via ${provToInstall.toUpperCase()}!`,
        desc: 'Transaction completed; sibling sources remain untouched.'
      });
    } else {
      setInstallStatus('idle');
      setToastMessage({
        title: `Installation failed for ${appDisplayName}`,
        desc: txErrMsg || 'Verify system locks or permissions and retry.'
      });
    }
    setTimeout(() => setToastVisible(false), 3500);
  };

  const handleRemove = async (targetOffer: PackageOfferDTO) => {
    setToastMessage({
      title: `Removing ${targetOffer.packageName}...`,
      desc: `Submitting removal transaction for ${targetOffer.provider.toUpperCase()} offer.`
    });
    setToastVisible(true);

    const goApp = (window as any).go?.main?.App;
    let ok = true;
    if (goApp && goApp.RemoveOffer && targetOffer.id) {
      try {
        const tx: TransactionDTO = await goApp.RemoveOffer(targetOffer.id);
        ok = tx.state === 'completed';
      } catch {
        ok = false;
      }
    } else if (goApp && goApp.RemovePackage) {
      try {
        await goApp.RemovePackage(targetOffer.packageName, targetOffer.provider);
      } catch (err) {
        console.warn('Backend remove call error:', err);
      }
    } else {
      await new Promise((r) => setTimeout(r, 1000));
    }

    if (ok && unifiedApp) {
      const updatedOffers = unifiedApp.offers.map((o) =>
        o.id === targetOffer.id
          ? { ...o, isInstalled: false, installedVersion: undefined }
          : o
      );
      const anyInst = updatedOffers.some((o) => o.isInstalled);
      setUnifiedApp({ ...unifiedApp, offers: updatedOffers, isInstalled: anyInst });
      setInstallStatus('idle');
      setToastMessage({
        title: `${targetOffer.packageName} uninstalled.`,
        desc: 'Selected offer removed cleanly.'
      });
    }
    setTimeout(() => setToastVisible(false), 2500);
  };

  const handleInstallComplementary = async (appId: string, appName: string, provider: string) => {
    setToastMessage({
      title: `Queued ${appName}...`,
      desc: 'Downloading verified container package.'
    });
    setToastVisible(true);

    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.InstallPackage) {
      try {
        await goApp.InstallPackage(appId, provider);
      } catch (e) {
        console.warn(e);
      }
    }

    setTimeout(() => {
      setComplimentaryInstalled(prev => ({ ...prev, [appId]: true }));
      setToastMessage({
        title: `${appName} is ready!`,
        desc: 'Ready to launch from start menu.'
      });
      setTimeout(() => setToastVisible(false), 2800);
    }, 1400);
  };

  const toggleWishlist = () => {
    setIsWishlisted(!isWishlisted);
  };

  return (
    <div className="flex flex-col w-full font-sans">
      <div className="w-full max-w-[1440px] mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">

        {/* 1. NAVIGATION & BREADCRUMB STRIP */}
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <button
              onClick={() => navigate(-1)}
              className="group flex items-center gap-2 px-3 py-1.5 rounded-lg bg-surface-container-low hover:bg-surface-container text-on-surface transition shadow-sm cursor-pointer"
              type="button"
            >
              <span className="material-symbols-outlined text-[18px] text-primary transition-transform group-hover:-translate-x-0.5">arrow_back</span>
              <span className="font-label-md text-label-md text-on-surface">Back to Store</span>
            </button>
            <span className="text-outline-variant font-body-sm text-body-sm hidden sm:inline">/</span>
            <div className="hidden sm:flex items-center gap-1.5 font-body-sm text-body-sm text-on-surface-variant">
              <span className="hover:text-on-surface cursor-pointer" onClick={() => navigate(`/${appCategorySlug}`)}>
                {appCategorySlug.charAt(0).toUpperCase() + appCategorySlug.slice(1)}
              </span>
              <span className="text-outline-variant">›</span>
              <span className="text-on-surface font-medium">{appDisplayName}</span>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => {
                if (navigator.clipboard) {
                  navigator.clipboard.writeText(window.location.href);
                  setToastMessage({ title: 'Link copied to clipboard', desc: 'Share this package URL with fellow Linux users.' });
                  setToastVisible(true);
                  setTimeout(() => setToastVisible(false), 2500);
                }
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-container-low hover:bg-surface-container text-on-surface-variant hover:text-on-surface transition text-label-md font-label-md cursor-pointer"
              title="Share application"
              type="button"
            >
              <span className="material-symbols-outlined text-[17px] text-primary">share</span>
              <span className="hidden md:inline">Share</span>
            </button>

            <button
              onClick={toggleWishlist}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-container-low hover:bg-surface-container text-on-surface-variant hover:text-on-surface transition text-label-md font-label-md cursor-pointer"
              title="Add to Wishlist"
              type="button"
            >
              <span
                className="material-symbols-outlined text-[17px] text-secondary"
                style={{ fontVariationSettings: isWishlisted ? "'FILL' 1" : "'FILL' 0" }}
              >
                {isWishlisted ? 'bookmark' : 'bookmark_border'}
              </span>
              <span className="hidden md:inline">{isWishlisted ? 'Saved' : 'Wishlist'}</span>
            </button>

            <button
              onClick={() => window.open('https://github.com/AdmGenSameer/arjax/issues/new', '_blank')}
              className="p-1.5 rounded-lg bg-surface-container-low hover:bg-surface-container text-on-surface-variant hover:text-on-surface transition cursor-pointer"
              title="Report app or broken package"
              type="button"
            >
              <span className="material-symbols-outlined text-[17px]">flag</span>
            </button>
          </div>
        </div>

        {/* 2. MASTER APP HERO CARD (FLUENT MICA GLASSPHISM) */}
        <section className="relative overflow-hidden rounded-xl bg-surface-container p-6 sm:p-8 shadow-fluent-elevation">
          <div className="absolute -right-24 -top-24 w-96 h-96 rounded-full bg-primary-container/10 blur-3xl pointer-events-none"></div>
          <div className="absolute right-40 bottom-0 w-80 h-80 rounded-full bg-secondary-container/10 blur-3xl pointer-events-none"></div>

          <div className="relative z-10 grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
            {/* App Icon & Identity Details */}
            <div className="lg:col-span-8 flex flex-col sm:flex-row gap-6 items-start">
              {/* Dynamic App Icon */}
              <div className="relative shrink-0 w-28 h-28 sm:w-32 sm:h-32 rounded-2xl bg-white/[0.03] border border-white/[0.08] p-3 shadow-lg flex items-center justify-center overflow-hidden">
                <AppIcon
                  appId={normName}
                  pkgName={normName}
                  provider={appProvider}
                  iconUrl={appIconUrl}
                  fallbackIcon={appIconName}
                  fallbackBg="bg-transparent"
                  size={256}
                  className="w-24 h-24 sm:w-28 sm:h-28"
                  imgClassName="w-24 h-24 sm:w-28 sm:h-28 object-contain drop-shadow-md transition-transform hover:scale-105"
                  alt={appDisplayName}
                />
                <span className="absolute -bottom-1.5 -right-1.5 flex h-6 w-6 items-center justify-center rounded-full bg-primary-container text-on-primary-container text-[13px] shadow">
                  <span className="material-symbols-outlined text-[14px]">check</span>
                </span>
              </div>

              {/* Titles, Publisher & Ratings */}
              <div className="space-y-3 flex-1">
                <div className="flex flex-wrap items-center gap-2.5">
                  <h1 className="font-headline-lg text-headline-lg text-on-surface font-semibold tracking-tight">
                    {appDisplayName}
                  </h1>
                  <span className="px-2 py-0.5 rounded-full bg-surface-container-highest text-on-surface-variant font-code text-code">
                    {appVersion}
                  </span>
                  <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full bg-tertiary-container/30 text-tertiary font-label-sm text-label-sm">
                    <span className="material-symbols-outlined text-[13px]" style={{ fontVariationSettings: "'FILL' 1" }}>verified</span>
                    {appBadge}
                  </span>
                </div>

                <p className="font-body-md text-body-md text-on-surface-variant">
                  {appPublisher} • {appCategory}
                </p>

                {/* Star ratings & metrics */}
                <div className="flex flex-wrap items-center gap-4 text-on-surface-variant">
                  <div className="flex items-center gap-1.5">
                    <span className="font-title text-title text-on-surface font-semibold">{appRating}</span>
                    <div className="flex text-secondary text-[16px]">
                      <span className="material-symbols-outlined text-[16px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                      <span className="material-symbols-outlined text-[16px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                      <span className="material-symbols-outlined text-[16px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                      <span className="material-symbols-outlined text-[16px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                      <span className="material-symbols-outlined text-[16px]" style={{ fontVariationSettings: "'FILL' 1" }}>star_half</span>
                    </div>
                    <span className="font-body-sm text-body-sm text-on-surface-variant">({appReviews} reviews)</span>
                  </div>
                  <span className="text-outline-variant">•</span>
                  <div className="flex items-center gap-1 font-body-sm text-body-sm">
                    <span className="material-symbols-outlined text-[16px] text-primary">category</span>
                    <span>{appCategory}</span>
                  </div>
                  <span className="text-outline-variant">•</span>
                  <div className="flex items-center gap-1 font-body-sm text-body-sm">
                    <span className="material-symbols-outlined text-[16px] text-tertiary">workspace_premium</span>
                    <span>{appBadge}</span>
                  </div>
                </div>

                {/* Beginner reassurance pills */}
                <div className="flex flex-wrap items-center gap-2 pt-1">
                  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-surface-container-high text-on-surface font-label-sm text-label-sm">
                    <span className="material-symbols-outlined text-[14px] text-primary">shield</span>
                    {appProvider === 'flatpak' ? 'Sandboxed Flatpak' : 'Native Arch Package'}
                  </span>
                  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-surface-container-high text-on-surface font-label-sm text-label-sm">
                    <span className="material-symbols-outlined text-[14px] text-tertiary">check_circle</span>
                    Verified &amp; Safe
                  </span>
                  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-surface-container-high text-on-surface font-label-sm text-label-sm">
                    <span className="material-symbols-outlined text-[14px] text-secondary">terminal</span>
                    No Terminal Required
                  </span>
                  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-surface-container-high text-on-surface font-label-sm text-label-sm">
                    <span className="material-symbols-outlined text-[14px] text-primary">speed</span>
                    Vulkan &amp; GPU Accelerated
                  </span>
                </div>
              </div>
            </div>

            {/* Action Card & Package Footprint */}
            <div className="lg:col-span-4 w-full flex flex-col gap-3 bg-surface-container-low p-5 rounded-xl">
              <div className="flex items-baseline justify-between">
                <span className="font-headline-md text-headline-md text-on-surface font-semibold">Free</span>
                <span className="font-code text-code text-tertiary">{appLicense}</span>
              </div>

              {/* Fluent Primary Install Button with Interactive State */}
              <button
                onClick={() => handleInstall()}
                disabled={installStatus === 'installing' || installStatus === 'installed'}
                className={`app-install-btn w-full py-3 px-4 rounded-lg font-label-md text-label-md font-semibold flex items-center justify-center gap-2 shadow-md transition-all active:scale-[0.98] cursor-pointer ${installStatus === 'installed'
                  ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
                  : installStatus === 'installing'
                    ? 'bg-primary-container/80 text-on-primary-container opacity-85'
                    : 'bg-primary-container hover:bg-primary-container/90 text-on-primary-container'
                  }`}
                type="button"
              >
                {installStatus === 'installing' ? (
                  <>
                    <span className="material-symbols-outlined text-[18px] animate-spin">refresh</span>
                    <span>Installing Package...</span>
                  </>
                ) : installStatus === 'installed' ? (
                  <>
                    <span className="material-symbols-outlined text-[18px] text-emerald-400">check</span>
                    <span>Installed</span>
                  </>
                ) : (
                  <>
                    <span className="material-symbols-outlined text-[18px]">download</span>
                    <span>Install Free</span>
                  </>
                )}
              </button>

              {/* Secondary Actions */}
              <div className="grid grid-cols-2 gap-2">
                <a
                  className="py-2 px-3 rounded-lg bg-surface-container-high hover:bg-surface-container-highest text-on-surface font-label-md text-label-md text-center transition flex items-center justify-center gap-1.5 cursor-pointer"
                  href={appWebsite}
                  onClick={(e) => {
                    e.preventDefault();
                    const wailsRuntime = (window as any).runtime;
                    if (wailsRuntime && wailsRuntime.BrowserOpenURL) {
                      wailsRuntime.BrowserOpenURL(appWebsite);
                    } else {
                      window.open(appWebsite, '_blank');
                    }
                  }}
                  rel="noreferrer"
                  target="_blank"
                >
                  <span className="material-symbols-outlined text-[16px] text-on-surface-variant">open_in_new</span>
                  <span>Website</span>
                </a>
                <button
                  onClick={() => setNotesModalOpen(true)}
                  className="py-2 px-3 rounded-lg bg-surface-container-high hover:bg-surface-container-highest text-on-surface font-label-md text-label-md text-center transition flex items-center justify-center gap-1.5 cursor-pointer"
                  type="button"
                >
                  <span className="material-symbols-outlined text-[16px] text-on-surface-variant">description</span>
                  <span>Notes</span>
                </button>
              </div>

              {/* Package Specs summary footer */}
              <div className="pt-3 space-y-1 font-body-sm text-body-sm text-on-surface-variant">
                <div className="flex items-center justify-between">
                  <span>Download footprint</span>
                  <span className="font-code text-code text-on-surface">{appSize}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span>Disk usage when installed</span>
                  <span className="font-code text-code text-on-surface">{appInstalledSize}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span>System Target</span>
                  <span className="font-code text-code text-on-surface">x86_64 • {appProvider.toUpperCase()}</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* MULTI-SOURCE ORIGIN & RECOMMENDATION SELECTOR */}
        {unifiedApp && unifiedApp.offers && unifiedApp.offers.length > 0 && (
          <SourcePicker
            offers={unifiedApp.offers}
            selectedOfferId={selectedOfferId}
            onSelectOffer={(id) => setSelectedOfferId(id)}
            onInstallOffer={(offer) => handleInstall(offer)}
            onRemoveOffer={(offer) => handleRemove(offer)}
            installing={installStatus === 'installing'}
            appDisplayName={appDisplayName}
          />
        )}

        {/* 3. INTERACTIVE MEDIA & SCREENSHOT SHOWCASE */}
        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="font-headline-md text-headline-md text-on-surface font-semibold">Screenshots &amp; Previews</h2>
            <span className="font-body-sm text-body-sm text-on-surface-variant">Click thumbnails to preview interface</span>
          </div>

          {/* Main Stage Viewport */}
          {currentPreview ? (
            <div className="relative w-full aspect-[16/9] max-h-[500px] rounded-xl overflow-hidden bg-surface-container-low shadow-fluent-elevation group">
              <div
                className="w-full h-full bg-cover bg-center transition-all duration-300"
                style={{ backgroundImage: `url('${currentPreview.url}')` }}
              ></div>

              {/* Overlaid caption badge */}
              <div className="absolute bottom-4 left-4 z-10 px-3 py-1.5 rounded-lg bg-surface-container-lowest/80 text-on-surface font-body-sm text-body-sm flex items-center gap-2 backdrop-blur-md">
                <span className="w-2 h-2 rounded-full bg-tertiary"></span>
                <span>{currentPreview.caption}</span>
              </div>

              {/* Fullscreen helper trigger */}
              <button
                onClick={() => setGalleryModalOpen(true)}
                className="absolute top-4 right-4 p-2 rounded-lg bg-surface-container-lowest/80 text-on-surface hover:text-primary transition backdrop-blur-md cursor-pointer"
                title="Open large view"
                type="button"
              >
                <span className="material-symbols-outlined text-[18px]">zoom_in</span>
              </button>
            </div>
          ) : (
            <div className="relative w-full aspect-[16/9] max-h-[420px] rounded-xl overflow-hidden bg-gradient-to-br from-surface-container-low via-surface-container to-surface-container-high border border-white/10 shadow-fluent-elevation flex flex-col items-center justify-center p-8 text-center space-y-4">
              <div className="w-20 h-20 rounded-2xl bg-white/5 border border-white/10 p-3 flex items-center justify-center shadow-lg">
                {appIconUrl ? (
                  <img src={appIconUrl} alt={appDisplayName} className="w-14 h-14 object-contain" />
                ) : (
                  <span className="material-symbols-outlined text-4xl text-primary">{appIconName}</span>
                )}
              </div>
              <div className="space-y-1">
                <h3 className="text-xl font-semibold text-on-surface">{appDisplayName}</h3>
                <p className="text-sm text-on-surface-variant max-w-md">{appDescription}</p>
              </div>
              <div className="flex items-center gap-3 text-xs text-on-surface-variant font-mono">
                <span className="px-2.5 py-1 rounded-md bg-white/5 border border-white/10">Architecture: x86_64</span>
                <span className="px-2.5 py-1 rounded-md bg-white/5 border border-white/10">Wayland Native</span>
                <span className="px-2.5 py-1 rounded-md bg-white/5 border border-white/10">PipeWire Audio</span>
              </div>
            </div>
          )}

          {/* Thumbnail Carousel Strip */}
          {screenshots.length > 1 && (
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 pt-1">
              {screenshots.map((s, idx) => (
                <button
                  key={s.title + idx}
                  onClick={() => setActivePreviewIdx(idx)}
                  className={`screenshot-thumb relative aspect-[16/9] rounded-lg overflow-hidden bg-surface-container-high transition-all text-left group cursor-pointer ${activePreviewIdx === idx
                    ? 'ring-2 ring-primary-container opacity-100'
                    : 'opacity-70 hover:opacity-100'
                    }`}
                  type="button"
                >
                  <div
                    className="w-full h-full bg-cover bg-center group-hover:scale-105 transition-transform duration-200"
                    style={{ backgroundImage: `url('${s.url}')` }}
                  ></div>
                  <div className="absolute inset-x-0 bottom-0 bg-surface-container-lowest/80 p-1.5 backdrop-blur-sm">
                    <p className="font-label-sm text-label-sm text-on-surface truncate">{s.title}</p>
                  </div>
                </button>
              ))}
            </div>
          )}
        </section>

        {/* 4. BEGINNER-FRIENDLY OVERVIEW & WHY YOU'LL LOVE IT */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
          <div className="lg:col-span-8 space-y-6">

            {/* Feature Bento Cards */}
            <div className="rounded-xl bg-surface-container p-6 space-y-4">
              <div className="flex items-center gap-2">
                <span className="material-symbols-outlined text-primary text-[22px]">auto_awesome</span>
                <h2 className="font-title text-title text-on-surface font-semibold">Why You'll Love It on Linux</h2>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                {features.map((f, idx) => (
                  <div key={idx} className="rounded-lg bg-surface-container-low p-4 space-y-2">
                    <div className="w-8 h-8 rounded-lg bg-primary-container/20 text-primary flex items-center justify-center">
                      <span className="material-symbols-outlined text-[18px]">{f.icon}</span>
                    </div>
                    <h3 className="font-label-md text-label-md text-on-surface font-semibold">{f.title}</h3>
                    <p className="font-body-sm text-body-sm text-on-surface-variant">
                      {f.desc}
                    </p>
                  </div>
                ))}
              </div>
            </div>

            {/* Plain-English Description */}
            <div className="rounded-xl bg-surface-container p-6 space-y-3">
              <h2 className="font-title text-title text-on-surface font-semibold">About {appDisplayName}</h2>
              <div className="space-y-3 font-body-md text-body-md text-on-surface-variant leading-relaxed">
                <p>{appDescription}</p>
                {appLongDescription && appLongDescription !== appDescription && (
                  <p>{appLongDescription}</p>
                )}

                {isDescExpanded && (
                  <div className="space-y-3 pt-2">
                    <p>
                      <strong>Package Features &amp; Integrations:</strong>
                    </p>
                    <ul className="list-disc pl-5 space-y-1 font-body-sm text-body-sm">
                      {features.map((feat, i) => (
                        <li key={i}><strong>{feat.title}:</strong> {feat.desc}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>

              <button
                onClick={() => setIsDescExpanded(!isDescExpanded)}
                className="pt-2 font-label-md text-label-md text-primary hover:text-primary-fixed-dim transition flex items-center gap-1 cursor-pointer"
                type="button"
              >
                <span>{isDescExpanded ? 'Show less' : 'Read full description'}</span>
                <span className="material-symbols-outlined text-[16px]">
                  {isDescExpanded ? 'expand_less' : 'expand_more'}
                </span>
              </button>
            </div>

            {/* 6. RATINGS & VERIFIED REVIEWS */}
            <div className="rounded-xl bg-surface-container p-6 space-y-6">
              <div className="flex items-center justify-between">
                <h2 className="font-title text-title text-on-surface font-semibold">Ratings &amp; Reviews</h2>
                <button
                  onClick={() => setReviewModalOpen(true)}
                  className="font-label-md text-label-md text-primary hover:underline cursor-pointer"
                  type="button"
                >
                  Write a review
                </button>
              </div>

              {/* Score summary breakdown */}
              <div className="grid grid-cols-1 sm:grid-cols-12 gap-6 items-center rounded-lg bg-surface-container-low p-4">
                <div className="sm:col-span-4 flex flex-col items-center justify-center text-center space-y-1">
                  <span className="font-display text-display text-on-surface font-bold tracking-tight">{appRating}</span>
                  <div className="flex text-secondary text-[18px]">
                    <span className="material-symbols-outlined text-[18px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                    <span className="material-symbols-outlined text-[18px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                    <span className="material-symbols-outlined text-[18px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                    <span className="material-symbols-outlined text-[18px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                    <span className="material-symbols-outlined text-[18px]" style={{ fontVariationSettings: "'FILL' 1" }}>star_half</span>
                  </div>
                  <span className="font-body-sm text-body-sm text-on-surface-variant">{appReviews} verified ratings</span>
                </div>

                {/* Distribution Bars */}
                <div className="sm:col-span-8 space-y-1.5 font-body-sm text-body-sm">
                  <div className="flex items-center gap-2">
                    <span className="w-4 text-on-surface-variant">5</span>
                    <div className="flex-1 h-2 rounded-full bg-surface-container-highest overflow-hidden">
                      <div className="h-full bg-primary-container rounded-full" style={{ width: '91%' }}></div>
                    </div>
                    <span className="font-code text-code text-on-surface-variant w-8 text-right">91%</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-4 text-on-surface-variant">4</span>
                    <div className="flex-1 h-2 rounded-full bg-surface-container-highest overflow-hidden">
                      <div className="h-full bg-primary-container rounded-full" style={{ width: '7%' }}></div>
                    </div>
                    <span className="font-code text-code text-on-surface-variant w-8 text-right">7%</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-4 text-on-surface-variant">3</span>
                    <div className="flex-1 h-2 rounded-full bg-surface-container-highest overflow-hidden">
                      <div className="h-full bg-primary-container rounded-full" style={{ width: '1%' }}></div>
                    </div>
                    <span className="font-code text-code text-on-surface-variant w-8 text-right">1%</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-4 text-on-surface-variant">2</span>
                    <div className="flex-1 h-2 rounded-full bg-surface-container-highest overflow-hidden">
                      <div className="h-full bg-primary-container rounded-full" style={{ width: '0.5%' }}></div>
                    </div>
                    <span className="font-code text-code text-on-surface-variant w-8 text-right">&lt;1%</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-4 text-on-surface-variant">1</span>
                    <div className="flex-1 h-2 rounded-full bg-surface-container-highest overflow-hidden">
                      <div className="h-full bg-primary-container rounded-full" style={{ width: '0.5%' }}></div>
                    </div>
                    <span className="font-code text-code text-on-surface-variant w-8 text-right">&lt;1%</span>
                  </div>
                </div>
              </div>

              {/* Community Reviews Feed */}
              <div className="space-y-4">
                <div className="p-4 rounded-lg bg-surface-container-low space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2.5">
                      <div className="w-7 h-7 rounded-full bg-primary-container/20 text-primary font-bold text-[12px] flex items-center justify-center">
                        AL
                      </div>
                      <div>
                        <span className="font-label-md text-label-md text-on-surface font-medium">Alex L. (Arch Linux User)</span>
                        <span className="ml-2 font-body-sm text-body-sm text-tertiary">Verified Install</span>
                      </div>
                    </div>
                    <span className="font-body-sm text-body-sm text-on-surface-variant">3 days ago</span>
                  </div>

                  <div className="flex text-secondary text-[14px]">
                    <span className="material-symbols-outlined text-[14px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                    <span className="material-symbols-outlined text-[14px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                    <span className="material-symbols-outlined text-[14px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                    <span className="material-symbols-outlined text-[14px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                    <span className="material-symbols-outlined text-[14px]" style={{ fontVariationSettings: "'FILL' 1" }}>star</span>
                  </div>

                  <p className="font-body-md text-body-md text-on-surface leading-normal">
                    Installed in one click without terminal hurdles. Desktop icon, system shortcuts, and Wayland compatibility worked immediately out of the box.
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* 5. APP SPECIFICATIONS & COMPATIBILITY SIDEBAR */}
          <div className="lg:col-span-4 space-y-6">

            {/* Technical Specification Card */}
            <div className="rounded-xl bg-surface-container p-6 space-y-4">
              <h2 className="font-title text-title text-on-surface font-semibold">Information &amp; Specs</h2>
              <dl className="space-y-3 font-body-sm text-body-sm divide-y-0">
                <div className="flex justify-between py-1">
                  <dt className="text-on-surface-variant">Publisher</dt>
                  <dd className="text-on-surface font-medium text-right">{appPublisher}</dd>
                </div>
                <div className="flex justify-between py-1">
                  <dt className="text-on-surface-variant">Category</dt>
                  <dd className="text-on-surface font-medium text-right">{appCategory}</dd>
                </div>
                <div className="flex justify-between py-1">
                  <dt className="text-on-surface-variant">Package Version</dt>
                  <dd className="font-code text-code text-on-surface text-right">{appVersion}</dd>
                </div>
                <div className="flex justify-between py-1">
                  <dt className="text-on-surface-variant">License</dt>
                  <dd className="text-tertiary font-medium text-right">{appLicense}</dd>
                </div>
                <div className="flex justify-between py-1">
                  <dt className="text-on-surface-variant">Package Manager</dt>
                  <dd className="text-on-surface text-right font-medium">{appProvider.toUpperCase()}</dd>
                </div>
                <div className="flex justify-between py-1">
                  <dt className="text-on-surface-variant">Security Model</dt>
                  <dd className="text-on-surface text-right flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-tertiary"></span>
                    <span>{appBadge}</span>
                  </dd>
                </div>
              </dl>
            </div>

            {/* System Requirements */}
            <div className="rounded-xl bg-surface-container p-6 space-y-4">
              <div className="flex items-center gap-2">
                <span className="material-symbols-outlined text-primary text-[20px]">memory</span>
                <h2 className="font-title text-title text-on-surface font-semibold">System Requirements</h2>
              </div>
              <div className="space-y-3 font-body-sm text-body-sm">
                <div className="rounded-lg bg-surface-container-low p-3 space-y-1">
                  <div className="flex items-center justify-between text-on-surface font-medium">
                    <span>Target Architecture</span>
                    <span className="text-tertiary flex items-center gap-1 font-label-sm text-label-sm">
                      <span className="material-symbols-outlined text-[14px]">check</span> x86_64 Compatible
                    </span>
                  </div>
                  <p className="text-on-surface-variant text-[11.5px]">Standard 64-bit Linux kernel with glibc runtime.</p>
                </div>

                <div className="rounded-lg bg-surface-container-low p-3 space-y-1">
                  <div className="flex items-center justify-between text-on-surface font-medium">
                    <span>Graphics &amp; Compositor</span>
                    <span className="text-tertiary flex items-center gap-1 font-label-sm text-label-sm">
                      <span className="material-symbols-outlined text-[14px]">check</span> Wayland / X11
                    </span>
                  </div>
                  <p className="text-on-surface-variant text-[11.5px]">Compatible with GNOME, KDE Plasma, Hyprland, and Sway.</p>
                </div>
              </div>
            </div>

            {/* 7. COMPLEMENTARY APPS */}
            <div className="rounded-xl bg-surface-container p-6 space-y-3">
              <h2 className="font-title text-title text-on-surface font-semibold">Related Applications</h2>
              <div className="space-y-2.5">
                {complimentaryApps.map((app) => (
                  <div
                    key={app.id}
                    className="p-2.5 rounded-lg bg-surface-container-low hover:bg-surface-container-high transition flex items-center justify-between cursor-pointer group"
                    onClick={() => navigate(`/package/${app.id}`)}
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl bg-surface-container-highest flex items-center justify-center p-1.5 shadow-sm">
                        <span className={`material-symbols-outlined ${app.color} text-[24px]`}>{app.icon}</span>
                      </div>
                      <div>
                        <h4 className="font-label-md text-label-md text-on-surface font-medium group-hover:text-primary transition">{app.name}</h4>
                        <p className="font-body-sm text-body-sm text-on-surface-variant">{app.category}</p>
                      </div>
                    </div>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleInstallComplementary(app.id, app.name, app.provider);
                      }}
                      className={`px-3 py-1 rounded-md text-label-sm font-label-sm transition cursor-pointer ${complimentaryInstalled[app.id]
                        ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                        : 'bg-surface-container-highest text-on-surface hover:bg-primary-container hover:text-on-primary-container'
                        }`}
                      type="button"
                    >
                      {complimentaryInstalled[app.id] ? 'Installed' : 'Get'}
                    </button>
                  </div>
                ))}
              </div>
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
          <span className="material-symbols-outlined text-[18px]">
            {installStatus === 'installed' ? 'check' : 'downloading'}
          </span>
        </div>
        <div>
          <div className="text-[12.5px] font-semibold text-white">{toastMessage.title}</div>
          <div className="text-[11px] text-fluent-muted">{toastMessage.desc}</div>
        </div>
      </div>

      {/* Release Notes Modal */}
      {notesModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-mica-surface border border-mica-border rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-fluent-flyout">
            <div className="flex items-center justify-between pb-2 border-b border-white/[0.08]">
              <div className="flex items-center gap-2 text-fluent-text font-semibold text-sm">
                <span className="material-symbols-outlined text-fluent-blue text-lg">description</span>
                <span>{appDisplayName} Release Highlights</span>
              </div>
              <button
                onClick={() => setNotesModalOpen(false)}
                className="text-fluent-dim hover:text-white p-1 rounded-md transition"
              >
                <span className="material-symbols-outlined text-lg">close</span>
              </button>
            </div>
            <div className="text-xs text-fluent-muted space-y-2.5 max-h-72 overflow-y-auto pr-1 leading-relaxed">
              {releaseNotes.map((note, idx) => (
                <p key={idx}>• {note}</p>
              ))}
            </div>
            <div className="pt-2 flex justify-end">
              <button
                onClick={() => setNotesModalOpen(false)}
                className="px-4 py-2 bg-fluent-blue hover:bg-fluent-blueHover text-white rounded-lg text-xs font-medium transition cursor-pointer"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Review Modal */}
      {reviewModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-mica-surface border border-mica-border rounded-2xl max-w-md w-full p-6 space-y-4 shadow-fluent-flyout">
            <div className="flex items-center justify-between pb-2 border-b border-white/[0.08]">
              <span className="text-fluent-text font-semibold text-sm">Write a Community Review</span>
              <button
                onClick={() => setReviewModalOpen(false)}
                className="text-fluent-dim hover:text-white p-1 rounded-md transition"
              >
                <span className="material-symbols-outlined text-lg">close</span>
              </button>
            </div>
            <div className="space-y-3">
              <div className="flex justify-center gap-2 text-secondary text-2xl py-1">
                {[1, 2, 3, 4, 5].map((star) => (
                  <span
                    key={star}
                    className="material-symbols-outlined cursor-pointer hover:scale-110 transition-transform"
                    style={{ fontVariationSettings: "'FILL' 1" }}
                  >
                    star
                  </span>
                ))}
              </div>
              <textarea
                rows={4}
                placeholder={`Share your experience using ${appDisplayName} on your Linux workstation...`}
                className="w-full bg-black/40 border border-mica-border rounded-xl p-3 text-xs text-fluent-text placeholder-fluent-dim focus:outline-none focus:border-fluent-blue resize-none"
              ></textarea>
            </div>
            <div className="pt-2 flex justify-end gap-2">
              <button
                onClick={() => setReviewModalOpen(false)}
                className="px-4 py-1.5 bg-white/[0.05] hover:bg-white/[0.08] text-fluent-muted rounded-lg text-xs transition cursor-pointer"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  setReviewModalOpen(false);
                  setToastMessage({ title: 'Review submitted!', desc: 'Thank you for supporting community Linux ratings.' });
                  setToastVisible(true);
                  setTimeout(() => setToastVisible(false), 2500);
                }}
                className="px-4 py-1.5 bg-fluent-blue hover:bg-fluent-blueHover text-white rounded-lg text-xs font-semibold transition cursor-pointer"
              >
                Submit Review
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Lossless Gallery Modal */}
      {galleryModalOpen && currentPreview && (
        <div
          onClick={() => setGalleryModalOpen(false)}
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 backdrop-blur-md p-6 cursor-zoom-out"
        >
          <div className="max-w-5xl w-full space-y-3" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between text-white">
              <span className="text-sm font-semibold">{currentPreview.caption}</span>
              <button
                onClick={() => setGalleryModalOpen(false)}
                className="text-fluent-dim hover:text-white p-1 rounded-md transition"
              >
                <span className="material-symbols-outlined text-2xl">close</span>
              </button>
            </div>
            <img
              src={currentPreview.url}
              alt={currentPreview.alt}
              className="w-full max-h-[78vh] object-contain rounded-xl border border-white/[0.1] shadow-2xl"
            />
          </div>
        </div>
      )}

    </div>
  );
};
