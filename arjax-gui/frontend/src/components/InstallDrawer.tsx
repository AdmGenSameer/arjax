import React from 'react';

interface InstallDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  packageName?: string;
  provider?: string;
  progress?: number;
  stage?: number;
  stageText?: string;
  logs?: string[];
  isCompleted?: boolean;
  isSuccess?: boolean;
  error?: string;
}

export const InstallDrawer: React.FC<InstallDrawerProps> = ({
  isOpen,
  onClose,
  packageName = 'brave-bin',
  provider = 'AUR',
  progress = 68,
  stage = 3,
  stageText = 'Building from source...',
  logs = [],
  isCompleted = false,
  isSuccess = true,
  error,
}) => {
  if (!isOpen) return null;

  return (
    <div 
      id="installDrawer"
      className="sticky bottom-1 w-full bg-surface-container rounded-t-xl shadow-2xl z-40 p-space-md space-y-space-sm border border-outline-variant/50 transition-all duration-200"
    >
      {/* Drawer Header Row */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
        <div className="flex items-center gap-space-sm flex-wrap">
          {/* Pulse Indicator */}
          <div className="relative flex items-center justify-center w-3 h-3">
            {isCompleted ? (
              <span className={`inline-flex rounded-full h-2.5 w-2.5 ${isSuccess ? 'bg-emerald-400' : 'bg-error'}`} />
            ) : (
              <>
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary-container opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-primary"></span>
              </>
            )}
          </div>
          <span className="font-headline-md text-headline-md text-on-surface font-semibold">
            {isCompleted
              ? isSuccess
                ? `Installed ${packageName}`
                : `Installation Failed: ${packageName}`
              : `Installing ${packageName}`}
          </span>
          <span className="px-1.5 py-0.5 rounded font-code-sm text-[10px] font-bold uppercase bg-tertiary-container/20 text-tertiary border border-tertiary-container/30">
            {provider}
          </span>
          <span className="font-code-sm text-code-sm text-outline">
            {isCompleted ? (isSuccess ? 'Finished successfully' : 'Halted with error') : '— active'}
          </span>
        </div>
        <div className="flex items-center gap-space-sm font-code-sm text-code-sm text-on-surface-variant">
          <span className="material-symbols-outlined text-[15px] text-primary">downloading</span>
          <span>{isCompleted ? '100%' : `${progress}%`}</span>
          <span className="text-outline">•</span>
          <span className="text-primary font-medium">{isCompleted ? 'Complete' : stageText}</span>
        </div>
      </div>

      {/* Animated Progress Bar */}
      <div className="w-full bg-surface-container-lowest h-1.5 rounded-full overflow-hidden">
        <div 
          className={`h-full rounded-full transition-all duration-300 ${
            isCompleted
              ? isSuccess
                ? 'bg-emerald-400'
                : 'bg-error'
              : 'bg-primary-container'
          }`}
          style={{ width: `${progress}%` }}
        />
      </div>

      {/* 4-Stage Checklist */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-space-xs pt-1">
        {/* Stage 1 */}
        <div className="flex items-center gap-1.5 font-label-sm text-label-sm text-secondary">
          <span className="material-symbols-outlined text-[16px]">check_circle</span>
          <span className="truncate font-medium">Provider resolved → {provider}</span>
        </div>
        {/* Stage 2 */}
        <div className="flex items-center gap-1.5 font-label-sm text-label-sm text-secondary">
          <span className="material-symbols-outlined text-[16px]">check_circle</span>
          <span className="truncate font-medium">Dependencies verified</span>
        </div>
        {/* Stage 3 */}
        <div className={`flex items-center gap-1.5 font-label-sm text-label-sm ${
          progress >= 70 ? 'text-secondary' : 'text-primary'
        }`}>
          <span className={`material-symbols-outlined text-[16px] ${progress < 70 ? 'animate-spin' : ''}`}>
            {progress >= 70 ? 'check_circle' : 'refresh'}
          </span>
          <span className="truncate font-semibold">{stageText}</span>
        </div>
        {/* Stage 4 */}
        <div className={`flex items-center gap-1.5 font-label-sm text-label-sm ${
          isCompleted && isSuccess ? 'text-secondary font-medium' : 'text-outline'
        }`}>
          <span className="material-symbols-outlined text-[16px]">
            {isCompleted && isSuccess ? 'check_circle' : 'radio_button_unchecked'}
          </span>
          <span className="truncate">Desktop integration</span>
        </div>
      </div>

      {/* Terminal Log Box */}
      <div className="bg-surface-container-lowest rounded-md p-2.5 font-code-sm text-code-sm text-on-surface-variant leading-relaxed select-text overflow-x-auto space-y-0.5 border border-outline-variant/30 max-h-36 overflow-y-auto">
        {logs.length > 0 ? (
          logs.map((log, idx) => (
            <div key={idx} className={log.includes('[ERROR]') ? 'text-error font-medium' : log.includes('[SUCCESS]') ? 'text-emerald-400 font-medium' : 'text-outline'}>
              {log}
            </div>
          ))
        ) : (
          <>
            <div className="text-outline">[INIT] Resolving package metadata for {packageName} ({provider})</div>
            <div className="text-secondary">[CHECKSUM] Validating integrity with sha512sums... passed</div>
            <div className="text-on-surface font-medium">
              [EXTRACT] Preparing files... [<span className="text-primary">=================&gt;-------</span>] {progress}%
            </div>
          </>
        )}
        {error && <div className="text-error font-semibold">[ERROR] {error}</div>}
      </div>

      {/* Drawer Action Bar */}
      <div className="flex items-center justify-between pt-1">
        <div className="font-code-sm text-code-sm text-outline">
          Transaction PID #4812 • Isolated sandbox active
        </div>
        <div className="flex items-center gap-space-sm">
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-1 rounded bg-surface-container-high hover:bg-surface-container-highest text-on-surface font-label-sm text-label-sm transition-colors border border-outline-variant/30"
          >
            {isCompleted ? 'Close Drawer' : 'Run in Background'}
          </button>
        </div>
      </div>
    </div>
  );
};
