package main

// Fingerprint is the full per-profile fingerprint config. It is written to the
// native startInfo blob that the bundled custom Chromium reads at launch.
type Fingerprint struct {
	OS              string   `json:"os"` // windows | macos | linux
	OSVersion       string   `json:"osVersion"`
	BrowserVersion  string   `json:"browserVersion"`
	UserAgent       string   `json:"userAgent"`
	Platform        string   `json:"platform"`
	ScreenWidth     int      `json:"screenWidth"`
	ScreenHeight    int      `json:"screenHeight"`
	PixelRatio      float64  `json:"pixelRatio"`
	ColorDepth      int      `json:"colorDepth"`
	HardwareConc    int      `json:"hardwareConcurrency"`
	DeviceMemory    int      `json:"deviceMemory"`
	MaxTouchPoints  int      `json:"maxTouchPoints"`
	WebGLVendor     string   `json:"webglVendor"`
	WebGLRenderer   string   `json:"webglRenderer"`
	Languages       []string `json:"languages"`
	LanguageFromIP  bool     `json:"languageFromIP"`
	Timezone        string   `json:"timezone"`
	TimezoneFromIP  bool     `json:"timezoneFromIP"`
	GeoFromIP       bool     `json:"geoFromIP"`
	Latitude        float64  `json:"latitude"`
	Longitude       float64  `json:"longitude"`
	CanvasNoise     bool     `json:"canvasNoise"`
	AudioNoise      bool     `json:"audioNoise"`
	WebGLNoise      bool     `json:"webglNoise"`
	ClientRectNoise bool     `json:"clientRectNoise"`
	NoiseSeed       int64    `json:"noiseSeed"`
	WebRTC          string   `json:"webrtc"` // proxy | disabled | real
	Fonts           []string `json:"fonts"`
	DoNotTrack      bool     `json:"doNotTrack"`
	HideMedia       bool     `json:"hideMediaDevices"`
}

type Proxy struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"` // none | http | https | socks5
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	Pass           string `json:"pass"`
	ChangeIP       string `json:"changeIpUrl"`
	Tag            string `json:"tag"`                      // batch label set on import, searchable
	RotateOnLaunch bool   `json:"rotateOnLaunch,omitempty"` // call ChangeIP before each launch
	Guessed        bool   `json:"typeGuessed,omitempty"`    // protocol inferred; the first check confirms it
	// last check result
	IP       string  `json:"ip"`
	Country  string  `json:"country"`
	City     string  `json:"city"`
	Timezone string  `json:"timezone"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Latency  int64   `json:"latency"`   // warm round trip through the proxy, ms
	Connect  int64   `json:"connectMs"` // first request incl. proxy handshake, ms
	Region   string  `json:"region"`
	ISP      string  `json:"isp"`
	IPType   string  `json:"ipType"` // residential | hosting | mobile
	Speed    int64   `json:"speedKbps"`
	SpeedAt  int64   `json:"speedAt"`
	Error    string  `json:"error"`
	Status   string  `json:"status"` // unknown | ok | fail
	Checked  int64   `json:"checkedAt"`
}

type Profile struct {
	Native      *NativeProfile `json:"native,omitempty"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Group       string         `json:"group"`
	Tags        []string       `json:"tags"`
	Notes       string         `json:"notes"`
	Storage     string         `json:"storage"` // local | cloud
	ProxyID     string         `json:"proxyId"`
	InlineProxy *Proxy         `json:"inlineProxy,omitempty"`
	StartURLs   []string       `json:"startUrls"`
	Cookies     string         `json:"cookies"` // JSON array to import on next start
	Fingerprint Fingerprint    `json:"fingerprint"`
	CreatedAt   int64          `json:"createdAt"`
	LastRun     int64          `json:"lastRun"`
	CloudVer    int64          `json:"cloudVersion"`
	SyncedAt    int64          `json:"syncedAt"`
	KeepLocal   int64          `json:"keepLocalVersion,omitempty"` // user kept local over this cloud version
	ChunkKey    string         `json:"chunkKey,omitempty"`         // incremental sync: per-profile chunk key (base64)
}

type Settings struct {
	UndetectableKit string `json:"undetectableKit"`
	KernelPath      string `json:"kernelPath"`
	KernelMode      string `json:"kernelMode"` // always undetectable
	ProfilesDir     string `json:"profilesDir"`
	APIPort         int    `json:"apiPort"`
	APIEnabled      bool   `json:"apiEnabled"`
	APIKey          string `json:"apiKey"`    // required by the local API (X-API-Key header or ?key=)
	APIKeyOff       bool   `json:"apiKeyOff"` // allow calls without the key (Host / Origin checks stay on)
	DefaultOS       string `json:"defaultOS"`
	SyncServer      string `json:"syncServer"`
	SyncToken       string `json:"syncToken"`
	SyncPassword    string `json:"syncPassword"`
	AutoSyncOff     bool   `json:"autoSyncOff"` // auto sync is on by default once sync is configured
	// Start-up prompt when the cloud has newer copies: wait this many seconds,
	// then apply SyncPromptAction ("pull" or "keep"). 0 = apply at once, no prompt.
	SyncPromptSecs   int    `json:"syncPromptSecs"`
	SyncPromptAction string `json:"syncPromptAction"`
	DeviceName       string `json:"deviceName"`
	DeviceID         string `json:"deviceId"`
	Theme            string `json:"theme"`
	ProxyCheckMins   int    `json:"proxyCheckMins"` // background re-check of every proxy; 0 = off
}

type RunState struct {
	ProfileID string `json:"profileId"`
	PID       int    `json:"pid"`
	DebugPort int    `json:"debugPort"`
	WSURL     string `json:"wsUrl"`
	Started   int64  `json:"startedAt"`
}
