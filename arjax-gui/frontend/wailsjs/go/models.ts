export namespace icons {
	
	export class IconProvenance {
	    source: string;
	    repository: string;
	    originalUrl: string;
	    localOriginPath: string;
	    contentHash: string;
	    // Go type: time
	    fetchedAt: any;
	    // Go type: time
	    validatedAt: any;
	    validationStatus: string;
	
	    static createFrom(source: any = {}) {
	        return new IconProvenance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.repository = source["repository"];
	        this.originalUrl = source["originalUrl"];
	        this.localOriginPath = source["localOriginPath"];
	        this.contentHash = source["contentHash"];
	        this.fetchedAt = this.convertValues(source["fetchedAt"], null);
	        this.validatedAt = this.convertValues(source["validatedAt"], null);
	        this.validationStatus = source["validationStatus"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IconResponseDTO {
	    appId: string;
	    name?: string;
	    iconUrl: string;
	    dataUri: string;
	    iconStatus: string;
	    format: string;
	    width: number;
	    height: number;
	    contentHash: string;
	    source: string;
	    repository: string;
	    localPath?: string;
	    provenance: IconProvenance;
	
	    static createFrom(source: any = {}) {
	        return new IconResponseDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appId = source["appId"];
	        this.name = source["name"];
	        this.iconUrl = source["iconUrl"];
	        this.dataUri = source["dataUri"];
	        this.iconStatus = source["iconStatus"];
	        this.format = source["format"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.contentHash = source["contentHash"];
	        this.source = source["source"];
	        this.repository = source["repository"];
	        this.localPath = source["localPath"];
	        this.provenance = this.convertValues(source["provenance"], IconProvenance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class CacheCleanResultDTO {
	    success: boolean;
	    reclaimedMB: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new CacheCleanResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.reclaimedMB = source["reclaimedMB"];
	        this.message = source["message"];
	    }
	}
	export class DistroInfoDTO {
	    id: string;
	    name: string;
	    packageManager: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new DistroInfoDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.packageManager = source["packageManager"];
	        this.label = source["label"];
	    }
	}
	export class EngineAppDetailsDTO {
	    id: string;
	    flathubId: string;
	    name: string;
	    headline: string;
	    publisher: string;
	    version: string;
	    rating: number;
	    reviews: string;
	    size: string;
	    provider: string;
	    license: string;
	    website: string;
	    iconName: string;
	    iconUrl: string;
	    iconBg: string;
	    badge: string;
	    description: string;
	    screenshots: string[];
	    releases: string[];
	
	    static createFrom(source: any = {}) {
	        return new EngineAppDetailsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.flathubId = source["flathubId"];
	        this.name = source["name"];
	        this.headline = source["headline"];
	        this.publisher = source["publisher"];
	        this.version = source["version"];
	        this.rating = source["rating"];
	        this.reviews = source["reviews"];
	        this.size = source["size"];
	        this.provider = source["provider"];
	        this.license = source["license"];
	        this.website = source["website"];
	        this.iconName = source["iconName"];
	        this.iconUrl = source["iconUrl"];
	        this.iconBg = source["iconBg"];
	        this.badge = source["badge"];
	        this.description = source["description"];
	        this.screenshots = source["screenshots"];
	        this.releases = source["releases"];
	    }
	}
	export class EnginePackageDTO {
	    name: string;
	    version: string;
	    description: string;
	    provider: string;
	    source: string;
	    repo?: string;
	    size?: string;
	
	    static createFrom(source: any = {}) {
	        return new EnginePackageDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.provider = source["provider"];
	        this.source = source["source"];
	        this.repo = source["repo"];
	        this.size = source["size"];
	    }
	}
	export class InstallProgressDTO {
	    packageName: string;
	    provider: string;
	    progress: number;
	    stage: string;
	    logs: string[];
	    completed: boolean;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallProgressDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.packageName = source["packageName"];
	        this.provider = source["provider"];
	        this.progress = source["progress"];
	        this.stage = source["stage"];
	        this.logs = source["logs"];
	        this.completed = source["completed"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class InstalledPackageDTO {
	    id: string;
	    name: string;
	    displayName: string;
	    version: string;
	    provider: string;
	    format: string;
	    size: string;
	    sizeMB: number;
	    hasUpdate: boolean;
	    updateVersion?: string;
	    category: string;
	    icon: string;
	    lastActivity: string;
	    isNative: boolean;
	    isSharedCore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InstalledPackageDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.version = source["version"];
	        this.provider = source["provider"];
	        this.format = source["format"];
	        this.size = source["size"];
	        this.sizeMB = source["sizeMB"];
	        this.hasUpdate = source["hasUpdate"];
	        this.updateVersion = source["updateVersion"];
	        this.category = source["category"];
	        this.icon = source["icon"];
	        this.lastActivity = source["lastActivity"];
	        this.isNative = source["isNative"];
	        this.isSharedCore = source["isSharedCore"];
	    }
	}
	export class UpdateItemDTO {
	    name: string;
	    oldVersion: string;
	    newVersion: string;
	    provider: string;
	    size: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateItemDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.oldVersion = source["oldVersion"];
	        this.newVersion = source["newVersion"];
	        this.provider = source["provider"];
	        this.size = source["size"];
	    }
	}

}

export namespace models {
	
	export class PackageInfo {
	    name: string;
	    version: string;
	    description: string;
	    size?: string;
	    dependencies?: string[];
	    files?: string[];
	    last_updated?: string;
	    repo?: string;
	    provider?: string;
	
	    static createFrom(source: any = {}) {
	        return new PackageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.size = source["size"];
	        this.dependencies = source["dependencies"];
	        this.files = source["files"];
	        this.last_updated = source["last_updated"];
	        this.repo = source["repo"];
	        this.provider = source["provider"];
	    }
	}
	export class Result {
	    command?: string;
	    package: string;
	    provider: string;
	    success: boolean;
	    // Go type: time
	    start_time: any;
	    // Go type: time
	    end_time: any;
	    duration_ms: number;
	    output: string;
	    error?: string;
	    install_cmd?: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.package = source["package"];
	        this.provider = source["provider"];
	        this.success = source["success"];
	        this.start_time = this.convertValues(source["start_time"], null);
	        this.end_time = this.convertValues(source["end_time"], null);
	        this.duration_ms = source["duration_ms"];
	        this.output = source["output"];
	        this.error = source["error"];
	        this.install_cmd = source["install_cmd"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

