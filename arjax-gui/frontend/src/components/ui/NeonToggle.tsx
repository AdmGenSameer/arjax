import React from 'react';

interface NeonToggleProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  description?: string;
  disabled?: boolean;
}

export const NeonToggle: React.FC<NeonToggleProps> = ({
  checked,
  onChange,
  label,
  description,
  disabled = false,
}) => {
  return (
    <label className={`flex items-center justify-between py-2 cursor-pointer select-none ${disabled ? 'opacity-50 pointer-events-none' : ''}`}>
      {(label || description) && (
        <div className="flex flex-col pr-4">
          {label && <span className="text-sm font-medium text-on-surface">{label}</span>}
          {description && <span className="text-xs text-outline">{description}</span>}
        </div>
      )}
      <div
        onClick={() => !disabled && onChange(!checked)}
        className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
          checked ? 'bg-primary-container shadow-[0_0_12px_rgba(122,162,247,0.35)]' : 'bg-surface-container-high'
        }`}
      >
        <span
          className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out ${
            checked ? 'translate-x-5 bg-on-primary-container' : 'translate-x-0 bg-outline'
          }`}
        />
      </div>
    </label>
  );
};

