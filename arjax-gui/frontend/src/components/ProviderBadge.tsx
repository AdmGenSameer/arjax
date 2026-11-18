import React from 'react';
import { getProviderBadgeStyle } from '../lib/utils';

interface ProviderBadgeProps {
  provider: string;
  size?: 'sm' | 'md';
}

export const ProviderBadge: React.FC<ProviderBadgeProps> = ({ provider, size = 'md' }) => {
  const { bg, label } = getProviderBadgeStyle(provider);
  
  const sizeClasses = size === 'sm' 
    ? 'text-[10px] px-1.5 py-0.5 tracking-wider font-semibold' 
    : 'text-[11px] px-2 py-0.5 tracking-wider font-semibold';

  return (
    <span
      className={`inline-flex items-center justify-center rounded border font-mono uppercase ${sizeClasses} ${bg}`}
    >
      {label}
    </span>
  );
};

