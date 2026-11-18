import React from 'react';
import { PackageOfferDTO } from '../types';

interface SourcePickerProps {
  offers: PackageOfferDTO[];
  selectedOfferId: string;
  onSelectOffer: (offerId: string) => void;
  onInstall: (packageName: string, provider: string) => void;
  onRemove?: (packageName: string, provider: string) => void;
  installing: boolean;
  appDisplayName: string;
}

export const SourcePicker: React.FC<SourcePickerProps> = ({
  offers,
  selectedOfferId,
  onSelectOffer,
  onInstall,
  onRemove,
  installing,
  appDisplayName,
}) => {
  if (!offers || offers.length === 0) {
    return null;
  }

  const selectedOffer = offers.find((o) => o.id === selectedOfferId) || offers[0];
  const anyInstalledOffer = offers.find((o) => o.isInstalled);
  const isSelectedInstalled = selectedOffer?.isInstalled;
  const isAnotherInstalled = anyInstalledOffer && anyInstalledOffer.id !== selectedOffer?.id;

  const getProviderIcon = (provider: string) => {
    switch (provider.toLowerCase()) {
      case 'flatpak':
        return 'view_in_ar';
      case 'snap':
        return 'layers';
      case 'aur':
        return 'build';
      default:
        return 'terminal';
    }
  };

  const getBadgeStyle = (badge: string) => {
    switch (badge.toLowerCase()) {
      case 'recommended':
        return 'bg-fluent-blue/20 text-fluent-blue border-fluent-blue/30';
      case 'installed':
        return 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30';
      case 'sandboxed':
        return 'bg-purple-500/20 text-purple-400 border-purple-500/30';
      case 'community (bin)':
      case 'community':
        return 'bg-amber-500/20 text-amber-400 border-amber-500/30';
      default:
        return 'bg-white/10 text-white/70 border-white/10';
    }
  };

  return (
    <div className="rounded-xl bg-surface-container p-5 space-y-4 border border-white/5 shadow-fluent-elevation">
      {/* Header and Source Count */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-primary text-[20px]">hub</span>
          <h3 className="font-title text-title text-on-surface font-semibold">
            Distribution Channels &amp; Origins
          </h3>
        </div>
        <span className="text-xs font-mono text-on-surface-variant bg-surface-container-high px-2.5 py-1 rounded-md">
          {offers.length} {offers.length === 1 ? 'Source Available' : 'Sources Available'}
        </span>
      </div>

      {/* Offer Pills */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2.5">
        {offers.map((offer) => {
          const isSelected = offer.id === selectedOffer?.id;
          return (
            <button
              key={offer.id}
              onClick={() => onSelectOffer(offer.id)}
              type="button"
              className={`p-3 rounded-lg text-left transition-all border flex flex-col justify-between gap-2 cursor-pointer ${
                isSelected
                  ? 'bg-primary-container/20 border-primary shadow-sm ring-1 ring-primary'
                  : 'bg-surface-container-low hover:bg-surface-container-high border-white/5 text-on-surface-variant'
              }`}
            >
              <div className="flex items-center justify-between gap-1 w-full">
                <div className="flex items-center gap-2 min-w-0">
                  <span className={`material-symbols-outlined text-[18px] ${isSelected ? 'text-primary' : 'text-on-surface-variant'}`}>
                    {getProviderIcon(offer.provider)}
                  </span>
                  <span className="font-semibold text-sm text-on-surface capitalize truncate">
                    {offer.provider}
                  </span>
                  <span className="text-[11px] font-mono text-on-surface-variant uppercase">
                    [{offer.source || 'repo'}]
                  </span>
                </div>
                <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${getBadgeStyle(offer.recommendationBadge)}`}>
                  {offer.recommendationBadge}
                </span>
              </div>

              <div className="flex items-center justify-between text-[11.5px] font-mono text-on-surface-variant">
                <span>{offer.version || 'Latest'}</span>
                <span>{offer.downloadSize}</span>
              </div>
            </button>
          );
        })}
      </div>

      {/* Selected Offer Detail Card */}
      {selectedOffer && (
        <div className="rounded-lg bg-surface-container-low p-4 space-y-3 border border-white/5">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-white/5 pb-2.5">
            <div className="space-y-0.5">
              <div className="flex items-center gap-2">
                <span className="font-medium text-sm text-on-surface">
                  Source: <span className="text-primary font-semibold capitalize">{selectedOffer.provider}</span> ({selectedOffer.source})
                </span>
                <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${getBadgeStyle(selectedOffer.recommendationBadge)}`}>
                  {selectedOffer.recommendationBadge}
                </span>
              </div>
              <p className="text-xs text-on-surface-variant">
                Package Name: <code className="text-on-surface font-mono text-[11px]">{selectedOffer.packageName}</code>
              </p>
            </div>

            <div className="text-right">
              <span className="text-xs text-on-surface-variant block">Isolation Scope</span>
              <span className="text-xs font-semibold capitalize text-on-surface">
                {selectedOffer.installScope === 'sandboxed' ? '🛡️ Sandboxed Container' : '⚙️ Host System Integration'}
              </span>
            </div>
          </div>

          {/* Recommendation Justification */}
          {selectedOffer.recommendationReason && (
            <div className="flex items-start gap-2 bg-primary/10 text-primary-fixed-dim p-2.5 rounded-md text-xs">
              <span className="material-symbols-outlined text-[16px] text-primary shrink-0 mt-0.5">info</span>
              <p className="leading-relaxed text-on-surface">
                <strong className="text-primary">{selectedOffer.recommendationBadge}:</strong> {selectedOffer.recommendationReason}
              </p>
            </div>
          )}

          {/* Sandbox Permissions preview */}
          {selectedOffer.permissions && selectedOffer.permissions.length > 0 && (
            <div className="space-y-1.5 pt-1">
              <span className="text-[11px] text-on-surface-variant font-medium block">
                Security &amp; Permissions:
              </span>
              <div className="flex flex-wrap gap-1.5">
                {selectedOffer.permissions.map((perm, idx) => (
                  <span
                    key={idx}
                    className="px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant text-[10.5px] font-mono border border-white/5"
                  >
                    {perm}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Dual-Install Safety Reassurance */}
          {isAnotherInstalled && !isSelectedInstalled && (
            <div className="flex items-center gap-2 p-2.5 rounded-md bg-amber-500/10 border border-amber-500/20 text-amber-300 text-xs">
              <span className="material-symbols-outlined text-[16px] shrink-0">notification_important</span>
              <span>
                <strong>{anyInstalledOffer?.provider.toUpperCase()}</strong> is currently installed on your system.
                Installing via <strong>{selectedOffer.provider.toUpperCase()}</strong> will keep both versions isolated without replacing system files.
              </span>
            </div>
          )}

          {/* Active Action Controls */}
          <div className="flex items-center justify-between gap-3 pt-2">
            <div className="text-xs text-on-surface-variant font-mono">
              Terminal: <code className="text-tertiary">{selectedOffer.installCmd}</code>
            </div>

            <div className="flex items-center gap-2">
              {isSelectedInstalled ? (
                <>
                  <div className="px-3.5 py-1.5 rounded-lg bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 text-xs font-semibold flex items-center gap-1.5">
                    <span className="material-symbols-outlined text-[15px]">check_circle</span>
                    <span>Installed ({selectedOffer.installedVersion || 'Active'})</span>
                  </div>
                  {onRemove && (
                    <button
                      onClick={() => onRemove(selectedOffer.packageName, selectedOffer.provider)}
                      className="px-3 py-1.5 rounded-lg bg-red-500/10 hover:bg-red-500/20 text-red-400 border border-red-500/20 text-xs font-medium transition cursor-pointer"
                      type="button"
                    >
                      Uninstall
                    </button>
                  )}
                </>
              ) : (
                <button
                  onClick={() => onInstall(selectedOffer.packageName, selectedOffer.provider)}
                  disabled={installing || !selectedOffer.isEligible}
                  className={`px-4 py-2 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition active:scale-95 cursor-pointer ${
                    !selectedOffer.isEligible
                      ? 'bg-surface-container-high text-on-surface-variant cursor-not-allowed opacity-60'
                      : installing
                      ? 'bg-primary-container/80 text-on-primary-container opacity-85'
                      : 'bg-primary-container hover:bg-primary-container/90 text-on-primary-container shadow-md'
                  }`}
                  type="button"
                >
                  {installing ? (
                    <>
                      <span className="material-symbols-outlined text-[15px] animate-spin">refresh</span>
                      <span>Installing...</span>
                    </>
                  ) : !selectedOffer.isEligible ? (
                    <>
                      <span className="material-symbols-outlined text-[15px]">block</span>
                      <span>{selectedOffer.ineligibleReason || 'Unavailable'}</span>
                    </>
                  ) : (
                    <>
                      <span className="material-symbols-outlined text-[15px]">download</span>
                      <span>Install via {selectedOffer.provider.toUpperCase()}</span>
                    </>
                  )}
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

