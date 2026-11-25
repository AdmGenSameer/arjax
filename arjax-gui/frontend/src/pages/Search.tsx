import React, { useState, useEffect, useRef } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { Filter, ArrowRight, Download, Check, Sparkles, Layers } from 'lucide-react';
import { AppIcon } from '../components/AppIcon';
import {
  UnifiedApplicationDTO,
  PackageOfferDTO,
  ProgressiveSearchBatchDTO,
  ProgressiveSearchDoneDTO,
  SearchStartedEventDTO,
  SearchProviderStatusDTO,
} from '../types';

export const SearchPage: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryParam = searchParams.get('q') || '';
  const [query, setQuery] = useState(queryParam);
  const [selectedFilter, setSelectedFilter] = useState('ALL');
  const [unifiedResults, setUnifiedResults] = useState<UnifiedApplicationDTO[]>([]);
  const [providerStatuses, setProviderStatuses] = useState<Record<string, SearchProviderStatusDTO>>({});
  const [loading, setLoading] = useState(false);
  const [installedMap, setInstalledMap] = useState<Record<string, boolean>>({});
  const activeSearchIdRef = useRef<string>('');
  const lastSequenceRef = useRef<number>(0);
  const resultsMapRef = useRef<Map<string, UnifiedApplicationDTO>>(new Map());

  const filters = ['ALL', 'PACMAN', 'FLATPAK', 'AUR', 'SNAP', 'GITHUB'];

  useEffect(() => {
    setQuery(queryParam);
    if (queryParam.trim()) {
      performSearch(queryParam.trim());
    }
  }, [queryParam]);

  useEffect(() => {
    return () => {
      const goApp = (window as any).go?.main?.App;
      if (goApp && goApp.CancelProgressiveSearch && activeSearchIdRef.current) {
        goApp.CancelProgressiveSearch(activeSearchIdRef.current);
      }
    };
  }, []);

  const performSearch = async (q: string) => {
    const goApp = (window as any).go?.main?.App;
    const runtime = (window as any).runtime;

    // Cancel prior search before starting a new query
    if (goApp && goApp.CancelProgressiveSearch && activeSearchIdRef.current) {
      goApp.CancelProgressiveSearch(activeSearchIdRef.current);
    }

    setLoading(true);
    const searchId = 'srch_' + Date.now() + '_' + Math.random().toString(36).slice(2, 7);
    activeSearchIdRef.current = searchId;
    lastSequenceRef.current = 0;
    resultsMapRef.current = new Map();
    setProviderStatuses({});
    setUnifiedResults([]);

    const shouldAcceptEvent = (ev: { searchId?: string; sequence?: number } | null | undefined) => {
      if (!ev || ev.searchId !== activeSearchIdRef.current) return false;
      if (typeof ev.sequence === 'number' && ev.sequence <= lastSequenceRef.current) {
        return false;
      }
      if (typeof ev.sequence === 'number') {
        lastSequenceRef.current = ev.sequence;
      }
      return true;
    };

    if (goApp) {
      try {
        if (goApp.SearchUnifiedProgressive && runtime && runtime.EventsOn) {
          runtime.EventsOn('search:started', (ev: SearchStartedEventDTO) => {
            if (!shouldAcceptEvent(ev)) return;
            const initial: Record<string, SearchProviderStatusDTO> = {};
            (ev.providers || []).forEach((p) => {
              initial[p] = {
                searchId: ev.searchId,
                sequence: ev.sequence,
                provider: p,
                status: 'started',
                durationMs: 0,
              };
            });
            setProviderStatuses(initial);
          });

          runtime.EventsOn('search:provider-status', (ev: SearchProviderStatusDTO) => {
            if (!shouldAcceptEvent(ev)) return;
            setProviderStatuses((prev) => ({
              ...prev,
              [ev.provider]: ev,
            }));
          });

          runtime.EventsOn('search:provider-error', (ev: SearchProviderStatusDTO) => {
            if (!shouldAcceptEvent(ev)) return;
            setProviderStatuses((prev) => ({
              ...prev,
              [ev.provider]: { ...ev, status: 'failed' },
            }));
          });

          runtime.EventsOn('search:batch', (batch: ProgressiveSearchBatchDTO) => {
            if (!shouldAcceptEvent(batch)) return;

            // Apply incremental deltas when supplied; otherwise fall back to full list
            if ((batch.addedApps && batch.addedApps.length > 0) || (batch.updatedApps && batch.updatedApps.length > 0)) {
              const map = resultsMapRef.current;
              (batch.addedApps || []).forEach((app) => map.set(app.appId, app));
              (batch.updatedApps || []).forEach((app) => map.set(app.appId, app));
              setUnifiedResults(Array.from(map.values()));
            } else if (Array.isArray(batch.apps)) {
              const map = new Map<string, UnifiedApplicationDTO>();
              batch.apps.forEach((app) => map.set(app.appId, app));
              resultsMapRef.current = map;
              setUnifiedResults(batch.apps);
            }
            setLoading(false);
          });

          runtime.EventsOn('search:cancelled', (ev: ProgressiveSearchDoneDTO) => {
            if (!shouldAcceptEvent(ev)) return;
            setLoading(false);
          });

          runtime.EventsOn('search:done', (done: ProgressiveSearchDoneDTO) => {
            if (!shouldAcceptEvent(done)) return;
            setLoading(false);
          });

          goApp.SearchUnifiedProgressive(q, searchId);
          return;
        }

        if (goApp.SearchUnifiedSync) {
          const apps = await goApp.SearchUnifiedSync(q);
          if (Array.isArray(apps) && apps.length > 0) {
            setUnifiedResults(apps);
            setLoading(false);
            return;
          }
        }
      } catch (e) {
        console.warn('Progressive search error:', e);
      }
    }

    // Offline preview fallback preserving complete multi-source schema
    setTimeout(() => {
      if (activeSearchIdRef.current !== searchId) return;
      const qLower = q.toLowerCase();
      const mockUnified: UnifiedApplicationDTO[] = [
        {
          appId: `org.mozilla.${qLower}`,
          name: stringsTitle(q),
          summary: `Fast, modern, and reliable ${q} built cleanly for Linux.`,
          description: 'Comprehensive desktop suite with native acceleration and sandbox options.',
          developer: 'Mozilla & Community Contributors',
          homepage: `https://${qLower}.org`,
          license: 'MPL-2.0 / Open Source',
          categories: ['Web', 'Productivity'],
          screenshots: [],
          releases: [],
          aliases: [qLower, `${qLower}-bin`],
          offers: [
            {
              id: `pacman:extra:${qLower}`,
              appId: `org.mozilla.${qLower}`,
              packageName: qLower,
              provider: 'pacman',
              repository: 'extra',
              source: 'extra',
              architecture: 'x86_64',
              version: '124.0.1-1',
              downloadSize: '78 MB',
              installedSize: '210 MB',
              installScope: 'system',
              permissions: ['system-native', 'hardware-direct'],
              isInstalled: false,
              isEligible: true,
              trustLevel: 'official',
              signatureStatus: 'verified-keyring',
              score: 95,
              recommendationBadge: 'Recommended',
              recommendationReason: 'Official Arch extra repository package with verified keyring signatures.',
              commandPreview: `sudo pacman -S ${qLower}`,
            },
            {
              id: `flatpak:flathub:org.mozilla.${qLower}:branch=stable`,
              appId: `org.mozilla.${qLower}`,
              packageName: `org.mozilla.${qLower}`,
              provider: 'flatpak',
              repository: 'flathub',
              source: 'flathub',
              architecture: 'x86_64',
              branch: 'stable',
              version: '124.0.1',
              downloadSize: '85 MB',
              installedSize: '240 MB',
              installScope: 'sandboxed',
              permissions: ['network', 'pulseaudio', 'wayland'],
              isInstalled: false,
              isEligible: true,
              trustLevel: 'verified',
              signatureStatus: 'ostree-signed',
              score: 88,
              recommendationBadge: 'Sandboxed',
              recommendationReason: 'Flathub container with isolated user permissions.',
              commandPreview: `flatpak install flathub org.mozilla.${qLower}`,
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
    }, 200);
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
    if (goApp && goApp.InstallOffer) {
      try {
        const tx = await goApp.InstallOffer(offer.id);
        if (tx && tx.state === 'completed') {
          setInstalledMap((prev) => ({ ...prev, [offer.id]: true }));
        }
      } catch (e) {
        console.error(e);
      }
    } else if (goApp && goApp.InstallPackage) {
      try {
        await goApp.InstallPackage(offer.packageName, offer.provider);
        setInstalledMap((prev) => ({ ...prev, [offer.id]: true }));
      } catch (e) {
        console.error(e);
      }
    } else {
      setTimeout(() => {
        setInstalledMap((prev) => ({ ...prev, [offer.id]: true }));
      }, 1000);
    }
  };

  const filteredApps = unifiedResults
    .filter((app) => {
      if (selectedFilter === 'ALL') return true;
      return app.offers.some((o) => o.provider.toUpperCase().includes(selectedFilter));
    })
    .slice(0, 50);

  const statusEntries = Object.values(providerStatuses);

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

        {/* Filter Pills & Live Provider Stream Status */}
        <div className="flex flex-wrap items-center justify-between gap-3">
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

          {statusEntries.length > 0 && (
            <div className="flex items-center gap-1.5 flex-wrap">
              {statusEntries.map((st) => (
                <span
                  key={st.provider}
                  title={st.error || `${st.provider}: ${st.status}`}
                  className={`text-[10px] font-mono px-2 py-0.5 rounded border ${st.status === 'completed'
                    ? 'bg-emerald-500/10 text-emerald-300 border-emerald-500/20'
                    : st.status === 'failed'
                      ? 'bg-red-500/10 text-red-300 border-red-500/20'
                      : 'bg-white/5 text-fluent-muted border-white/10'
                    }`}
                >
                  {st.provider}
                  {st.status === 'completed' ? ` (${st.resultCount ?? 0})` : st.status === 'failed' ? ' !' : '...'}
                </span>
              ))}
            </div>
          )}
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
            const isOfferInstalled = (defaultOffer && (defaultOffer.isInstalled || installedMap[defaultOffer.id])) || app.isInstalled;

            return (
              <div
                key={app.appId}
                className="fluent-card p-4 rounded-xl flex items-center justify-between gap-4 group cursor-pointer border border-white/5 hover:border-fluent-blue/30 transition-all bg-surface-container/60 hover:bg-surface-container"
              >
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

                      <span className="text-[10px] font-mono text-fluent-dim bg-white/5 px-2 py-0.5 rounded border border-white/5">
                        {app.appId}
                      </span>

                      {defaultOffer?.recommendationBadge && (
                        <span className="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-full font-medium bg-fluent-blue/15 text-fluent-blue border border-fluent-blue/30">
                          <Sparkles className="w-2.5 h-2.5" />
                          {defaultOffer.recommendationBadge}
                        </span>
                      )}

                      {defaultOffer?.trustLevel && (
                        <span className="text-[10px] px-1.5 py-0.5 rounded border border-white/10 bg-white/5 text-fluent-muted font-mono">
                          trust:{defaultOffer.trustLevel}
                        </span>
                      )}

                      <span className="text-[11px] text-fluent-dim font-mono">
                        {defaultOffer?.version || 'Latest'}
                      </span>
                    </div>

                    <p className="text-xs text-fluent-muted line-clamp-1">
                      {app.summary || app.description || 'Verified Linux desktop application.'}
                    </p>

                    <div className="flex items-center gap-1.5 pt-0.5 text-[11px] text-fluent-dim font-mono">
                      <Layers className="w-3 h-3 text-fluent-blue" />
                      <span>{app.offers.length} {app.offers.length === 1 ? 'Offer:' : 'Offers:'}</span>
                      <div className="flex items-center gap-1 flex-wrap">
                        {app.offers.map((offer) => (
                          <span
                            key={offer.id}
                            className={`px-1.5 py-0.2 rounded uppercase text-[10px] font-semibold border ${offer.id === app.defaultOfferId
                              ? 'bg-primary/20 text-primary border-primary/30'
                              : 'bg-white/5 text-fluent-muted border-white/5'
                              }`}
                          >
                            {offer.provider} ({offer.repository || offer.source})
                          </span>
                        ))}
                      </div>
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-2 flex-shrink-0">
                  <button
                    type="button"
                    onClick={() => navigate(`/package/${encodeURIComponent(app.appId)}`)}
                    className="px-3 py-1.5 rounded-lg bg-white/[0.04] hover:bg-white/[0.08] text-fluent-muted hover:text-fluent-text text-xs font-medium border border-mica-border transition-colors flex items-center gap-1.5 cursor-pointer"
                    title="View all sources, trust, permissions & screenshots"
                  >
                    <span>Details ({app.offers.length})</span>
                    <ArrowRight className="w-3 h-3" />
                  </button>

                  <button
                    onClick={() => handleInstall(app)}
                    disabled={isOfferInstalled}
                    className={`px-3.5 py-1.5 rounded-lg text-xs font-medium transition-all flex items-center gap-1.5 cursor-pointer ${isOfferInstalled
                      ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                      : 'bg-fluent-blue hover:bg-fluent-blueHover text-white shadow-fluent-btn active:scale-95'
                      }`}
                  >
                    {isOfferInstalled ? (
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
