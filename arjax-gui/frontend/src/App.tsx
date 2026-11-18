import React, { useState, useEffect } from 'react';
import { Routes, Route } from 'react-router-dom';
import { Header } from './components/Header';
import { Sidebar } from './components/Sidebar';
import { Home } from './pages/Home';
import { SearchPage } from './pages/Search';
import { CategoryPage } from './pages/CategoryPage';
import { PackagePage } from './pages/Package';
import { InstalledPage } from './pages/Installed';
import { DownloadsPage } from './pages/Downloads';
import { SettingsPage } from './pages/Settings';
import { DistroInfoDTO } from './types';

export const App: React.FC = () => {
  const [distro, setDistro] = useState<DistroInfoDTO>({
    id: 'arch',
    name: 'Arch Linux',
    packageManager: 'pacman',
    label: 'PACMAN (Arch Linux)',
  });

  useEffect(() => {
    try {
      import('../wailsjs/go/main/App').then(({ GetDistroInfo }) => {
        GetDistroInfo().then((res: any) => {
          if (res && res.label) setDistro(res);
        }).catch(() => {});
      }).catch(() => {});
    } catch {
      // Dev mode fallback
    }
  }, []);

  return (
    <div className="bg-mica-bg font-sans text-fluent-text antialiased select-none h-screen w-screen overflow-hidden flex flex-col">
      {/* 1. Window Titlebar / Mica Chrome Header (h-12) */}
      <Header />

      {/* 2. Left Sidebar Navigation (230px, top-12 bottom-0) */}
      <Sidebar distroInfo={distro} />

      {/* 3. Main Store Content Viewport */}
      <div className="pl-[230px] pt-12 h-full flex-1 overflow-y-auto">
        <main className="relative min-h-full w-full bg-mica-bg">
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/apps" element={<CategoryPage type="apps" />} />
            <Route path="/gaming" element={<CategoryPage type="gaming" />} />
            <Route path="/creativity" element={<CategoryPage type="creativity" />} />
            <Route path="/entertainment" element={<CategoryPage type="entertainment" />} />
            <Route path="/search" element={<SearchPage />} />
            <Route path="/package/:name" element={<PackagePage />} />
            <Route path="/installed" element={<InstalledPage />} />
            <Route path="/downloads" element={<DownloadsPage />} />
            <Route path="/downloads-and-updates" element={<DownloadsPage />} />
            <Route path="/updates" element={<DownloadsPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Routes>
        </main>
      </div>
    </div>
  );
};
