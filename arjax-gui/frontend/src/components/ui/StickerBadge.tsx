import React from 'react';
import { Sticker } from 'stick-it-up';

interface StickerBadgeProps {
  name: 'metal' | 'star-burst' | 'flame-star' | 'sparkles' | 'sparkle-trio' | 'app-badge' | 'devil-cone' | 'cosmic-eye';
  size?: number;
  className?: string;
}

export const StickerBadge: React.FC<StickerBadgeProps> = ({ name, size = 32, className = '' }) => {
  return (
    <div className={`inline-flex items-center justify-center filter drop-shadow-[0_4px_12px_rgba(0,0,0,0.5)] transition-transform hover:scale-110 ${className}`}>
      <Sticker name={name} size={size} peel />
    </div>
  );
};

