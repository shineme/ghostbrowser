export namespace main {
	
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
	
	    static createFrom(source: any = {}) {
	        return new Fingerprint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
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
	    ip: string;
	    country: string;
	    city: string;
	    timezone: string;
	    lat: number;
	    lon: number;
	    latency: number;
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
	        this.ip = source["ip"];
	        this.country = source["country"];
	        this.city = source["city"];
	        this.timezone = source["timezone"];
	        this.lat = source["lat"];
	        this.lon = source["lon"];
	        this.latency = source["latency"];
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
	    profilesDir: string;
	    apiPort: number;
	    apiEnabled: boolean;
	    defaultOS: string;
	    syncServer: string;
	    syncToken: string;
	    syncPassword: string;
	    autoSyncOff: boolean;
	    deviceName: string;
	    deviceId: string;
	    theme: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.undetectableKit = source["undetectableKit"];
	        this.kernelPath = source["kernelPath"];
	        this.kernelMode = source["kernelMode"];
	        this.profilesDir = source["profilesDir"];
	        this.apiPort = source["apiPort"];
	        this.apiEnabled = source["apiEnabled"];
	        this.defaultOS = source["defaultOS"];
	        this.syncServer = source["syncServer"];
	        this.syncToken = source["syncToken"];
	        this.syncPassword = source["syncPassword"];
	        this.autoSyncOff = source["autoSyncOff"];
	        this.deviceName = source["deviceName"];
	        this.deviceId = source["deviceId"];
	        this.theme = source["theme"];
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
	    }
	}
	export class cloudMeta {
	    id: string;
	    name: string;
	    version: number;
	    sha256: string;
	    meta: number[];
	    lockedBy: string;
	
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
	    }
	}

}

