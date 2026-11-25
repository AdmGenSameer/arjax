import React from 'react';
import { PackageOfferDTO } from '../types';

interface SourcePickerProps {
  offers: PackageOfferDTO[];
  selectedOfferId: string;
  onSelectOffer: (offerId: string) => void;
  onInstallOffer: (offer: PackageOfferDTO) => void;
  onRemoveOffer?: (offer: PackageOfferDTO) => void;
  installing: boolean;
  appDisplayName: string;
}

export const SourcePicker: React.FC<SourcePickerProps> = ({
  offers,
  selectedOfferId,
  onSelectOffer,
  onInstallOffer,
  onRemoveOffer,
  installing,
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
      case 'official':
        return 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30';
      case 'sandboxed':
        return 'bg-purple-500/20 text-purple-400 border-purple-500/30';
      case 'community (bin)':
      case 'community (source)':
      case 'community':
        return 'bg-amber-500/20 text-amber-400 border-amber-500/30';
      default:
        return 'bg-white/10 text-white/70 border-white/10';
    }
  };

  const getTrustBadgeStyle = (level?: string) => {
    switch ((level || '').toLowerCase()) {
      case 'official':
        return 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30';
      case 'verified':
        return 'bg-sky-500/15 text-sky-300 border-sky-500/30';
      case 'community':
        return 'bg-amber-500/15 text-amber-300 border-amber-500/30';
      case 'third-party':
        return 'bg-orange-500/15 text-orange-300 border-orange-500/30';
      default:
        return 'bg-white/10 text-white/60 border-white/10';
    }
  };

  const formatTrustLabel = (level?: string) => {
    switch ((level || '').toLowerCase()) {
      case 'official':
        return 'Official Signed Repo';
      case 'verified':
        return 'Verified Publisher';
      case 'community':
        return 'Community Maintained';
      case 'third-party':
        return 'Third-Party Release';
      default:
        return 'Unclassified Trust';
    }
  };

  const trustLevel = selectedOffer?.trust?.trustLevel || selectedOffer?.trustLevel || 'unknown';

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
          {offers.length} {offers.length === 1 ? 'Offer Available' : 'Offers Available'}
        </span>
      </div>

      {/* Offer Pills */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2.5">
        {offers.map((offer) => {
          const isSelected = offer.id === selectedOffer?.id;
          const repoName = offer.repository || offer.source || 'repo';
          const offerTrust = offer.trust?.trustLevel || offer.trustLevel;

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
                    [{repoName}]
                  </span>
                </div>
                <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium border shrink-0 ${getBadgeStyle(offer.recommendationBadge)}`}>
                  {offer.recommendationBadge}
                </span>
              </div>

              <div className="flex items-center justify-between text-[11px] font-mono text-on-surface-variant">
                <span>
                  {offer.version || 'Latest'}
                  {offer.branch ? ` (${offer.branch})` : offer.channel ? ` (${offer.channel})` : ''}
                </span>
                <span>{offer.architecture || 'x86_64'} · {offer.downloadSize}</span>
              </div>

              {offerTrust && (
                <div className="flex items-center justify-between pt-0.5 border-t border-white/5 text-[10px]">
                  <span className={`px-1.5 py-0.2 rounded border font-medium ${getTrustBadgeStyle(offerTrust)}`}>
                    {formatTrustLabel(offerTrust)}
                  </span>
                  <span className="text-on-surface-variant font-mono">
                    {offer.installScope === 'sandboxed' ? 'Sandboxed' : offer.installScope === 'user' ? 'User Scope' : 'System Scope'}
                  </span>
                </div>
              )}
            </button>
          );
        })}
      </div>

      {/* Selected Offer Detail Card separating Recommendation, Trust, and Isolation */}
      {selectedOffer && (
        <div className="rounded-lg bg-surface-container-low p-4 space-y-3 border border-white/5">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-white/5 pb-2.5">
            <div className="space-y-0.5">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="font-medium text-sm text-on-surface">
                  Origin: <span className="text-primary font-semibold capitalize">{selectedOffer.provider}</span> ({selectedOffer.repository || selectedOffer.source})
                </span>
                <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${getBadgeStyle(selectedOffer.recommendationBadge)}`}>
                  {selectedOffer.recommendationBadge}
                </span>
                <span className={`text-[10px] px-2 py-0.5 rounded-full font-medium border ${getTrustBadgeStyle(trustLevel)}`}>
                  {formatTrustLabel(trustLevel)}
                </span>
              </div>
              <p className="text-xs text-on-surface-variant font-mono">
                OfferID: <code className="text-on-surface text-[11px]">{selectedOffer.id}</code>
                {selectedOffer.architecture ? ` · Arch: ${selectedOffer.architecture}` : ''}
                {selectedOffer.branch ? ` · Branch: ${selectedOffer.branch}` : ''}
                {selectedOffer.channel ? ` · Channel: ${selectedOffer.channel}` : ''}
              </p>
            </div>

            <div className="text-right">
              <span className="text-xs text-on-surface-variant block">Isolation Scope</span>
              <span className="text-xs font-semibold capitalize text-on-surface">
                {selectedOffer.installScope === 'sandboxed'
                  ? '🛡️ Sandboxed Container'
                  : selectedOffer.installScope === 'user'
                  ? '👤 User Build Scope'
                  : '⚙️ Host System Integration'}
              </span>
            </div>
          </div>

          {/* 3-Column Separation: Recommendation vs Trust vs Dependencies */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-2.5">
            {selectedOffer.recommendationReason && (
              <div className="flex items-start gap-2 bg-primary/10 p-2.5 rounded-md text-xs border border-primary/15">
                <span className="material-symbols-outlined text-[16px] text-primary shrink-0 mt-0.5">auto_awesome</span>
                <p className="leading-relaxed text-on-surface">
                  <strong className="text-primary block">Recommendation:</strong>
                  {selectedOffer.recommendationReason}
                </p>
              </div>
            )}

            <div className="flex items-start gap-2 bg-white/[0.03] p-2.5 rounded-md text-xs border border-white/5">
              <span className="material-symbols-outlined text-[16px] text-emerald-400 shrink-0 mt-0.5">
                {selectedOffer.trust?.signatureChecked ? 'verified_user' : 'warning'}
              </span>
              <div className="leading-relaxed text-on-surface">
                <strong className="text-on-surface block">
                  Source Trust ({selectedOffer.trust?.signatureStatus || selectedOffer.signatureStatus || 'standard'}):
                </strong>
                <span className="text-on-surface-variant block">
                  {selectedOffer.trust?.summary || formatTrustLabel(trustLevel)}
                  {selectedOffer.trust?.buildRecipe ? ` • Recipe: ${selectedOffer.trust.buildRecipe}` : ''}
                </span>
                {selectedOffer.trust?.provenance && (
                  <span className="text-[11px] font-mono text-on-surface-variant block mt-1">
                    Origin: {selectedOffer.trust.provenance.origin} · Key: {selectedOffer.trust.provenance.signingKey}
                  </span>
                )}
              </div>
            </div>
          </div>

          {/* Sandbox Permissions preview */}
          {selectedOffer.permissions && selectedOffer.permissions.length > 0 && (
            <div className="space-y-1.5 pt-1">
              <span className="text-[11px] text-on-surface-variant font-medium block">
                Isolation &amp; Runtime Permissions:
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
                <strong>{anyInstalledOffer?.provider.toUpperCase()}</strong> is currently installed on your host.
                Installing via <strong>{selectedOffer.provider.toUpperCase()}</strong> keeps both isolated with zero silent migration.
              </span>
            </div>
          )}

          {/* Active Action Controls — authority held by backend OfferID */}
          <div className="flex flex-wrap items-center justify-between gap-3 pt-2">
            <div className="text-xs text-on-surface-variant font-mono">
              {selectedOffer.commandPreview && (
                <>
                  CLI Equivalent: <code className="text-tertiary select-all">{selectedOffer.commandPreview}</code>
                </>
              )}
            </div>

            <div className="flex items-center gap-2">
              {isSelectedInstalled ? (
                <>
                  <div className="px-3.5 py-1.5 rounded-lg bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 text-xs font-semibold flex items-center gap-1.5">
                    <span className="material-symbols-outlined text-[15px]">check_circle</span>
                    <span>Installed ({selectedOffer.installedVersion || 'Active'})</span>
                  </div>
                  {onRemoveOffer && (
                    <button
                      onClick={() => onRemoveOffer(selectedOffer)}
                      className="px-3 py-1.5 rounded-lg bg-red-500/10 hover:bg-red-500/20 text-red-400 border border-red-500/20 text-xs font-medium transition cursor-pointer"
                      type="button"
                    >
                      Uninstall
                    </button>
                  )}
                </>
              ) : (
                <button
                  onClick={() => onInstallOffer(selectedOffer)}
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
                      <span>Processing Transaction...</span>
                    </>
                  ) : !selectedOffer.isEligible ? (
                    <>
                      <span className="material-symbols-outlined text-[15px]">block</span>
                      <span>{selectedOffer.ineligibleReason || 'Unavailable'}</span>
                    </>
                  ) : (
                    <>
                      <span className="material-symbols-outlined text-[15px]">download</span>
                      <span>Install {selectedOffer.provider.toUpperCase()} Offer</span>
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
