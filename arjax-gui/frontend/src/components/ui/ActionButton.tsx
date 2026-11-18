import React from 'react';
import { Loader2, Check, Download, Trash2 } from 'lucide-react';

interface ActionButtonProps {
  status?: 'idle' | 'installing' | 'installed' | 'removing';
  onClick?: (e: React.MouseEvent) => void;
  variant?: 'primary' | 'secondary' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  label?: string;
  className?: string;
}

export const ActionButton: React.FC<ActionButtonProps> = ({
  status = 'idle',
  onClick,
  variant = 'primary',
  size = 'md',
  label,
  className = '',
}) => {
  const isInstalling = status === 'installing';
  const isInstalled = status === 'installed';

  const sizeClasses = {
    sm: 'px-2.5 py-1 text-xs gap-1.5',
    md: 'px-3.5 py-1.5 text-xs font-semibold gap-2',
    lg: 'px-4 py-2 text-sm font-semibold gap-2.5',
  }[size];

  const variantClasses = {
    primary:
      'bg-primary-container text-on-primary-container hover:bg-primary-container/90 active:scale-[0.98] shadow-sm shadow-primary-container/20 border border-primary-container/30',
    secondary:
      'bg-surface-container hover:bg-surface-container-high text-on-surface border border-outline-variant/40 active:scale-[0.98]',
    danger:
      'bg-error-container/20 text-error hover:bg-error-container/40 border border-error/30 active:scale-[0.98]',
  }[variant];

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={isInstalling}
      className={`relative inline-flex items-center justify-center rounded-lg transition-all font-medium select-none overflow-hidden group ${sizeClasses} ${
        isInstalled
          ? 'bg-surface-container-high text-on-surface-variant border border-outline-variant/30 cursor-default'
          : variantClasses
      } ${className}`}
    >
      {/* Background shimmer on hover */}
      <span className="absolute inset-0 w-full h-full bg-gradient-to-r from-transparent via-white/10 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-700 ease-in-out pointer-events-none" />

      {isInstalling ? (
        <>
          <Loader2 className="w-3.5 h-3.5 animate-spin text-current" />
          <span>Installing...</span>
        </>
      ) : isInstalled ? (
        <>
          <Check className="w-3.5 h-3.5 text-emerald-400" />
          <span>{label || 'Installed'}</span>
        </>
      ) : (
        <>
          <Download className="w-3.5 h-3.5 transition-transform group-hover:-translate-y-0.5" />
          <span>{label || 'Install'}</span>
        </>
      )}
    </button>
  );
};

