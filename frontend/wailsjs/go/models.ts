export namespace main {
	
	export class Announcement {
	    id: string;
	    title: string;
	    body: string;
	    level: string;
	    start: number;
	    end: number;
	
	    static createFrom(source: any = {}) {
	        return new Announcement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.body = source["body"];
	        this.level = source["level"];
	        this.start = source["start"];
	        this.end = source["end"];
	    }
	}
	export class Rect {
	    x: number;
	    y: number;
	    w: number;
	    h: number;
	
	    static createFrom(source: any = {}) {
	        return new Rect(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.w = source["w"];
	        this.h = source["h"];
	    }
	}
	export class ArrangeLayout {
	    mode: string;
	    screen: Rect;
	    cols: number;
	    rows: number;
	    cells: Rect[];
	
	    static createFrom(source: any = {}) {
	        return new ArrangeLayout(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.screen = this.convertValues(source["screen"], Rect);
	        this.cols = source["cols"];
	        this.rows = source["rows"];
	        this.cells = this.convertValues(source["cells"], Rect);
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
	export class ArrangeResult {
	    mode: string;
	    screen: Rect;
	    cols: number;
	    rows: number;
	    cells: Rect[];
	    arranged: number;
	    total: number;
	    failed: string[];
	
	    static createFrom(source: any = {}) {
	        return new ArrangeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.screen = this.convertValues(source["screen"], Rect);
	        this.cols = source["cols"];
	        this.rows = source["rows"];
	        this.cells = this.convertValues(source["cells"], Rect);
	        this.arranged = source["arranged"];
	        this.total = source["total"];
	        this.failed = source["failed"];
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
	export class AuditEntry {
	    id: string;
	    time: number;
	    source: string;
	    action: string;
	    target: string;
	    name: string;
	    ok: boolean;
	    error?: string;
	    detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.time = source["time"];
	        this.source = source["source"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.name = source["name"];
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.detail = source["detail"];
	    }
	}
	export class AuditFilter {
	    category: string;
	    action: string;
	    source: string;
	    target: string;
	    query: string;
	    onlyFail: boolean;
	    since: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new AuditFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.action = source["action"];
	        this.source = source["source"];
	        this.target = source["target"];
	        this.query = source["query"];
	        this.onlyFail = source["onlyFail"];
	        this.since = source["since"];
	        this.limit = source["limit"];
	    }
	}
	export class AuditTargetCount {
	    id: string;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new AuditTargetCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class AuditResult {
	    items: AuditEntry[];
	    total: number;
	    all: number;
	    failed: number;
	    categories: Record<string, number>;
	    sources: Record<string, number>;
	    targets: AuditTargetCount[];
	
	    static createFrom(source: any = {}) {
	        return new AuditResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], AuditEntry);
	        this.total = source["total"];
	        this.all = source["all"];
	        this.failed = source["failed"];
	        this.categories = source["categories"];
	        this.sources = source["sources"];
	        this.targets = this.convertValues(source["targets"], AuditTargetCount);
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
	
	export class BulkResult {
	    ok: number;
	    failed: number;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new BulkResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.failed = source["failed"];
	        this.errors = source["errors"];
	    }
	}
	export class CatalogSite {
	    id: string;
	    name: string;
	    url: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new CatalogSite(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.icon = source["icon"];
	    }
	}
	export class CatalogGroup {
	    id: string;
	    name: string;
	    sites: CatalogSite[];
	
	    static createFrom(source: any = {}) {
	        return new CatalogGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sites = this.convertValues(source["sites"], CatalogSite);
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
	export class Catalog {
	    revision: number;
	    groups: CatalogGroup[];
	    announcements: Announcement[];
	
	    static createFrom(source: any = {}) {
	        return new Catalog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.groups = this.convertValues(source["groups"], CatalogGroup);
	        this.announcements = this.convertValues(source["announcements"], Announcement);
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
	
	
	export class CatalogState {
	    catalog: Catalog;
	    fetchedAt: number;
	    offline: boolean;
	    error?: string;
	    read: string[];
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new CatalogState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.catalog = this.convertValues(source["catalog"], Catalog);
	        this.fetchedAt = source["fetchedAt"];
	        this.offline = source["offline"];
	        this.error = source["error"];
	        this.read = source["read"];
	        this.source = source["source"];
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
	export class ClockReport {
	    status: string;
	    skewSecs: number;
	    source: string;
	    localTime: number;
	    serverTime: number;
	    error: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new ClockReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.skewSecs = source["skewSecs"];
	        this.source = source["source"];
	        this.localTime = source["localTime"];
	        this.serverTime = source["serverTime"];
	        this.error = source["error"];
	        this.detail = source["detail"];
	    }
	}
	export class CloudDevice {
	    id: string;
	    name: string;
	    os: string;
	    firstSeen: number;
	    lastSeen: number;
	    approved: boolean;
	    revoked: boolean;
	    status: string;
	    current: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CloudDevice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.os = source["os"];
	        this.firstSeen = source["firstSeen"];
	        this.lastSeen = source["lastSeen"];
	        this.approved = source["approved"];
	        this.revoked = source["revoked"];
	        this.status = source["status"];
	        this.current = source["current"];
	    }
	}
	export class CloudOnlyItem {
	    id: string;
	    name: string;
	    version: number;
	    updatedAt: number;
	    deviceName: string;
	    size: number;
	    lockedBy: string;
	    localDeleted: boolean;
	    deletedVersion: number;
	    deletedAt: number;
	    cloudNewer: boolean;
	    pendingDelete: boolean;
	    pendingError?: string;
	
	    static createFrom(source: any = {}) {
	        return new CloudOnlyItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.updatedAt = source["updatedAt"];
	        this.deviceName = source["deviceName"];
	        this.size = source["size"];
	        this.lockedBy = source["lockedBy"];
	        this.localDeleted = source["localDeleted"];
	        this.deletedVersion = source["deletedVersion"];
	        this.deletedAt = source["deletedAt"];
	        this.cloudNewer = source["cloudNewer"];
	        this.pendingDelete = source["pendingDelete"];
	        this.pendingError = source["pendingError"];
	    }
	}
	export class CloudOnlyList {
	    items: CloudOnlyItem[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new CloudOnlyList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], CloudOnlyItem);
	        this.error = source["error"];
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
	export class CloudVersion {
	    version: number;
	    size: number;
	    updatedAt: number;
	    deviceName: string;
	    current: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CloudVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.size = source["size"];
	        this.updatedAt = source["updatedAt"];
	        this.deviceName = source["deviceName"];
	        this.current = source["current"];
	    }
	}
	export class CookieImportResult {
	    count: number;
	    skipped: number;
	    mode: string;
	    staged: string;
	
	    static createFrom(source: any = {}) {
	        return new CookieImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.skipped = source["skipped"];
	        this.mode = source["mode"];
	        this.staged = source["staged"];
	    }
	}
	export class CookieInfo {
	    running: boolean;
	    snapshotAt: number;
	    snapshot: number;
	    staged: number;
	
	    static createFrom(source: any = {}) {
	        return new CookieInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.snapshotAt = source["snapshotAt"];
	        this.snapshot = source["snapshot"];
	        this.staged = source["staged"];
	    }
	}
	export class DefaultBrowserStatus {
	    supported: boolean;
	    registered: boolean;
	    isSystemDefault: boolean;
	    profileId: string;
	    profileName: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DefaultBrowserStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supported = source["supported"];
	        this.registered = source["registered"];
	        this.isSystemDefault = source["isSystemDefault"];
	        this.profileId = source["profileId"];
	        this.profileName = source["profileName"];
	        this.error = source["error"];
	    }
	}
	export class DeleteResult {
	    id: string;
	    name: string;
	    local: boolean;
	    cloud?: string;
	    error?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.local = source["local"];
	        this.cloud = source["cloud"];
	        this.error = source["error"];
	        this.message = source["message"];
	    }
	}
	export class DeviceList {
	    supported: boolean;
	    self: string;
	    selfStatus: string;
	    requireApproval: boolean;
	    devices: CloudDevice[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supported = source["supported"];
	        this.self = source["self"];
	        this.selfStatus = source["selfStatus"];
	        this.requireApproval = source["requireApproval"];
	        this.devices = this.convertValues(source["devices"], CloudDevice);
	        this.error = source["error"];
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
	export class DevicePreset {
	    id: string;
	    name: string;
	    os: string;
	    osVersion: string;
	    webglVendor: string;
	    webglRenderer: string;
	    screenWidth: number;
	    screenHeight: number;
	    pixelRatio: number;
	    hardwareConcurrency: number;
	    deviceMemory: number;
	    maxTouchPoints: number;
	    custom?: boolean;
	    deviceType: string;
	    model?: string;
	    deviceName?: string;
	    userAgent?: string;
	    platform?: string;
	
	    static createFrom(source: any = {}) {
	        return new DevicePreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.os = source["os"];
	        this.osVersion = source["osVersion"];
	        this.webglVendor = source["webglVendor"];
	        this.webglRenderer = source["webglRenderer"];
	        this.screenWidth = source["screenWidth"];
	        this.screenHeight = source["screenHeight"];
	        this.pixelRatio = source["pixelRatio"];
	        this.hardwareConcurrency = source["hardwareConcurrency"];
	        this.deviceMemory = source["deviceMemory"];
	        this.maxTouchPoints = source["maxTouchPoints"];
	        this.custom = source["custom"];
	        this.deviceType = source["deviceType"];
	        this.model = source["model"];
	        this.deviceName = source["deviceName"];
	        this.userAgent = source["userAgent"];
	        this.platform = source["platform"];
	    }
	}
	export class DeviceTypeInfo {
	    id: string;
	    name: string;
	    oses: string[];
	    trust: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceTypeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.oses = source["oses"];
	        this.trust = source["trust"];
	    }
	}
	export class EnvItem {
	    key: string;
	    label: string;
	    status: string;
	    actual: string;
	    expected: string;
	    detail: string;
	    fixable: boolean;
	    fixHint: string;
	
	    static createFrom(source: any = {}) {
	        return new EnvItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.actual = source["actual"];
	        this.expected = source["expected"];
	        this.detail = source["detail"];
	        this.fixable = source["fixable"];
	        this.fixHint = source["fixHint"];
	    }
	}
	export class GeoMatch {
	    ok: boolean;
	    error: string;
	    stale: boolean;
	    direct: boolean;
	    ip: string;
	    country: string;
	    region: string;
	    city: string;
	    isp: string;
	    ipType: string;
	    timezone: string;
	    languages: string[];
	    lat: number;
	    lon: number;
	    latency: number;
	
	    static createFrom(source: any = {}) {
	        return new GeoMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.stale = source["stale"];
	        this.direct = source["direct"];
	        this.ip = source["ip"];
	        this.country = source["country"];
	        this.region = source["region"];
	        this.city = source["city"];
	        this.isp = source["isp"];
	        this.ipType = source["ipType"];
	        this.timezone = source["timezone"];
	        this.languages = source["languages"];
	        this.lat = source["lat"];
	        this.lon = source["lon"];
	        this.latency = source["latency"];
	    }
	}
	export class EnvReport {
	    profileId: string;
	    profileName: string;
	    kernel: string;
	    checkedAt: number;
	    exit: GeoMatch;
	    items: EnvItem[];
	    pass: number;
	    warn: number;
	    fail: number;
	    fixable: string[];
	    clock: ClockReport;
	
	    static createFrom(source: any = {}) {
	        return new EnvReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.profileName = source["profileName"];
	        this.kernel = source["kernel"];
	        this.checkedAt = source["checkedAt"];
	        this.exit = this.convertValues(source["exit"], GeoMatch);
	        this.items = this.convertValues(source["items"], EnvItem);
	        this.pass = source["pass"];
	        this.warn = source["warn"];
	        this.fail = source["fail"];
	        this.fixable = source["fixable"];
	        this.clock = this.convertValues(source["clock"], ClockReport);
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
	export class EnvStatus {
	    ready: boolean;
	    kitDir: string;
	    chromeExe: string;
	    phase: string;
	    kind?: string;
	    pct: number;
	    done: number;
	    total: number;
	    speedBps: number;
	    message?: string;
	    error?: string;
	    updatedAt: number;
	    manifestFromServer: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EnvStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.kitDir = source["kitDir"];
	        this.chromeExe = source["chromeExe"];
	        this.phase = source["phase"];
	        this.kind = source["kind"];
	        this.pct = source["pct"];
	        this.done = source["done"];
	        this.total = source["total"];
	        this.speedBps = source["speedBps"];
	        this.message = source["message"];
	        this.error = source["error"];
	        this.updatedAt = source["updatedAt"];
	        this.manifestFromServer = source["manifestFromServer"];
	    }
	}
	export class Extension {
	    id: string;
	    name: string;
	    version: string;
	    description: string;
	    source: string;
	    origin: string;
	    dir: string;
	    icon: string;
	    manifestVersion: number;
	    size: number;
	    addedAt: number;
	    profiles: number;
	
	    static createFrom(source: any = {}) {
	        return new Extension(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.source = source["source"];
	        this.origin = source["origin"];
	        this.dir = source["dir"];
	        this.icon = source["icon"];
	        this.manifestVersion = source["manifestVersion"];
	        this.size = source["size"];
	        this.addedAt = source["addedAt"];
	        this.profiles = source["profiles"];
	    }
	}
	export class FPCatalog {
	    osVersions: string[];
	    browserMajors: string[];
	    userAgents: string[];
	    platforms: string[];
	    vendors: string[];
	    gpus: any[];
	    fonts: string[];
	    screens: string[];
	    cores: string[];
	    memory: string[];
	    custom: Record<string, Array<string>>;
	
	    static createFrom(source: any = {}) {
	        return new FPCatalog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.osVersions = source["osVersions"];
	        this.browserMajors = source["browserMajors"];
	        this.userAgents = source["userAgents"];
	        this.platforms = source["platforms"];
	        this.vendors = source["vendors"];
	        this.gpus = source["gpus"];
	        this.fonts = source["fonts"];
	        this.screens = source["screens"];
	        this.cores = source["cores"];
	        this.memory = source["memory"];
	        this.custom = source["custom"];
	    }
	}
	export class Fingerprint {
	    os: string;
	    deviceType: string;
	    deviceModel?: string;
	    deviceName?: string;
	    osVersion: string;
	    browserVersion: string;
	    userAgent: string;
	    platform: string;
	    screenWidth: number;
	    screenHeight: number;
	    pixelRatio: number;
	    colorDepth: number;
	    hardwareConcurrency: number;
	    deviceMemory: number;
	    maxTouchPoints: number;
	    webglVendor: string;
	    webglRenderer: string;
	    languages: string[];
	    languageFromIP: boolean;
	    timezone: string;
	    timezoneFromIP: boolean;
	    geoFromIP: boolean;
	    latitude: number;
	    longitude: number;
	    canvasNoise: boolean;
	    audioNoise: boolean;
	    webglNoise: boolean;
	    clientRectNoise: boolean;
	    noiseSeed: number;
	    webrtc: string;
	    fonts: string[];
	    doNotTrack: boolean;
	    hideMediaDevices: boolean;
	    canvasMode?: string;
	    webglImageMode?: string;
	    audioMode?: string;
	    clientRectsMode?: string;
	    mediaDevicesMode?: string;
	    mediaMics?: number;
	    mediaSpeakers?: number;
	    mediaCameras?: number;
	    speechVoices?: string;
	    webgpu?: string;
	    portScanProtection?: boolean;
	    scanWhitelist?: string;
	    hardwareAcceleration?: boolean;
	    randomFingerprintOnLaunch?: boolean;
	    windowMode?: string;
	    windowWidth?: number;
	    windowHeight?: number;
	    windowX?: number;
	    windowY?: number;
	    blockImages?: boolean;
	    blockVideo?: boolean;
	    blockSound?: boolean;
	    startupParameters?: string;
	    mediaIdAlias?: boolean;
	    ffStrict?: boolean;
	    ffFontVisibility?: number;
	    ffWebglOff?: boolean;
	    ffUaOverride?: boolean;
	    gpc?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Fingerprint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.deviceType = source["deviceType"];
	        this.deviceModel = source["deviceModel"];
	        this.deviceName = source["deviceName"];
	        this.osVersion = source["osVersion"];
	        this.browserVersion = source["browserVersion"];
	        this.userAgent = source["userAgent"];
	        this.platform = source["platform"];
	        this.screenWidth = source["screenWidth"];
	        this.screenHeight = source["screenHeight"];
	        this.pixelRatio = source["pixelRatio"];
	        this.colorDepth = source["colorDepth"];
	        this.hardwareConcurrency = source["hardwareConcurrency"];
	        this.deviceMemory = source["deviceMemory"];
	        this.maxTouchPoints = source["maxTouchPoints"];
	        this.webglVendor = source["webglVendor"];
	        this.webglRenderer = source["webglRenderer"];
	        this.languages = source["languages"];
	        this.languageFromIP = source["languageFromIP"];
	        this.timezone = source["timezone"];
	        this.timezoneFromIP = source["timezoneFromIP"];
	        this.geoFromIP = source["geoFromIP"];
	        this.latitude = source["latitude"];
	        this.longitude = source["longitude"];
	        this.canvasNoise = source["canvasNoise"];
	        this.audioNoise = source["audioNoise"];
	        this.webglNoise = source["webglNoise"];
	        this.clientRectNoise = source["clientRectNoise"];
	        this.noiseSeed = source["noiseSeed"];
	        this.webrtc = source["webrtc"];
	        this.fonts = source["fonts"];
	        this.doNotTrack = source["doNotTrack"];
	        this.hideMediaDevices = source["hideMediaDevices"];
	        this.canvasMode = source["canvasMode"];
	        this.webglImageMode = source["webglImageMode"];
	        this.audioMode = source["audioMode"];
	        this.clientRectsMode = source["clientRectsMode"];
	        this.mediaDevicesMode = source["mediaDevicesMode"];
	        this.mediaMics = source["mediaMics"];
	        this.mediaSpeakers = source["mediaSpeakers"];
	        this.mediaCameras = source["mediaCameras"];
	        this.speechVoices = source["speechVoices"];
	        this.webgpu = source["webgpu"];
	        this.portScanProtection = source["portScanProtection"];
	        this.scanWhitelist = source["scanWhitelist"];
	        this.hardwareAcceleration = source["hardwareAcceleration"];
	        this.randomFingerprintOnLaunch = source["randomFingerprintOnLaunch"];
	        this.windowMode = source["windowMode"];
	        this.windowWidth = source["windowWidth"];
	        this.windowHeight = source["windowHeight"];
	        this.windowX = source["windowX"];
	        this.windowY = source["windowY"];
	        this.blockImages = source["blockImages"];
	        this.blockVideo = source["blockVideo"];
	        this.blockSound = source["blockSound"];
	        this.startupParameters = source["startupParameters"];
	        this.mediaIdAlias = source["mediaIdAlias"];
	        this.ffStrict = source["ffStrict"];
	        this.ffFontVisibility = source["ffFontVisibility"];
	        this.ffWebglOff = source["ffWebglOff"];
	        this.ffUaOverride = source["ffUaOverride"];
	        this.gpc = source["gpc"];
	    }
	}
	export class FirefoxInfo {
	    path: string;
	    auto: string;
	    version: string;
	    esr: boolean;
	    custom: boolean;
	    missing: boolean;
	    downloadUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new FirefoxInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.auto = source["auto"];
	        this.version = source["version"];
	        this.esr = source["esr"];
	        this.custom = source["custom"];
	        this.missing = source["missing"];
	        this.downloadUrl = source["downloadUrl"];
	    }
	}
	
	export class GroupInfo {
	    name: string;
	    count: number;
	    running: number;
	    saved: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GroupInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.count = source["count"];
	        this.running = source["running"];
	        this.saved = source["saved"];
	    }
	}
	export class InstalledKernel {
	    name: string;
	    version: string;
	    path: string;
	    dir: string;
	    size: number;
	    installed: number;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InstalledKernel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.path = source["path"];
	        this.dir = source["dir"];
	        this.size = source["size"];
	        this.installed = source["installed"];
	        this.active = source["active"];
	    }
	}
	export class KernelCompat {
	    path: string;
	    version: string;
	    major: number;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new KernelCompat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.version = source["version"];
	        this.major = source["major"];
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}
	export class KernelInfo {
	    id: string;
	    name: string;
	    path: string;
	    found: boolean;
	    version: string;
	    auto: boolean;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new KernelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.found = source["found"];
	        this.version = source["version"];
	        this.auto = source["auto"];
	        this.note = source["note"];
	    }
	}
	export class KernelProgress {
	    phase: string;
	    running: boolean;
	    source: string;
	    revision?: string;
	    version?: string;
	    path?: string;
	    url?: string;
	    done: number;
	    total: number;
	    pct: number;
	    speed: number;
	    eta: number;
	    resumed?: boolean;
	    message: string;
	    hint?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new KernelProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.running = source["running"];
	        this.source = source["source"];
	        this.revision = source["revision"];
	        this.version = source["version"];
	        this.path = source["path"];
	        this.url = source["url"];
	        this.done = source["done"];
	        this.total = source["total"];
	        this.pct = source["pct"];
	        this.speed = source["speed"];
	        this.eta = source["eta"];
	        this.resumed = source["resumed"];
	        this.message = source["message"];
	        this.hint = source["hint"];
	        this.error = source["error"];
	    }
	}
	export class McpStatus {
	    running: boolean;
	    enabled: boolean;
	    port: number;
	    url: string;
	    error: string;
	    keyRequired: boolean;
	    tools: string[];
	    calls: number;
	    lastCall: number;
	    lastClient: string;
	    startedAt: number;
	    exePath: string;
	    protocol: string;
	    configHttp: string;
	    configStdio: string;
	    claudeCodeCl: string;
	
	    static createFrom(source: any = {}) {
	        return new McpStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.enabled = source["enabled"];
	        this.port = source["port"];
	        this.url = source["url"];
	        this.error = source["error"];
	        this.keyRequired = source["keyRequired"];
	        this.tools = source["tools"];
	        this.calls = source["calls"];
	        this.lastCall = source["lastCall"];
	        this.lastClient = source["lastClient"];
	        this.startedAt = source["startedAt"];
	        this.exePath = source["exePath"];
	        this.protocol = source["protocol"];
	        this.configHttp = source["configHttp"];
	        this.configStdio = source["configStdio"];
	        this.claudeCodeCl = source["claudeCodeCl"];
	    }
	}
	export class MihomoNode {
	    name: string;
	    type: string;
	    now?: string;
	    all?: string[];
	    delay: number;
	    alive: boolean;
	    udp: boolean;
	    isGroup: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MihomoNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.now = source["now"];
	        this.all = source["all"];
	        this.delay = source["delay"];
	        this.alive = source["alive"];
	        this.udp = source["udp"];
	        this.isGroup = source["isGroup"];
	    }
	}
	export class MihomoProxies {
	    groups: MihomoNode[];
	    nodes: MihomoNode[];
	
	    static createFrom(source: any = {}) {
	        return new MihomoProxies(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groups = this.convertValues(source["groups"], MihomoNode);
	        this.nodes = this.convertValues(source["nodes"], MihomoNode);
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
	export class MihomoStatus {
	    configured: boolean;
	    connected: boolean;
	    controller: string;
	    hasSecret: boolean;
	    version: string;
	    meta: boolean;
	    mode: string;
	    listenHost: string;
	    listenPort: number;
	    listenKind: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new MihomoStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configured = source["configured"];
	        this.connected = source["connected"];
	        this.controller = source["controller"];
	        this.hasSecret = source["hasSecret"];
	        this.version = source["version"];
	        this.meta = source["meta"];
	        this.mode = source["mode"];
	        this.listenHost = source["listenHost"];
	        this.listenPort = source["listenPort"];
	        this.listenKind = source["listenKind"];
	        this.error = source["error"];
	    }
	}
	export class NativePreset {
	    id: string;
	    name: string;
	    configId: string;
	
	    static createFrom(source: any = {}) {
	        return new NativePreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.configId = source["configId"];
	    }
	}
	export class NativeProfile {
	    extraArgs?: string[];
	    headless?: boolean;
	    templateKey: string;
	    sourceConfigId: string;
	    data: Record<string, any>;
	    configMeta: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new NativeProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.extraArgs = source["extraArgs"];
	        this.headless = source["headless"];
	        this.templateKey = source["templateKey"];
	        this.sourceConfigId = source["sourceConfigId"];
	        this.data = source["data"];
	        this.configMeta = source["configMeta"];
	    }
	}
	export class Proxy {
	    id: string;
	    name: string;
	    type: string;
	    host: string;
	    port: number;
	    user: string;
	    pass: string;
	    changeIpUrl: string;
	    tag: string;
	    rotateOnLaunch?: boolean;
	    typeGuessed?: boolean;
	    ip: string;
	    country: string;
	    city: string;
	    timezone: string;
	    lat: number;
	    lon: number;
	    latency: number;
	    connectMs: number;
	    region: string;
	    isp: string;
	    ipType: string;
	    speedKbps: number;
	    speedAt: number;
	    error: string;
	    status: string;
	    checkedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Proxy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.pass = source["pass"];
	        this.changeIpUrl = source["changeIpUrl"];
	        this.tag = source["tag"];
	        this.rotateOnLaunch = source["rotateOnLaunch"];
	        this.typeGuessed = source["typeGuessed"];
	        this.ip = source["ip"];
	        this.country = source["country"];
	        this.city = source["city"];
	        this.timezone = source["timezone"];
	        this.lat = source["lat"];
	        this.lon = source["lon"];
	        this.latency = source["latency"];
	        this.connectMs = source["connectMs"];
	        this.region = source["region"];
	        this.isp = source["isp"];
	        this.ipType = source["ipType"];
	        this.speedKbps = source["speedKbps"];
	        this.speedAt = source["speedAt"];
	        this.error = source["error"];
	        this.status = source["status"];
	        this.checkedAt = source["checkedAt"];
	    }
	}
	export class Profile {
	    native?: NativeProfile;
	    id: string;
	    name: string;
	    group: string;
	    tags: string[];
	    notes: string;
	    storage: string;
	    proxyId: string;
	    inlineProxy?: Proxy;
	    startUrls: string[];
	    cookies: string;
	    fingerprint: Fingerprint;
	    createdAt: number;
	    lastRun: number;
	    cloudVersion: number;
	    syncedAt: number;
	    keepLocalVersion?: number;
	    chunkKey?: string;
	    kernel?: string;
	    extensionIds?: string[];
	    trashed?: boolean;
	    trashedAt?: number;
	    trashDir?: string;
	    trashCloudDeleted?: boolean;
	    shareId?: string;
	    sharedFrom?: string;
	    shareMode?: string;
	    shareExpires?: number;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.native = this.convertValues(source["native"], NativeProfile);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.group = source["group"];
	        this.tags = source["tags"];
	        this.notes = source["notes"];
	        this.storage = source["storage"];
	        this.proxyId = source["proxyId"];
	        this.inlineProxy = this.convertValues(source["inlineProxy"], Proxy);
	        this.startUrls = source["startUrls"];
	        this.cookies = source["cookies"];
	        this.fingerprint = this.convertValues(source["fingerprint"], Fingerprint);
	        this.createdAt = source["createdAt"];
	        this.lastRun = source["lastRun"];
	        this.cloudVersion = source["cloudVersion"];
	        this.syncedAt = source["syncedAt"];
	        this.keepLocalVersion = source["keepLocalVersion"];
	        this.chunkKey = source["chunkKey"];
	        this.kernel = source["kernel"];
	        this.extensionIds = source["extensionIds"];
	        this.trashed = source["trashed"];
	        this.trashedAt = source["trashedAt"];
	        this.trashDir = source["trashDir"];
	        this.trashCloudDeleted = source["trashCloudDeleted"];
	        this.shareId = source["shareId"];
	        this.sharedFrom = source["sharedFrom"];
	        this.shareMode = source["shareMode"];
	        this.shareExpires = source["shareExpires"];
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
	
	export class ProxyPreview {
	    line: number;
	    raw: string;
	    ok: boolean;
	    error: string;
	    dup: boolean;
	    proxy: Proxy;
	
	    static createFrom(source: any = {}) {
	        return new ProxyPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.raw = source["raw"];
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.dup = source["dup"];
	        this.proxy = this.convertValues(source["proxy"], Proxy);
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
	export class RecordFinalizeOptions {
	    keepDelays: boolean;
	    mergeFills: boolean;
	    passwordVar: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RecordFinalizeOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keepDelays = source["keepDelays"];
	        this.mergeFills = source["mergeFills"];
	        this.passwordVar = source["passwordVar"];
	    }
	}
	export class RpaStep {
	    id: string;
	    type: string;
	    name?: string;
	    disabled?: boolean;
	    continueOnError?: boolean;
	    params: Record<string, string>;
	    children?: RpaStep[];
	    else?: RpaStep[];
	
	    static createFrom(source: any = {}) {
	        return new RpaStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.disabled = source["disabled"];
	        this.continueOnError = source["continueOnError"];
	        this.params = source["params"];
	        this.children = this.convertValues(source["children"], RpaStep);
	        this.else = this.convertValues(source["else"], RpaStep);
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
	export class RecordFinalized {
	    steps: RpaStep[];
	    vars: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RecordFinalized(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steps = this.convertValues(source["steps"], RpaStep);
	        this.vars = source["vars"];
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
	export class RecordOptions {
	    overlay: boolean;
	    scroll: boolean;
	    passwordVar: boolean;
	    maxDelayMs: number;
	    startUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new RecordOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.overlay = source["overlay"];
	        this.scroll = source["scroll"];
	        this.passwordVar = source["passwordVar"];
	        this.maxDelayMs = source["maxDelayMs"];
	        this.startUrl = source["startUrl"];
	    }
	}
	export class RecordState {
	    active: boolean;
	    paused: boolean;
	    profileId: string;
	    profileName: string;
	    startedAt: number;
	    pausedMs: number;
	    pausedAt: number;
	    steps: RpaStep[];
	    vars: Record<string, string>;
	    tabs: number;
	    reason?: string;
	    stoppedBy?: string;
	    options: RecordOptions;
	
	    static createFrom(source: any = {}) {
	        return new RecordState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.paused = source["paused"];
	        this.profileId = source["profileId"];
	        this.profileName = source["profileName"];
	        this.startedAt = source["startedAt"];
	        this.pausedMs = source["pausedMs"];
	        this.pausedAt = source["pausedAt"];
	        this.steps = this.convertValues(source["steps"], RpaStep);
	        this.vars = source["vars"];
	        this.tabs = source["tabs"];
	        this.reason = source["reason"];
	        this.stoppedBy = source["stoppedBy"];
	        this.options = this.convertValues(source["options"], RecordOptions);
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
	export class RecoveryState {
	    supported: boolean;
	    configured: boolean;
	    unlocked: boolean;
	    legacy: boolean;
	    kid: string;
	    rkCreated: number;
	    pwChanged: number;
	    shareReady: boolean;
	    user: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RecoveryState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supported = source["supported"];
	        this.configured = source["configured"];
	        this.unlocked = source["unlocked"];
	        this.legacy = source["legacy"];
	        this.kid = source["kid"];
	        this.rkCreated = source["rkCreated"];
	        this.pwChanged = source["pwChanged"];
	        this.shareReady = source["shareReady"];
	        this.user = source["user"];
	        this.error = source["error"];
	    }
	}
	
	export class RepairChange {
	    item: string;
	    label: string;
	    field: string;
	    old: string;
	    new: string;
	    prior: number[];
	    after: number[];
	
	    static createFrom(source: any = {}) {
	        return new RepairChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item = source["item"];
	        this.label = source["label"];
	        this.field = source["field"];
	        this.old = source["old"];
	        this.new = source["new"];
	        this.prior = source["prior"];
	        this.after = source["after"];
	    }
	}
	export class RepairEntry {
	    id: string;
	    profileId: string;
	    at: number;
	    kind: string;
	    items: string[];
	    changes: RepairChange[];
	    skipped: string[];
	    restored: boolean;
	    restoredAt?: number;
	    restoreOf?: string;
	
	    static createFrom(source: any = {}) {
	        return new RepairEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.profileId = source["profileId"];
	        this.at = source["at"];
	        this.kind = source["kind"];
	        this.items = source["items"];
	        this.changes = this.convertValues(source["changes"], RepairChange);
	        this.skipped = source["skipped"];
	        this.restored = source["restored"];
	        this.restoredAt = source["restoredAt"];
	        this.restoreOf = source["restoreOf"];
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
	export class RepairResult {
	    entry?: RepairEntry;
	    changed: number;
	    skipped: string[];
	    profile: Profile;
	
	    static createFrom(source: any = {}) {
	        return new RepairResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = this.convertValues(source["entry"], RepairEntry);
	        this.changed = source["changed"];
	        this.skipped = source["skipped"];
	        this.profile = this.convertValues(source["profile"], Profile);
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
	export class RestoreResult {
	    id: string;
	    oldId?: string;
	    name: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RestoreResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.oldId = source["oldId"];
	        this.name = source["name"];
	        this.error = source["error"];
	    }
	}
	export class RpaIssue {
	    stepId: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new RpaIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stepId = source["stepId"];
	        this.message = source["message"];
	    }
	}
	export class RpaStepEvent {
	    stepId: string;
	    index: number;
	    type: string;
	    status: string;
	    log: string;
	    ms: number;
	    iter?: number;
	    vars?: Record<string, string>;
	    shot?: string;
	
	    static createFrom(source: any = {}) {
	        return new RpaStepEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stepId = source["stepId"];
	        this.index = source["index"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.log = source["log"];
	        this.ms = source["ms"];
	        this.iter = source["iter"];
	        this.vars = source["vars"];
	        this.shot = source["shot"];
	    }
	}
	export class RpaTarget {
	    profileId: string;
	    profileName: string;
	    status: string;
	    error: string;
	    startedAt: number;
	    endedAt: number;
	    autoStarted: boolean;
	    steps: RpaStepEvent[];
	    vars: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RpaTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.profileName = source["profileName"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.startedAt = source["startedAt"];
	        this.endedAt = source["endedAt"];
	        this.autoStarted = source["autoStarted"];
	        this.steps = this.convertValues(source["steps"], RpaStepEvent);
	        this.vars = source["vars"];
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
	export class RpaRun {
	    id: string;
	    workflowId: string;
	    workflowName: string;
	    status: string;
	    startedAt: number;
	    endedAt: number;
	    concurrency: number;
	    stepCount: number;
	    targets: RpaTarget[];
	
	    static createFrom(source: any = {}) {
	        return new RpaRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workflowId = source["workflowId"];
	        this.workflowName = source["workflowName"];
	        this.status = source["status"];
	        this.startedAt = source["startedAt"];
	        this.endedAt = source["endedAt"];
	        this.concurrency = source["concurrency"];
	        this.stepCount = source["stepCount"];
	        this.targets = this.convertValues(source["targets"], RpaTarget);
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
	export class RpaRunOptions {
	    concurrency: number;
	    autoStart: boolean;
	    closeAfter: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RpaRunOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.concurrency = source["concurrency"];
	        this.autoStart = source["autoStart"];
	        this.closeAfter = source["closeAfter"];
	    }
	}
	
	
	
	export class RunState {
	    profileId: string;
	    pid: number;
	    debugPort: number;
	    wsUrl: string;
	    startedAt: number;
	    kernel?: string;
	    bidiUrl?: string;
	
	    static createFrom(source: any = {}) {
	        return new RunState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.pid = source["pid"];
	        this.debugPort = source["debugPort"];
	        this.wsUrl = source["wsUrl"];
	        this.startedAt = source["startedAt"];
	        this.kernel = source["kernel"];
	        this.bidiUrl = source["bidiUrl"];
	    }
	}
	export class SessionStoreInfo {
	    enabled: boolean;
	    schema: number;
	    savedAt: number;
	    cookies: number;
	    partitioned: number;
	    origins: number;
	    lsKeys: number;
	    idbDbs: number;
	    idbRecords: number;
	    idbPartial: boolean;
	    dropped: number;
	    bytes: number;
	    capBytes: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionStoreInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.schema = source["schema"];
	        this.savedAt = source["savedAt"];
	        this.cookies = source["cookies"];
	        this.partitioned = source["partitioned"];
	        this.origins = source["origins"];
	        this.lsKeys = source["lsKeys"];
	        this.idbDbs = source["idbDbs"];
	        this.idbRecords = source["idbRecords"];
	        this.idbPartial = source["idbPartial"];
	        this.dropped = source["dropped"];
	        this.bytes = source["bytes"];
	        this.capBytes = source["capBytes"];
	        this.error = source["error"];
	    }
	}
	export class TerminalPrefs {
	    tool: string;
	    dir: string;
	    envMode: string;
	    profileId: string;
	    tz: string;
	    lang: string;
	    host: string;
	
	    static createFrom(source: any = {}) {
	        return new TerminalPrefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.dir = source["dir"];
	        this.envMode = source["envMode"];
	        this.profileId = source["profileId"];
	        this.tz = source["tz"];
	        this.lang = source["lang"];
	        this.host = source["host"];
	    }
	}
	export class SyncerPrefs {
	    click: boolean;
	    input: boolean;
	    key: boolean;
	    scroll: boolean;
	    tabs: boolean;
	    delayMin: number;
	    delayMax: number;
	    master: string;
	    followers: string[];
	    newTab: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SyncerPrefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.click = source["click"];
	        this.input = source["input"];
	        this.key = source["key"];
	        this.scroll = source["scroll"];
	        this.tabs = source["tabs"];
	        this.delayMin = source["delayMin"];
	        this.delayMax = source["delayMax"];
	        this.master = source["master"];
	        this.followers = source["followers"];
	        this.newTab = source["newTab"];
	    }
	}
	export class Settings {
	    undetectableKit: string;
	    kernelPath: string;
	    kernelMode: string;
	    chromiumPath: string;
	    profilesDir: string;
	    apiPort: number;
	    apiEnabled: boolean;
	    apiKey: string;
	    apiKeyOff: boolean;
	    defaultOS: string;
	    syncServer: string;
	    syncToken: string;
	    syncPassword: string;
	    autoSyncOff: boolean;
	    syncPromptSecs: number;
	    syncPromptAction: string;
	    deviceName: string;
	    deviceId: string;
	    theme: string;
	    proxyCheckMins: number;
	    kernelSource?: string;
	    kernelUrl?: string;
	    syncer?: SyncerPrefs;
	    deleteMode?: string;
	    trashDays?: number;
	    syncSessionStorage?: boolean;
	    sessionStoreCapMB?: number;
	    terminal?: TerminalPrefs;
	    firefoxPath?: string;
	    closeToTrayOff?: boolean;
	    updateAutoOff?: boolean;
	    updateUrl?: string;
	    updateGithub?: string;
	    catalogUrl?: string;
	    defaultProfileId?: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.undetectableKit = source["undetectableKit"];
	        this.kernelPath = source["kernelPath"];
	        this.kernelMode = source["kernelMode"];
	        this.chromiumPath = source["chromiumPath"];
	        this.profilesDir = source["profilesDir"];
	        this.apiPort = source["apiPort"];
	        this.apiEnabled = source["apiEnabled"];
	        this.apiKey = source["apiKey"];
	        this.apiKeyOff = source["apiKeyOff"];
	        this.defaultOS = source["defaultOS"];
	        this.syncServer = source["syncServer"];
	        this.syncToken = source["syncToken"];
	        this.syncPassword = source["syncPassword"];
	        this.autoSyncOff = source["autoSyncOff"];
	        this.syncPromptSecs = source["syncPromptSecs"];
	        this.syncPromptAction = source["syncPromptAction"];
	        this.deviceName = source["deviceName"];
	        this.deviceId = source["deviceId"];
	        this.theme = source["theme"];
	        this.proxyCheckMins = source["proxyCheckMins"];
	        this.kernelSource = source["kernelSource"];
	        this.kernelUrl = source["kernelUrl"];
	        this.syncer = this.convertValues(source["syncer"], SyncerPrefs);
	        this.deleteMode = source["deleteMode"];
	        this.trashDays = source["trashDays"];
	        this.syncSessionStorage = source["syncSessionStorage"];
	        this.sessionStoreCapMB = source["sessionStoreCapMB"];
	        this.terminal = this.convertValues(source["terminal"], TerminalPrefs);
	        this.firefoxPath = source["firefoxPath"];
	        this.closeToTrayOff = source["closeToTrayOff"];
	        this.updateAutoOff = source["updateAutoOff"];
	        this.updateUrl = source["updateUrl"];
	        this.updateGithub = source["updateGithub"];
	        this.catalogUrl = source["catalogUrl"];
	        this.defaultProfileId = source["defaultProfileId"];
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
	export class shareWrap {
	    eph: string;
	    nonce: string;
	    ct: string;
	    pub?: string;
	
	    static createFrom(source: any = {}) {
	        return new shareWrap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.eph = source["eph"];
	        this.nonce = source["nonce"];
	        this.ct = source["ct"];
	        this.pub = source["pub"];
	    }
	}
	export class ShareInfo {
	    id: string;
	    from: string;
	    to: string;
	    name: string;
	    mode: string;
	    note: string;
	    os: string;
	    createdAt: number;
	    expiresAt: number;
	    updatedAt: number;
	    size: number;
	    status: string;
	    wrap?: shareWrap;
	    localId?: string;
	
	    static createFrom(source: any = {}) {
	        return new ShareInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.from = source["from"];
	        this.to = source["to"];
	        this.name = source["name"];
	        this.mode = source["mode"];
	        this.note = source["note"];
	        this.os = source["os"];
	        this.createdAt = source["createdAt"];
	        this.expiresAt = source["expiresAt"];
	        this.updatedAt = source["updatedAt"];
	        this.size = source["size"];
	        this.status = source["status"];
	        this.wrap = this.convertValues(source["wrap"], shareWrap);
	        this.localId = source["localId"];
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
	export class ShareList {
	    supported: boolean;
	    ready: boolean;
	    self: string;
	    out: ShareInfo[];
	    in: ShareInfo[];
	    removed: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ShareList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supported = source["supported"];
	        this.ready = source["ready"];
	        this.self = source["self"];
	        this.out = this.convertValues(source["out"], ShareInfo);
	        this.in = this.convertValues(source["in"], ShareInfo);
	        this.removed = source["removed"];
	        this.error = source["error"];
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
	export class ShareRequest {
	    profileId: string;
	    to: string;
	    mode: string;
	    days: number;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new ShareRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.to = source["to"];
	        this.mode = source["mode"];
	        this.days = source["days"];
	        this.note = source["note"];
	    }
	}
	export class SyncItem {
	    id: string;
	    name: string;
	    action: string;
	    reason: string;
	    conflict: boolean;
	    cloudVersion: number;
	    localVersion: number;
	    cloudUpdatedAt: number;
	    cloudDevice: string;
	    localLastRun: number;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.action = source["action"];
	        this.reason = source["reason"];
	        this.conflict = source["conflict"];
	        this.cloudVersion = source["cloudVersion"];
	        this.localVersion = source["localVersion"];
	        this.cloudUpdatedAt = source["cloudUpdatedAt"];
	        this.cloudDevice = source["cloudDevice"];
	        this.localLastRun = source["localLastRun"];
	        this.size = source["size"];
	    }
	}
	export class SyncPrompt {
	    items: SyncItem[];
	    seconds: number;
	    action: string;
	    deadline: number;
	    held: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SyncPrompt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], SyncItem);
	        this.seconds = source["seconds"];
	        this.action = source["action"];
	        this.deadline = source["deadline"];
	        this.held = source["held"];
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
	export class SyncEvent {
	    phase: string;
	    id?: string;
	    name?: string;
	    action?: string;
	    error?: string;
	    pulled: number;
	    pushed: number;
	    failed: number;
	    message?: string;
	    prompt?: SyncPrompt;
	
	    static createFrom(source: any = {}) {
	        return new SyncEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.action = source["action"];
	        this.error = source["error"];
	        this.pulled = source["pulled"];
	        this.pushed = source["pushed"];
	        this.failed = source["failed"];
	        this.message = source["message"];
	        this.prompt = this.convertValues(source["prompt"], SyncPrompt);
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
	export class SyncFollowerState {
	    id: string;
	    name: string;
	    ok: boolean;
	    events: number;
	    error?: string;
	    latencyMs: number;
	    tabs: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncFollowerState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.ok = source["ok"];
	        this.events = source["events"];
	        this.error = source["error"];
	        this.latencyMs = source["latencyMs"];
	        this.tabs = source["tabs"];
	    }
	}
	
	export class SyncLogEntry {
	    time: number;
	    id: string;
	    name: string;
	    action: string;
	    ok: boolean;
	    error: string;
	    bytes: number;
	    ms: number;
	    version: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.action = source["action"];
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.bytes = source["bytes"];
	        this.ms = source["ms"];
	        this.version = source["version"];
	    }
	}
	export class SyncOptions {
	    click: boolean;
	    input: boolean;
	    key: boolean;
	    scroll: boolean;
	    tabs: boolean;
	    delayMin: number;
	    delayMax: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.click = source["click"];
	        this.input = source["input"];
	        this.key = source["key"];
	        this.scroll = source["scroll"];
	        this.tabs = source["tabs"];
	        this.delayMin = source["delayMin"];
	        this.delayMax = source["delayMax"];
	    }
	}
	
	
	export class SyncerState {
	    running: boolean;
	    master: string;
	    masterName: string;
	    followers: SyncFollowerState[];
	    events: number;
	    lastEvent: string;
	    startedAt: number;
	    reason?: string;
	    options: SyncOptions;
	    tabs: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncerState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.master = source["master"];
	        this.masterName = source["masterName"];
	        this.followers = this.convertValues(source["followers"], SyncFollowerState);
	        this.events = source["events"];
	        this.lastEvent = source["lastEvent"];
	        this.startedAt = source["startedAt"];
	        this.reason = source["reason"];
	        this.options = this.convertValues(source["options"], SyncOptions);
	        this.tabs = source["tabs"];
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
	export class TabActionResult {
	    action: string;
	    done: number;
	    total: number;
	    failed: string[];
	    url?: string;
	
	    static createFrom(source: any = {}) {
	        return new TabActionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.done = source["done"];
	        this.total = source["total"];
	        this.failed = source["failed"];
	        this.url = source["url"];
	    }
	}
	export class TagInfo {
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new TagInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	
	export class TerminalRequest {
	    tool: string;
	    dir: string;
	    envMode: string;
	    profileId: string;
	    tz: string;
	    lang: string;
	    host: string;
	
	    static createFrom(source: any = {}) {
	        return new TerminalRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.dir = source["dir"];
	        this.envMode = source["envMode"];
	        this.profileId = source["profileId"];
	        this.tz = source["tz"];
	        this.lang = source["lang"];
	        this.host = source["host"];
	    }
	}
	export class TerminalSession {
	    id: string;
	    tool: string;
	    name: string;
	    dir: string;
	    profileId: string;
	    profileName: string;
	    tz: string;
	    lang: string;
	    proxy: string;
	    relay: string;
	    host: string;
	    pid: number;
	    tracked: boolean;
	    started: number;
	
	    static createFrom(source: any = {}) {
	        return new TerminalSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tool = source["tool"];
	        this.name = source["name"];
	        this.dir = source["dir"];
	        this.profileId = source["profileId"];
	        this.profileName = source["profileName"];
	        this.tz = source["tz"];
	        this.lang = source["lang"];
	        this.proxy = source["proxy"];
	        this.relay = source["relay"];
	        this.host = source["host"];
	        this.pid = source["pid"];
	        this.tracked = source["tracked"];
	        this.started = source["started"];
	    }
	}
	export class TerminalTool {
	    id: string;
	    name: string;
	    bin: string;
	    installed: boolean;
	    version: string;
	    path: string;
	    installUrl: string;
	    installCmd: string;
	    desc: string;
	
	    static createFrom(source: any = {}) {
	        return new TerminalTool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.bin = source["bin"];
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.path = source["path"];
	        this.installUrl = source["installUrl"];
	        this.installCmd = source["installCmd"];
	        this.desc = source["desc"];
	    }
	}
	export class TransferEvent {
	    id: string;
	    name: string;
	    dir: string;
	    phase: string;
	    done: number;
	    total: number;
	    speed: number;
	    eta: number;
	    saved: number;
	    started: number;
	    error?: string;
	    final: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TransferEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.dir = source["dir"];
	        this.phase = source["phase"];
	        this.done = source["done"];
	        this.total = source["total"];
	        this.speed = source["speed"];
	        this.eta = source["eta"];
	        this.saved = source["saved"];
	        this.started = source["started"];
	        this.error = source["error"];
	        this.final = source["final"];
	    }
	}
	export class TrashItem {
	    native?: NativeProfile;
	    id: string;
	    name: string;
	    group: string;
	    tags: string[];
	    notes: string;
	    storage: string;
	    proxyId: string;
	    inlineProxy?: Proxy;
	    startUrls: string[];
	    cookies: string;
	    fingerprint: Fingerprint;
	    createdAt: number;
	    lastRun: number;
	    cloudVersion: number;
	    syncedAt: number;
	    keepLocalVersion?: number;
	    chunkKey?: string;
	    kernel?: string;
	    extensionIds?: string[];
	    trashed?: boolean;
	    trashedAt?: number;
	    trashDir?: string;
	    trashCloudDeleted?: boolean;
	    shareId?: string;
	    sharedFrom?: string;
	    shareMode?: string;
	    shareExpires?: number;
	    size: number;
	    expiresAt: number;
	    hasData: boolean;
	    cloudNote: string;
	    restoreAsNew: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TrashItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.native = this.convertValues(source["native"], NativeProfile);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.group = source["group"];
	        this.tags = source["tags"];
	        this.notes = source["notes"];
	        this.storage = source["storage"];
	        this.proxyId = source["proxyId"];
	        this.inlineProxy = this.convertValues(source["inlineProxy"], Proxy);
	        this.startUrls = source["startUrls"];
	        this.cookies = source["cookies"];
	        this.fingerprint = this.convertValues(source["fingerprint"], Fingerprint);
	        this.createdAt = source["createdAt"];
	        this.lastRun = source["lastRun"];
	        this.cloudVersion = source["cloudVersion"];
	        this.syncedAt = source["syncedAt"];
	        this.keepLocalVersion = source["keepLocalVersion"];
	        this.chunkKey = source["chunkKey"];
	        this.kernel = source["kernel"];
	        this.extensionIds = source["extensionIds"];
	        this.trashed = source["trashed"];
	        this.trashedAt = source["trashedAt"];
	        this.trashDir = source["trashDir"];
	        this.trashCloudDeleted = source["trashCloudDeleted"];
	        this.shareId = source["shareId"];
	        this.sharedFrom = source["sharedFrom"];
	        this.shareMode = source["shareMode"];
	        this.shareExpires = source["shareExpires"];
	        this.size = source["size"];
	        this.expiresAt = source["expiresAt"];
	        this.hasData = source["hasData"];
	        this.cloudNote = source["cloudNote"];
	        this.restoreAsNew = source["restoreAsNew"];
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
	export class UpdateInfo {
	    current: string;
	    latest: string;
	    notes: string;
	    available: boolean;
	    source: string;
	    checkedAt: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.notes = source["notes"];
	        this.available = source["available"];
	        this.source = source["source"];
	        this.checkedAt = source["checkedAt"];
	        this.error = source["error"];
	    }
	}
	export class UpdateProgress {
	    phase: string;
	    got: number;
	    total: number;
	    pct: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.got = source["got"];
	        this.total = source["total"];
	        this.pct = source["pct"];
	        this.error = source["error"];
	    }
	}
	export class Workflow {
	    id: string;
	    name: string;
	    description: string;
	    steps: RpaStep[];
	    vars: Record<string, string>;
	    createdAt: number;
	    updatedAt: number;
	    lastRunAt: number;
	    lastStatus: string;
	    lastRunId: string;
	    runCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Workflow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.steps = this.convertValues(source["steps"], RpaStep);
	        this.vars = source["vars"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.lastRunAt = source["lastRunAt"];
	        this.lastStatus = source["lastStatus"];
	        this.lastRunId = source["lastRunId"];
	        this.runCount = source["runCount"];
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
	export class cloudMeta {
	    id: string;
	    name: string;
	    version: number;
	    sha256: string;
	    meta: number[];
	    lockedBy: string;
	    updatedAt: number;
	    size: number;
	    deviceName: string;
	    deleted?: boolean;
	    deletedAt?: number;
	    deletedByName?: string;
	    deletedBy?: string;
	
	    static createFrom(source: any = {}) {
	        return new cloudMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.sha256 = source["sha256"];
	        this.meta = source["meta"];
	        this.lockedBy = source["lockedBy"];
	        this.updatedAt = source["updatedAt"];
	        this.size = source["size"];
	        this.deviceName = source["deviceName"];
	        this.deleted = source["deleted"];
	        this.deletedAt = source["deletedAt"];
	        this.deletedByName = source["deletedByName"];
	        this.deletedBy = source["deletedBy"];
	    }
	}
	export class serverInfo {
	    features: string[];
	    keepVersions: number;
	    maxBody: number;
	    usedBytes: number;
	    freeBytes: number;
	    user: string;
	    protocol: number;
	
	    static createFrom(source: any = {}) {
	        return new serverInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.features = source["features"];
	        this.keepVersions = source["keepVersions"];
	        this.maxBody = source["maxBody"];
	        this.usedBytes = source["usedBytes"];
	        this.freeBytes = source["freeBytes"];
	        this.user = source["user"];
	        this.protocol = source["protocol"];
	    }
	}

}

