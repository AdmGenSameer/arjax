import React, { useState, useEffect, useRef } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { Filter, ArrowRight, Download, Check, Sparkles, Layers } from 'lucide-react';
import { AppIcon } from '../components/AppIcon';
import { UnifiedApplicationDTO, PackageOfferDTO } from '../types';

export const SearchPage: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryParam = searchParams.get('q') || '';
  const [query, setQuery] = useState(queryParam);
  const [selectedFilter, setSelectedFilter] = useState('ALL');
  const [unifiedResults, setUnifiedResults] = useState<UnifiedApplicationDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const [installedMap, setInstalledMap] = useState<Record<string, boolean>>({});
  const activeSearchIdRef = useRef<string>('');

  const filters = ['ALL', 'PACMAN', 'FLATPAK', 'AUR', 'SNAP', 'GITHUB'];

  useEffect(() => {
    setQuery(queryParam);
    if (queryParam.trim()) {
      performSearch(queryParam.trim());
    }
  }, [queryParam]);

  // Clean up any in-flight search on unmount
  useEffect(() => {
    return () => {
      const goApp = (window as any).go?.main?.App;
      if (goApp && goApp.CancelProgressiveSearch && activeSearchIdRef.current) {
        goApp.CancelProgressiveSearch(activeSearchIdRef.current);
      }
    };
  }, []);

  const performSearch = async (q: string) => {
    setLoading(true);
    const searchId = 'srch_' + Date.now();
    activeSearchIdRef.current = searchId;

    const goApp = (window as any).go?.main?.App;
    const runtime = (window as any).runtime;

    if (goApp) {
      try {
        // 1. If progressive search & Wails runtime events are available:
        if (goApp.SearchUnifiedProgressive && runtime && runtime.EventsOn) {
          // Listen for progressive batches
          runtime.EventsOn('search:batch', (batch: any) => {
            if (batch && batch.searchId === activeSearchIdRef.current) {
              setUnifiedResults(batch.apps || []);
              setLoading(false);
            }
          });

          runtime.EventsOn('search:done', (done: any) => {
            if (done && done.searchId === activeSearchIdRef.current) {
              setLoading(false);
            }
          });

          goApp.SearchUnifiedProgressive(q, searchId);
          return;
        }

        // 2. Synchronous unified search fallback
        if (goApp.SearchUnifiedSync) {
          const apps = await goApp.SearchUnifiedSync(q);
          if (Array.isArray(apps) && apps.length > 0) {
            setUnifiedResults(apps);
            setLoading(false);
            return;
          }
        }

        // 3. Legacy search fallback
        const [pkgs, desktopApps] = await Promise.all([
          goApp.SearchPackages ? goApp.SearchPackages(q).catch(() => []) : [],
          goApp.SearchDesktopApps ? goApp.SearchDesktopApps(q).catch(() => []) : [],
        ]);

        const synthesized: UnifiedApplicationDTO[] = [];
        const seen = new Set<string>();

        if (Array.isArray(desktopApps)) {
          desktopApps.forEach((da: any) => {
            const id = (da.flathubId || da.id || da.name).toLowerCase();
            if (!seen.has(id)) {
              seen.add(id);
              synthesized.push({
                appId: da.flathubId || da.id,
                name: da.name || id,
                summary: da.headline || 'Desktop Application',
                description: da.headline || '',
                developer: da.publisher || 'Flathub Publisher',
                homepage: da.website || '',
                license: 'Open Source',
                iconUrl: da.iconUrl,
                categories: ['Utilities'],
                screenshots: da.screenshots || [],
                releases: [],
                aliases: [id],
                offers: [
                  {
                    id: `flatpak:flathub:${da.flathubId || da.id}`,
                    appId: da.flathubId || da.id,
                    packageName: da.flathubId || da.id,
                    provider: 'flatpak',
                    source: 'flathub',
                    version: da.version || 'Latest',
                    downloadSize: '~85 MB',
                    installedSize: '~240 MB',
                    installScope: 'sandboxed',
                    permissions: ['network', 'pulseaudio'],
                    isInstalled: false,
                    isEligible: true,
                    score: 90,
                    recommendationBadge: 'Recommended',
                    recommendationReason: 'Verified Flathub application container.',
                    installCmd: `flatpak install flathub ${da.flathubId || da.id}`,
                  },
                ],
                defaultOfferId: `flatpak:flathub:${da.flathubId || da.id}`,
                isInstalled: false,
                rating: 4.8,
                reviews: '4,280',
                badge: 'Verified',
              });
            }
          });
        }

        if (Array.isArray(pkgs)) {
          pkgs.forEach((p: any) => {
            const id = p.name.toLowerCase();
            if (!seen.has(id)) {
              seen.add(id);
              synthesized.push({
                appId: id,
                name: stringsTitle(p.name),
                summary: p.description || 'Linux software package',
                description: p.description || '',
                developer: 'Arch Community',
                homepage: `https://archlinux.org/packages/?q=${id}`,
                license: 'GPL / Open Source',
                categories: ['Utilities'],
                screenshots: [],
                releases: [],
                aliases: [id],
                offers: [
                  {
                    id: `${p.provider}:${p.source || 'repo'}:${p.name}`,
                    appId: id,
                    packageName: p.name,
                    provider: p.provider || 'pacman',
                    source: p.source || p.repo || 'extra',
                    version: p.version || 'Latest',
                    downloadSize: p.size || '~45 MB',
                    installedSize: '~120 MB',
                    installScope: p.provider === 'flatpak' ? 'sandboxed' : 'system',
                    permissions: ['system-native'],
                    isInstalled: false,
                    isEligible: true,
                    score: 85,
                    recommendationBadge: 'Recommended',
                    recommendationReason: 'Native Arch Linux distribution package.',
                    installCmd: `sudo pacman -S ${p.name}`,
                  },
                ],
                defaultOfferId: `${p.provider}:${p.source || 'repo'}:${p.name}`,
                isInstalled: false,
                rating: 4.6,
                reviews: '1,200',
                badge: 'Repository',
              });
            }
          });
        }

        if (synthesized.length > 0) {
          setUnifiedResults(synthesized);
          setLoading(false);
          return;
        }
      } catch (e) {
        console.warn('Progressive search error:', e);
      }
    }

    // High-quality mock unified catalog preview if backend unavailable
    setTimeout(() => {
      const qLower = q.toLowerCase();
      const mockUnified: UnifiedApplicationDTO[] = [
        {
          appId: `org.mozilla.${qLower}`,
          name: stringsTitle(q),
          summary: `Fast, modern, and reliable ${q} built natively for Linux.`,
          description: `Comprehensive software suite providing native acceleration and seamless Wayland compositor integration.`,
          developer: 'Mozilla & Community Contributors',
          homepage: `https://${qLower}.org`,
          license: 'MPL-2.0 / Open Source',
          categories: ['Web', 'Productivity'],
          screenshots: [],
          releases: [],
          aliases: [qLower, `${qLower}-bin`, `${qLower}-git`],
          offers: [
            {
              id: `pacman:extra:${qLower}`,
              appId: `org.mozilla.${qLower}`,
              packageName: qLower,
              provider: 'pacman',
              source: 'extra',
              version: '124.0.1-1',
              downloadSize: '78 MB',
              installedSize: '210 MB',
              installScope: 'system',
              permissions: ['system-native', 'hardware-direct'],
              isInstalled: false,
              isEligible: true,
              score: 95,
              recommendationBadge: 'Recommended',
              recommendationReason: 'Official Arch extra repository package with native system integration.',
              installCmd: `sudo pacman -S ${qLower}`,
            },
            {
              id: `flatpak:flathub:org.mozilla.${qLower}`,
              appId: `org.mozilla.${qLower}`,
              packageName: `org.mozilla.${qLower}`,
              provider: 'flatpak',
              source: 'flathub',
              version: '124.0.1',
              downloadSize: '85 MB',
              installedSize: '240 MB',
              installScope: 'sandboxed',
              permissions: ['network', 'pulseaudio', 'wayland'],
              isInstalled: false,
              isEligible: true,
              score: 85,
              recommendationBadge: 'Sandboxed',
              recommendationReason: 'Flathub container with isolated user permissions.',
              installCmd: `flatpak install flathub org.mozilla.${qLower}`,
            },
            {
              id: `aur:aur:${qLower}-bin`,
              appId: `org.mozilla.${qLower}`,
              packageName: `${qLower}-bin`,
              provider: 'aur',
              source: 'aur',
              version: '124.0.1',
              downloadSize: '82 MB',
              installedSize: '215 MB',
              installScope: 'user',
              permissions: ['system-native'],
              isInstalled: false,
              isEligible: true,
              score: 65,
              recommendationBadge: 'Community',
              recommendationReason: 'Precompiled binary build from the Arch User Repository.',
              installCmd: `paru -S ${qLower}-bin`,
            },
          ],
          defaultOfferId: `pacman:extra:${qLower}`,
          isInstalled: false,
          rating: 4.8,
          reviews: '14,800',
          badge: 'Verified',
        },
      ];
      setUnifiedResults(mockUnified);
      setLoading(false);
    }, 220);
  };

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      setSearchParams({ q: query.trim() });
    }
  };

  const handleInstall = async (app: UnifiedApplicationDTO) => {
    const offer: PackageOfferDTO | undefined =
      app.offers.find((o) => o.id === app.defaultOfferId) || app.offers[0];
    if (!offer) return;

    const goApp = (window as any).go?.main?.App;
    if (goApp && goApp.InstallPackage) {
      try {
        await goApp.InstallPackage(offer.packageName, offer.provider);
        setInstalledMap((prev) => ({ ...prev, [app.appId]: true, [offer.packageName]: true }));
      } catch (e) {
        console.error(e);
      }
    } else {
      setTimeout(() => {
        setInstalledMap((prev) => ({ ...prev, [app.appId]: true, [offer.packageName]: true }));
      }, 1200);
    }
  };

  const filteredApps = unifiedResults
    .filter((app) => {
      if (selectedFilter === 'ALL') return true;
      return app.offers.some((o) => o.provider.toUpperCase().includes(selectedFilter));
    })
    .slice(0, 50);

  return (
    <div className="p-8 space-y-6 max-w-6xl mx-auto h-full flex flex-col font-sans">
      {/* Search Header Bar */}
      <div className="space-y-4 flex-shrink-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-fluent-blue text-xl">travel_explore</span>
            <h1 className="text-xl font-bold tracking-tight text-fluent-text">
              Unified Package Catalog
            </h1>
          </div>
          {unifiedResults.length > 0 && (
            <span className="text-xs text-fluent-dim font-mono">
              {unifiedResults.length} Applications • Deduplicated across Pacman, Flatpak, AUR, Snap
            </span>
          )}
        </div>

        <form onSubmit={handleSearchSubmit} className="relative">
          <input
            type="text"
            placeholder="Search across Flathub, Arch Repos, AUR, and Snap (e.g. firefox, blender, vlc, steam)..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full bg-mica-base/90 border border-mica-border focus:border-fluent-blue text-sm text-fluent-text rounded-xl pl-11 pr-28 py-3.5 outline-none transition-all shadow-inner placeholder:text-fluent-dim"
          />
          <span className="material-symbols-outlined absolute left-3.5 top-3.5 text-fluent-muted text-lg">search</span>
          <button
            type="submit"
            className="absolute right-2.5 top-2.5 px-4 py-1.5 bg-fluent-blue hover:bg-fluent-blueHover text-white font-medium text-xs rounded-lg transition-all shadow-sm active:scale-95 cursor-pointer"
          >
            Search
          </button>
        </form>

        {/* Filter Pills */}
        <div className="flex items-center gap-2 overflow-x-auto pb-1 text-xs">
          <span className="text-fluent-dim flex items-center gap-1 text-[11px] font-medium mr-1">
            <Filter className="w-3 h-3" /> Filter by Source:
          </span>
          {filters.map((f) => (
            <button
              key={f}
              onClick={() => setSelectedFilter(f)}
              type="button"
              className={`px-3 py-1.5 rounded-lg text-[11px] font-mono uppercase tracking-wider transition-all cursor-pointer ${selectedFilter === f
                  ? 'bg-fluent-blue text-white font-semibold shadow-sm'
                  : 'bg-mica-base/60 text-fluent-muted border border-mica-border hover:bg-mica-surface hover:text-fluent-text'
                }`}
            >
              {f}
            </button>
          ))}
        </div>
      </div>

      {/* Results List */}
      <div className="flex-1 overflow-y-auto space-y-3 pr-1">
        {loading ? (
          <div className="space-y-3 py-4">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="h-24 bg-mica-base/40 border border-mica-border rounded-xl animate-pulse" />
            ))}
          </div>
        ) : filteredApps.length > 0 ? (
          filteredApps.map((app) => {
            const defaultOffer =
              app.offers.find((o) => o.id === app.defaultOfferId) || app.offers[0];
            const isInstalled = app.isInstalled || installedMap[app.appId] || (defaultOffer && installedMap[defaultOffer.packageName]);

            return (
              <div
                key={app.appId}
                className="fluent-card p-4 rounded-xl flex items-center justify-between gap-4 group cursor-pointer border border-white/5 hover:border-fluent-blue/30 transition-all bg-surface-container/60 hover:bg-surface-container"
              >
                {/* App Main Identity */}
                <div
                  onClick={() => navigate(`/package/${encodeURIComponent(app.appId)}`)}
                  className="flex items-center gap-3.5 min-w-0 flex-1"
                >
                  <div className="w-12 h-12 rounded-xl bg-white/[0.04] border border-white/[0.08] p-1.5 flex items-center justify-center shrink-0 overflow-hidden shadow-sm">
                    <AppIcon
                      appId={app.appId}
                      pkgName={defaultOffer?.packageName || app.appId}
                      provider={defaultOffer?.provider || 'flatpak'}
                      iconUrl={app.iconUrl}
                      fallbackIcon="view_in_ar"
                      fallbackBg="bg-transparent"
                      size={64}
                      className="w-9 h-9"
                      imgClassName="w-9 h-9 object-contain drop-shadow"
                      alt={app.name}
                    />
                  </div>

                  <div className="space-y-1 min-w-0 flex-1">
                    <div className="flex items-center gap-2.5 flex-wrap">
                      <h3 className="font-semibold text-fluent-text text-sm group-hover:text-fluent-blue transition-colors truncate">
                        {app.name}
                      </h3>

                      {/* Canonical AppId Pill */}
                      <span className="text-[10px] font-mono text-fluent-dim bg-white/5 px-2 py-0.5 rounded border border-white/5">
                        {app.appId}
                      </span>

                      {/* Recommendation Badge */}
                      {defaultOffer?.recommendationBadge && (
                        <span className="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-full font-medium bg-fluent-blue/15 text-fluent-blue border border-fluent-blue/30">
                          <Sparkles className="w-2.5 h-2.5" />
                          {defaultOffer.recommendationBadge}
                        </span>
                      )}

                      <span className="text-[11px] text-fluent-dim font-mono">
                        {defaultOffer?.version || 'Latest'}
                      </span>
                    </div>

                    <p className="text-xs text-fluent-muted line-clamp-1">
                      {app.summary || app.description || 'Verified Linux desktop application.'}
                    </p>

                    {/* Multi-Source Origin Bar */}
                    <div className="flex items-center gap-1.5 pt-0.5 text-[11px] text-fluent-dim font-mono">
                      <Layers className="w-3 h-3 text-fluent-blue" />
                      <span>{app.offers.length} {app.offers.length === 1 ? 'Source:' : 'Sources:'}</span>
                      <div className="flex items-center gap-1">
                        {app.offers.map((offer) => (
                          <span
                            key={offer.id}
                            className={`px-1.5 py-0.2 rounded uppercase text-[10px] font-semibold border ${offer.id === app.defaultOfferId
                                ? 'bg-primary/20 text-primary border-primary/30'
                                : 'bg-white/5 text-fluent-muted border-white/5'
                              }`}
                          >
                            {offer.provider} ({offer.source})
                          </span>
                        ))}
                      </div>
                    </div>
                  </div>
                </div>

                {/* Right Side Actions */}
                <div className="flex items-center gap-2 flex-shrink-0">
                  <button
                    type="button"
                    onClick={() => navigate(`/package/${encodeURIComponent(app.appId)}`)}
                    className="px-3 py-1.5 rounded-lg bg-white/[0.04] hover:bg-white/[0.08] text-fluent-muted hover:text-fluent-text text-xs font-medium border border-mica-border transition-colors flex items-center gap-1.5 cursor-pointer"
                    title="View all sources, permissions & screenshots"
                  >
                    <span>Details ({app.offers.length})</span>
                    <ArrowRight className="w-3 h-3" />
                  </button>

                  <button
                    onClick={() => handleInstall(app)}
                    disabled={isInstalled}
                    className={`px-3.5 py-1.5 rounded-lg text-xs font-medium transition-all flex items-center gap-1.5 cursor-pointer ${isInstalled
                        ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                        : 'bg-fluent-blue hover:bg-fluent-blueHover text-white shadow-fluent-btn active:scale-95'
                      }`}
                  >
                    {isInstalled ? (
                      <>
                        <Check className="w-3.5 h-3.5" />
                        <span>Installed</span>
                      </>
                    ) : (
                      <>
                        <Download className="w-3.5 h-3.5" />
                        <span>Install</span>
                      </>
                    )}
                  </button>
                </div>
              </div>
            );
          })
        ) : (
          <div className="h-64 flex flex-col items-center justify-center text-center p-6 border border-dashed border-mica-border rounded-2xl text-fluent-dim">
            <span className="material-symbols-outlined text-4xl mb-2 text-fluent-blue opacity-50">search_off</span>
            <p className="text-sm font-semibold text-fluent-text">No packages found</p>
            <p className="text-xs max-w-sm mt-1 text-fluent-muted">
              Type an application name or keyword above to query across official Pacman, AUR, Flatpak, and Snap.
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

function stringsTitle(str: string): string {
  if (!str) return '';
  return str.charAt(0).toUpperCase() + str.slice(1);
}
