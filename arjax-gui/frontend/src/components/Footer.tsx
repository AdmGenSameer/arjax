import React from 'react';

interface FooterProps {
  distroLabel?: string;
}

export const Footer: React.FC<FooterProps> = ({ distroLabel }) => {
  return (
    <footer className="fixed bottom-0 left-0 right-0 h-8 bg-surface-container-low border-t border-outline-variant z-50 flex items-center justify-between px-space-md select-none text-xs">
      <div className="flex items-center gap-space-lg">
        <div className="flex items-center gap-1.5">
          <span className="material-symbols-outlined text-[14px] text-primary">sync</span>
          <span className="font-code-sm text-code-sm text-on-surface-variant">Mirrors: 18ms</span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="material-symbols-outlined text-[14px] text-tertiary">pending</span>
          <span className="font-code-sm text-code-sm text-on-surface-variant">Pending tasks: 0</span>
        </div>
      </div>
      <div className="flex items-center gap-space-sm">
        <span className="font-code-sm text-code-sm text-outline">
          {distroLabel || 'System v6.9.1-arch1-1'}
        </span>
      </div>
    </footer>
  );
};

