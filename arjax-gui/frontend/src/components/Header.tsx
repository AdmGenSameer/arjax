import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';

export const Header: React.FC = () => {
  const navigate = useNavigate();
  const [searchVal, setSearchVal] = useState('');

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchVal.trim()) {
      navigate(`/search?q=${encodeURIComponent(searchVal.trim())}`);
    }
  };

  return (
    <header className="fixed top-0 left-0 right-0 h-12 bg-mica-base/90 fluent-acrylic border-b border-white/[0.08] z-50 flex items-center justify-between px-3 sm:px-4 wails-drag select-none">
      {/* Left: Window controls & Store branding */}
      <div className="flex items-center gap-3.5 min-w-[220px]">
        {/* Window Controls (Fluent / Linux standard) */}
        <div className="flex items-center gap-2 pr-3.5 border-r border-white/[0.08] wails-no-drag">
          <span className="w-3 h-3 rounded-full bg-white/20 hover:bg-[#e81123] transition cursor-pointer" title="Close"></span>
          <span className="w-3 h-3 rounded-full bg-white/20 hover:bg-white/40 transition cursor-pointer" title="Minimize"></span>
          <span className="w-3 h-3 rounded-full bg-white/20 hover:bg-[#107c41] transition cursor-pointer" title="Maximize"></span>
        </div>

        {/* Store Brand Icon & Name */}
        <div 
          onClick={() => navigate('/')} 
          className="flex items-center gap-2 cursor-pointer wails-no-drag group"
        >
          <div className="w-6 h-6 rounded-md bg-fluent-blue flex items-center justify-center shadow-sm group-hover:scale-105 transition">
            <span className="material-symbols-outlined text-white text-[16px]">shopping_bag</span>
          </div>
          <span className="font-semibold text-[13px] tracking-normal text-white">
            Arjax Store
          </span>
        </div>
      </div>

      {/* Center: Authentic Windows 11 Store Wide Search Bar */}
      <div className="flex-1 max-w-xl mx-4 wails-no-drag">
        <form onSubmit={handleSearchSubmit} className="relative flex items-center">
          <span className="material-symbols-outlined absolute left-3.5 text-fluent-muted text-[17px] pointer-events-none">
            search
          </span>
          <input
            type="text"
            value={searchVal}
            onChange={(e) => setSearchVal(e.target.value)}
            placeholder="Search apps, games, movies and tools..."
            className="w-full pl-10 pr-24 py-1.5 bg-black/30 hover:bg-black/40 focus:bg-black/60 text-[12.5px] text-white placeholder-fluent-dim rounded-md border border-white/[0.08] focus:border-fluent-blue focus:outline-none focus:ring-1 focus:ring-fluent-blue transition"
          />
          <div className="absolute right-2.5 flex items-center gap-1.5">
            <button
              type="button"
              onClick={() => navigate('/search')}
              className="text-fluent-dim hover:text-white p-0.5 transition"
              title="Filter search"
            >
              <span className="material-symbols-outlined text-[15px]">tune</span>
            </button>
            <span className="text-[10px] font-mono text-fluent-dim bg-white/[0.06] px-1.5 py-0.5 rounded border border-white/[0.08]">
              Ctrl+E
            </span>
          </div>
        </form>
      </div>

      {/* Right: Account Profile & Distro Status */}
      <div className="flex items-center gap-2 min-w-[200px] justify-end wails-no-drag">
        <button
          type="button"
          onClick={() => navigate('/settings')}
          className="flex items-center gap-2 p-1.5 hover:bg-white/[0.06] rounded-md transition text-fluent-muted hover:text-white"
          title="Account & Settings"
        >
          <div className="w-6 h-6 rounded-full bg-gradient-to-tr from-fluent-blue to-teal-500 flex items-center justify-center text-[11px] font-bold text-white shadow-sm">
            A
          </div>
          <span className="text-[12px] font-medium hidden md:inline text-white">Alex Morgan</span>
          <span className="material-symbols-outlined text-[14px]">expand_more</span>
        </button>
      </div>
    </header>
  );
};
