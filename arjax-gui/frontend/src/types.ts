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

export interface PackageProvenanceDTO {
  origin: string;
  publisher: string;
  signingKey: string;
  artifactDigest: string;
  verification: string;
  verifiedAt: string;
}

export interface SourceTrustDTO {
  trustLevel: 'official' | 'verified' | 'community' | 'third-party' | 'unknown' | string;
  officialRepo: boolean;
  signatureChecked: boolean;
  signatureStatus: string;
  maintainer: string;
  buildRecipe: string;
  verified: boolean;
  summary: string;
  provenance?: PackageProvenanceDTO;
}

export interface PackageOfferDTO {
  id: string;
  appId: string;
  packageName: string;
  provider: string;
  repository: string;
  source: string;
  architecture: string;
  version: string;
  installedVersion?: string;
  branch?: string;
  channel?: string;
  downloadSize: string;
  installedSize: string;
  installScope: 'sandboxed' | 'system' | 'user' | string;
  scope?: 'sandboxed' | 'system' | 'user' | string;
  permissions: string[];
  dependencyCount?: number;
  dependencies?: string[];
  isInstalled: boolean;
  isEligible: boolean;
  ineligibleReason?: string;
  trustLevel?: string;
  signatureStatus?: string;
  trust?: SourceTrustDTO;
  score: number;
  recommendationBadge: string;
  recommendationReason: string;
  commandPreview?: string;
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
  identityConfidence?: string;
  offers: PackageOfferDTO[];
  defaultOfferId: string;
  isInstalled: boolean;
  installedOfferId?: string;
  rating: number;
  reviews: string;
  badge: string;
}

export interface SearchStartedEventDTO {
  searchId: string;
  sequence: number;
  status: string;
  query: string;
  providers: string[];
}

export interface SearchProviderStatusDTO {
  searchId: string;
  sequence: number;
  provider: string;
  status: 'started' | 'running' | 'completed' | 'failed' | 'cancelled' | string;
  durationMs: number;
  resultCount?: number;
  error?: string;
}

export interface ProgressiveSearchBatchDTO {
  searchId: string;
  sequence: number;
  provider: string;
  status: string;
  addedApps?: UnifiedApplicationDTO[];
  updatedApps?: UnifiedApplicationDTO[];
  apps?: UnifiedApplicationDTO[];
  isFinal: boolean;
  durationMs: number;
}

export interface ProgressiveSearchDoneDTO {
  searchId: string;
  sequence: number;
  status: string;
  totalApps: number;
  durationMs: number;
  cancelled?: boolean;
}

export interface TransactionDTO {
  id: string;
  offerId: string;
  appId: string;
  packageName: string;
  provider: string;
  repository: string;
  operation: 'install' | 'remove' | 'update' | string;
  state:
  | 'queued'
  | 'resolving_deps'
  | 'awaiting_auth'
  | 'downloading'
  | 'verifying'
  | 'installing'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | string;
  progress: number;
  message: string;
  dependencies?: string[];
  logs: string[];
  canCancelSafely: boolean;
  cancelRequested: boolean;
  exitCode?: number;
  errorCode?: string;
  errorMessage?: string;
  recoveryStatus?: string;
  startedAt?: string;
  completedAt?: string;
}

export interface ProviderCapabilityDTO {
  provider: string;
  displayName: string;
  search: string;
  install: string;
  remove: string;
  update: string;
  installedDetection: string;
  dependencyPreview: string;
  lockDetection: boolean;
  officialDistro: boolean;
  availableOnHost: boolean;
  statusNote: string;
}

export interface ProviderStatusDTO {
  provider: string;
  displayName: string;
  supported: boolean;
  installed: boolean;
  available: boolean;
  healthy: boolean;
  statusMessage: string;
  capabilities: ProviderCapabilityDTO;
  lastHealthCheck: string;
}

export interface AvailableUpdateDTO {
  offerId: string;
  appId: string;
  packageName: string;
  displayName: string;
  provider: string;
  repository: string;
  installScope?: string;
  branch?: string;
  installedVersion: string;
  availableVersion: string;
  downloadSize: string;
  isSecurity: boolean;
  partialUpgradeSafe?: boolean;
  upgradeGroupPolicy?: string;
  trustLevel: string;
  summary: string;
}

export interface UpdateSummaryDTO {
  totalCount: number;
  providerCount: number;
  securityCount: number;
  requiresFullSysUpgrade?: boolean;
  byProvider: Record<string, number>;
  updates: AvailableUpdateDTO[];
  providerErrors?: Record<string, string>;
  notes?: string[];
  checkedAt: string;
}

export interface UpdateItemOutcomeDTO {
  offerId: string;
  packageName: string;
  provider: string;
  installScope?: string;
  success: boolean;
  skippedUpToDate?: boolean;
  errorCode?: string;
  transactionId?: string;
  errorMessage?: string;
}

export interface TransactionPreviewDTO {
  previewId: string;
  requestedOfferIds: string[];
  effectiveOfferIds: string[];
  packagesToUpgrade: AvailableUpdateDTO[];
  packagesToInstall: string[];
  packagesToRemove: string[];
  estimatedDownloadSize: string;
  requiresOfficialFirst: boolean;
  includesFullSysUpgrade: boolean;
  includesAurBuild: boolean;
  warnings: string[];
  providerErrors?: Record<string, string>;
  requiresExplicitApproval: boolean;
  createdAt: string;
  expiresAt: string;
}

export interface UpdateBatchResultDTO {
  totalRequested: number;
  succeeded: number;
  failed: number;
  results: UpdateItemOutcomeDTO[];
  note: string;
}
