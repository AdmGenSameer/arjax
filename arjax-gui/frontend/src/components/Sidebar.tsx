import React from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { DistroInfoDTO } from '../types';

interface SidebarProps {
  distroInfo?: DistroInfoDTO;
}

export const Sidebar: React.FC<SidebarProps> = ({ distroInfo }) => {
  const navigate = useNavigate();

  return (
    <aside className="fixed left-0 top-12 bottom-0 w-[230px] bg-mica-base/95 border-r border-white/[0.08] z-40 flex flex-col justify-between py-3 px-2 select-none">
      <div className="space-y-6 overflow-y-auto">
        {/* Main Store Navigation */}
        <div>
          <nav className="space-y-0.5">
            <NavLink
              to="/"
              end
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-md font-medium text-[13px] transition ${
                  isActive
                    ? 'fluent-indicator bg-white/[0.08] text-white'
                    : 'text-fluent-muted hover:text-white hover:bg-white/[0.05]'
                }`
              }
            >
              {({ isActive }) => (
                <>
                  <span className={`material-symbols-outlined text-[19px] ${isActive ? 'text-fluent-blue' : 'text-fluent-dim'}`}>
                    home
                  </span>
                  <span>Home</span>
                </>
              )}
            </NavLink>

            <NavLink
              to="/apps"
              end
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-md font-medium text-[13px] transition group ${
                  isActive
                    ? 'fluent-indicator bg-white/[0.08] text-white'
                    : 'text-fluent-muted hover:text-white hover:bg-white/[0.05]'
                }`
              }
            >
              {({ isActive }) => (
                <>
                  <span className={`material-symbols-outlined text-[19px] transition ${isActive ? 'text-fluent-blue' : 'text-fluent-dim group-hover:text-fluent-blue'}`}>
                    grid_view
                  </span>
                  <span>Apps</span>
                </>
              )}
            </NavLink>

            <NavLink
              to="/gaming"
              end
              className={({ isActive }) =>
                `flex items-center justify-between px-3 py-2 rounded-md font-medium text-[13px] transition group ${
                  isActive
                    ? 'fluent-indicator bg-white/[0.08] text-white'
                    : 'text-fluent-muted hover:text-white hover:bg-white/[0.05]'
                }`
              }
            >
              {({ isActive }) => (
                <>
                  <div className="flex items-center gap-3">
                    <span className={`material-symbols-outlined text-[19px] transition ${isActive ? 'text-fluent-blue' : 'text-fluent-dim group-hover:text-fluent-blue'}`}>
                      sports_esports
                    </span>
                    <span>Gaming</span>
                  </div>
                  <span className="px-1.5 py-0.2 rounded text-[10px] font-medium bg-fluent-blue/20 text-[#60a5fa]">
                    Proton
                  </span>
                </>
              )}
            </NavLink>

            <NavLink
              to="/creativity"
              end
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-md font-medium text-[13px] transition group ${
                  isActive
                    ? 'fluent-indicator bg-white/[0.08] text-white'
                    : 'text-fluent-muted hover:text-white hover:bg-white/[0.05]'
                }`
              }
            >
              {({ isActive }) => (
                <>
                  <span className={`material-symbols-outlined text-[19px] transition ${isActive ? 'text-fluent-blue' : 'text-fluent-dim group-hover:text-fluent-blue'}`}>
                    palette
                  </span>
                  <span>Creativity</span>
                </>
              )}
            </NavLink>

            <NavLink
              to="/entertainment"
              end
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-md font-medium text-[13px] transition group ${
                  isActive
                    ? 'fluent-indicator bg-white/[0.08] text-white'
                    : 'text-fluent-muted hover:text-white hover:bg-white/[0.05]'
                }`
              }
            >
              {({ isActive }) => (
                <>
                  <span className={`material-symbols-outlined text-[19px] transition ${isActive ? 'text-fluent-blue' : 'text-fluent-dim group-hover:text-fluent-blue'}`}>
                    theaters
                  </span>
                  <span>Entertainment</span>
                </>
              )}
            </NavLink>
          </nav>
        </div>

        {/* Library & Updates Section */}
        <div className="pt-2 border-t border-white/[0.06]">
          <p className="px-3 text-[10px] font-semibold uppercase tracking-wider text-fluent-dim mb-1.5">
            My Library
          </p>
          <nav className="space-y-0.5">
            <NavLink
              to="/installed"
              className={({ isActive }) =>
                `flex items-center justify-between px-3 py-2 rounded-md font-medium text-[13px] transition group ${
                  isActive
                    ? 'fluent-indicator bg-white/[0.08] text-white'
                    : 'text-fluent-muted hover:text-white hover:bg-white/[0.05]'
                }`
              }
            >
              <div className="flex items-center gap-3">
                <span className="material-symbols-outlined text-[19px] text-fluent-dim group-hover:text-fluent-blue transition">
                  folder_special
                </span>
                <span>Installed</span>
              </div>
              <span className="text-[11px] font-mono text-fluent-dim">28</span>
            </NavLink>

            <NavLink
              to="/updates"
              className={({ isActive }) =>
                `flex items-center justify-between px-3 py-2 rounded-md font-medium text-[13px] transition group ${
                  isActive
                    ? 'fluent-indicator bg-white/[0.08] text-white'
                    : 'text-fluent-muted hover:text-white hover:bg-white/[0.05]'
                }`
              }
            >
              <div className="flex items-center gap-3">
                <span className="material-symbols-outlined text-[19px] text-fluent-dim group-hover:text-fluent-blue transition">
                  system_update
                </span>
                <span>Update Center</span>
              </div>
              <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
                Hub
              </span>
            </NavLink>

            <NavLink
              to="/downloads"
              className={({ isActive }) =>
                `flex items-center justify-between px-3 py-2 rounded-md font-medium text-[13px] transition group ${
                  isActive
                    ? 'fluent-indicator bg-white/[0.08] text-white'
                    : 'text-fluent-muted hover:text-white hover:bg-white/[0.05]'
                }`
              }
            >
              <div className="flex items-center gap-3">
                <span className="material-symbols-outlined text-[19px] text-fluent-dim group-hover:text-fluent-blue transition">
                  downloading
                </span>
                <span>Downloads</span>
              </div>
              <span className="px-1.5 py-0.5 rounded-full text-[10px] font-bold bg-fluent-blue text-white">
                3
              </span>
            </NavLink>

            <div className="flex items-center gap-3 px-3 py-2 rounded-md text-fluent-muted hover:text-white hover:bg-white/[0.05] font-medium text-[13px] transition group cursor-pointer">
              <span className="material-symbols-outlined text-[19px] text-fluent-dim group-hover:text-fluent-blue transition">
                bookmark_border
              </span>
              <span>Wishlist</span>
            </div>
          </nav>
        </div>
      </div>

      {/* Bottom Navigation Items: Settings & Distro badge */}
      <div className="space-y-1 pt-2 border-t border-white/[0.06]">
        <NavLink
          to="/settings"
          className={({ isActive }) =>
            `flex items-center gap-3 px-3 py-2 rounded-md font-medium text-[13px] transition ${
              isActive
                ? 'fluent-indicator bg-white/[0.08] text-white'
                : 'text-fluent-muted hover:text-white hover:bg-white/[0.05]'
            }`
          }
        >
          <span className="material-symbols-outlined text-[19px] text-fluent-dim">
            tune
          </span>
          <span>Store Settings</span>
        </NavLink>

        <div className="px-3 py-2 rounded-md bg-white/[0.03] border border-white/[0.05] flex items-center justify-between mt-2">
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-[16px] text-fluent-dim">
              desktop_windows
            </span>
            <span className="text-[11.5px] text-fluent-muted">
              {distroInfo?.name || 'Arch'} / Flathub
            </span>
          </div>
          <span className="w-2 h-2 rounded-full bg-[#107c41]" title="Repositories synced"></span>
        </div>
      </div>
    </aside>
  );
};
