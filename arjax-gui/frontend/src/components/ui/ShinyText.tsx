import React from 'react';

interface ShinyTextProps {
  text: string;
  disabled?: boolean;
  speed?: number;
  className?: string;
}

export const ShinyText: React.FC<ShinyTextProps> = ({
  text,
  disabled = false,
  speed = 5,
  className = '',
}) => {
  const animationDuration = `${speed}s`;

  return (
    <span
      className={`inline-block bg-clip-text text-transparent ${
        disabled
          ? 'text-on-surface'
          : 'bg-gradient-to-r from-on-surface via-primary to-on-surface bg-[length:250%_100%] animate-shine'
      } ${className}`}
      style={{
        animationDuration,
      }}
    >
      {text}
    </span>
  );
};

