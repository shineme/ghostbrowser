export namespace main {
	
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
	
	
	export class RunState {
	    profileId: string;
	    pid: number;
	    debugPort: number;
	    wsUrl: string;
	    startedAt: number;
	
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
	
	export class SyncerState {
	    running: boolean;
	    master: string;
	    masterName: string;
	    followers: SyncFollowerState[];
	    events: number;
	    lastEvent: string;
	    startedAt: number;
	    reason?: string;
	
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

