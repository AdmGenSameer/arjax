export interface PackageDTO {
  name: string;
  version: string;
  description: string;
  provider: 'pacman' | 'aur' | 'flatpak' | 'snap' | 'github' | string;
  source?: string;
  repo?: string;
  size?: string;
  iconUrl?: string;
}

export interface PackageInfoDTO {
  name: string;
  version: string;
  description: string;
  size: string;
  dependencies: string[];
  repo: string;
  provider: string;
  last_updated?: string;
}

export interface InstallProgressDTO {
  packageName: string;
  provider: string;
  progress: number;
  stage: string;
  logs: string[];
  completed: boolean;
  success: boolean;
  error?: string;
}

export interface InstalledPackageDTO {
  id: string;
  name: string;
  displayName: string;
  version: string;
  provider: string;
  format: string;
  size: string;
  sizeMB: number;
  hasUpdate?: boolean;
  updateVersion?: string;
  category: 'apps' | 'games' | 'runtimes' | string;
  icon: string;
  lastActivity: string;
  isNative?: boolean;
  isSharedCore?: boolean;
}

export interface UpdateItemDTO {
  name: string;
  oldVersion: string;
  newVersion: string;
  provider: string;
  size: string;
}

export interface CacheCleanResultDTO {
  success: boolean;
  reclaimedMB: number;
  message: string;
}

export interface DistroInfoDTO {
  id: string;
  name: string;
  packageManager: string;
  label: string;
}

export interface CategoryDTO {
  name: string;
  icon: string;
  query: string;
  description?: string;
}

export interface PackageOfferDTO {
  id: string;
  appId: string;
  packageName: string;
  provider: string;
  source: string;
  version: string;
  downloadSize: string;
  installedSize: string;
  installScope: 'sandboxed' | 'system' | 'user' | string;
  permissions: string[];
  isInstalled: boolean;
  installedVersion?: string;
  isEligible: boolean;
  ineligibleReason?: string;
  score: number;
  recommendationBadge: string;
  recommendationReason: string;
  installCmd?: string;
}

export interface UnifiedApplicationDTO {
  appId: string;
  name: string;
  summary: string;
  description: string;
  developer: string;
  homepage: string;
  license: string;
  iconUrl?: string;
  categories: string[];
  screenshots: string[];
  releases: string[];
  aliases: string[];
  offers: PackageOfferDTO[];
  defaultOfferId: string;
  isInstalled: boolean;
  installedOfferId?: string;
  rating: number;
  reviews: string;
  badge: string;
}

export interface ProgressiveSearchBatchDTO {
  searchId: string;
  apps: UnifiedApplicationDTO[];
  provider: string;
  isFinal: boolean;
  durationMs: number;
}

export interface ProgressiveSearchDoneDTO {
  searchId: string;
  totalApps: number;
  durationMs: number;
}
