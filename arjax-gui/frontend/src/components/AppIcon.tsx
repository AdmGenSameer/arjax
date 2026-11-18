import React, { useState, useEffect } from 'react';
import { GetAppIcon, ResolvePackageIcon, ResolveCatalogIcon, ResolveInstalledIcon } from '../../wailsjs/go/main/App';

// Global cache to avoid redundant IPC calls across card renders and navigation
const iconCache = new Map<string, { url: string; dataUri?: string }>();
const pendingResolutions = new Map<string, Promise<{ url: string; dataUri?: string } | null>>();

export interface AppIconProps {
  appId?: string;
  pkgName?: string;
  provider?: string;
  mode?: 'catalog' | 'installed';
  iconUrl?: string;
  fallbackIcon?: string;
  fallbackBg?: string;
  size?: number;
  className?: string;
  imgClassName?: string;
  alt?: string;
}

export const AppIcon: React.FC<AppIconProps> = ({
  appId = '',
  pkgName = '',
  provider = '',
  mode = 'catalog',
  iconUrl: initialIconUrl,
  fallbackIcon = 'inventory_2',
  fallbackBg = 'bg-transparent',
  size = 128,
  className = 'w-10 h-10',
  imgClassName = 'w-full h-full object-contain',
  alt = 'Application Icon',
}) => {
  const normKey = (appId || pkgName || '').trim().toLowerCase();
  const cacheKey = `${normKey}:${mode}:${size}`;

  // 1. Initial source: Check cache, then initial prop
  const [resolvedSrc, setResolvedSrc] = useState<string | null>(() => {
    if (initialIconUrl && initialIconUrl.startsWith('data:')) {
      return initialIconUrl;
    }
    if (normKey && iconCache.has(cacheKey)) {
      const cached = iconCache.get(cacheKey)!;
      return cached.url || cached.dataUri || null;
    }
    return initialIconUrl || null;
  });

  const [hasError, setHasError] = useState(false);
  const [isLoaded, setIsLoaded] = useState(false);

  useEffect(() => {
    // If we already have a valid data URI, use it directly
    if (initialIconUrl && initialIconUrl.startsWith('data:')) {
      setResolvedSrc(initialIconUrl);
      if (normKey) iconCache.set(cacheKey, { url: initialIconUrl, dataUri: initialIconUrl });
      return;
    }

    if (normKey && iconCache.has(cacheKey)) {
      const cached = iconCache.get(cacheKey)!;
      setResolvedSrc(cached.url || cached.dataUri || null);
      return;
    }

    if (!normKey) return;

    // Check if another component is already resolving this icon
    if (!pendingResolutions.has(cacheKey)) {
      const resolvePromise = (async () => {
        try {
          let res: any = null;
          if (mode === 'installed' && ResolveInstalledIcon) {
            res = await ResolveInstalledIcon(normKey, pkgName || normKey, provider || '', size);
          } else if (ResolveCatalogIcon) {
            res = await ResolveCatalogIcon(normKey, pkgName || normKey, provider || '', size);
          } else if (ResolvePackageIcon) {
            res = await ResolvePackageIcon(normKey, pkgName || normKey, provider || '', size);
          } else if (GetAppIcon) {
            res = await GetAppIcon(normKey, size);
          }

          if (res) {
            const entry = {
              url: res.iconUrl || res.dataUri || '',
              dataUri: res.dataUri || '',
            };
            if (entry.url || entry.dataUri) {
              iconCache.set(cacheKey, entry);
              return entry;
            }
          }
        } catch (err) {
          console.warn(`[IconResolver] Resolution failed for ${normKey}:`, err);
        }
        return null;
      })();

      pendingResolutions.set(cacheKey, resolvePromise);
    }

    const activePromise = pendingResolutions.get(cacheKey);
    if (activePromise) {
      let isMounted = true;
      activePromise.then((entry) => {
        if (isMounted && entry) {
          setResolvedSrc(entry.url || entry.dataUri || null);
          setHasError(false);
        }
      });
      return () => {
        isMounted = false;
      };
    }
  }, [normKey, cacheKey, initialIconUrl, pkgName, provider, mode, size]);

  const handleImageError = () => {
    // If local asset route failed, try the base64 dataUri fallback
    const cached = iconCache.get(cacheKey);
    if (cached?.dataUri && resolvedSrc !== cached.dataUri) {
      setResolvedSrc(cached.dataUri);
      setHasError(false);
      return;
    }

    // Attempt direct IPC refresh
    if (normKey && GetAppIcon) {
      GetAppIcon(normKey, size)
        .then((res: any) => {
          if (res?.dataUri) {
            iconCache.set(cacheKey, { url: res.dataUri, dataUri: res.dataUri });
            setResolvedSrc(res.dataUri);
            setHasError(false);
            return;
          }
          setHasError(true);
        })
        .catch(() => setHasError(true));
      return;
    }

    setHasError(true);
  };

  // If we have a resolved image source and no error
  if (resolvedSrc && !hasError) {
    return (
      <div className={`relative flex items-center justify-center overflow-hidden shrink-0 ${className}`}>
        <img
          src={resolvedSrc}
          alt={alt}
          className={`${imgClassName} transition-opacity duration-200 ${isLoaded ? 'opacity-100' : 'opacity-80'}`}
          onLoad={() => setIsLoaded(true)}
          onError={handleImageError}
          loading="lazy"
        />
      </div>
    );
  }

  // Graceful Fallback: Clean, polished transparent container with Material Symbol glyph
  return (
    <div
      className={`relative flex items-center justify-center shrink-0 rounded-lg text-white shadow-none overflow-hidden ${fallbackBg} ${className}`}
      title={alt}
    >
      <span className="material-symbols-outlined text-[60%] select-none opacity-85">
        {fallbackIcon || 'inventory_2'}
      </span>
    </div>
  );
};
