package main

import (
	"os"
	"path/filepath"
	"strings"
)

// 同步内容 (Settings.SyncContent, Profile.SyncContent): which parts of a
// browser profile go to the cloud.
//
//   - 精简 lite (default): login state and setup only — cookies (Network/
//     Cookies, Device Bound Sessions), Local Storage, IndexedDB, WebStorage
//     (storage buckets), Login Data, Web Data (autofill), extensions with
//     their settings, bookmarks, Preferences / Secure Preferences / Local
//     State, MediaDeviceSalts, our fingerprint / session snapshot files.
//   - 标准 standard: lite + browsing history (History, Favicons, Top Sites,
//     Visited Links, Shortcuts), open tabs (Sessions, Session Storage),
//     service workers with their Cache API data (Service Worker incl.
//     CacheStorage and ScriptCache, CacheStorage anywhere), web push
//     registrations (GCM Store), File System, notification permissions DB.
//   - 完整 full: everything except what no mode ever syncs.
//
// Never synced in any mode (Chromium rebuilds or re-downloads all of it):
// HTTP / code / GPU / shader caches (syncExcludedDirs), component-updater
// data at the user-data root (screen_ai, WidevineCdm, Safe Browsing,
// OnDeviceHeadSuggestModel, ZxcvbnData, hyphen-data, Subresource Filter …),
// downloaded models and hint stores, metrics, crash dumps, and per-profile
// throwaway databases (Download Service, Feature Engagement Tracker,
// optimization_guide_*, commerce / coupon / cart DBs, Shared Dictionary,
// Network/Reporting and NEL, BudgetDatabase, VideoDecodeStats …).
//
// Every rule only removes files from new uploads: a manifest or zip made
// with another mode (or by an older client) restores exactly what it lists,
// and a pull keeps the local copies of what the cloud copy does not carry
// (carryLocalOnly), so switching to 精简 never deletes history on the device
// that has it.
const (
	syncScopeLite     = "lite"
	syncScopeStandard = "standard"
	syncScopeFull     = "full"
)

// tiers: the lowest mode that syncs an entry.
const (
	tierLite     = 0
	tierStandard = 1
	tierFull     = 2
	tierNever    = 3
)

func normSyncScope(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case syncScopeStandard:
		return syncScopeStandard
	case syncScopeFull:
		return syncScopeFull
	}
	return syncScopeLite
}

func scopeLevel(scope string) int {
	switch normSyncScope(scope) {
	case syncScopeStandard:
		return tierStandard
	case syncScopeFull:
		return tierFull
	}
	return tierLite
}

// syncScopeFor resolves a profile's mode: its own override, else the global
// setting.
func syncScopeFor(st Settings, p *Profile) string {
	if p != nil && strings.TrimSpace(p.SyncContent) != "" {
		return normSyncScope(p.SyncContent)
	}
	return normSyncScope(st.SyncContent)
}

// syncRootTier: entries directly in the user-data directory (Chromium) or
// the Firefox profile directory.
var syncRootTier = map[string]int{
	// Chromium component updater: everything re-downloads on its own.
	"screen_ai": tierNever, "widevinecdm": tierNever, "ondeviceheadsuggestmodel": tierNever,
	"safe browsing": tierNever, "zxcvbndata": tierNever, "hyphen-data": tierNever,
	"subresource filter": tierNever, "trusttokenkeycommitments": tierNever, "meipreload": tierNever,
	"filetypepolicies": tierNever, "sslerrorassistant": tierNever, "certificaterevocation": tierNever,
	"crowd deny": tierNever, "origintrials": tierNever, "pkimetadata": tierNever,
	"autofillstates": tierNever, "cookiereadinesslist": tierNever, "firstpartysetspreloaded": tierNever,
	"first_party_sets.db": tierNever, "first_party_sets.db-journal": tierNever,
	"privacysandboxattestationspreloaded": tierNever, "safetytips": tierNever, "optimizationhints": tierNever,
	"optimizationguidemodelsmanifest": tierNever, "wasmttsengine": tierNever, "probabilisticrevealtokenregistry": tierNever,
	"amountextractionheuristicregexes": tierNever, "tpcdmetadata": tierNever, "masked domain list": tierNever,
	"commerceheuristics": tierNever, "actorsafetylists": tierNever, "fingerprintingprotection": tierNever,
	"recoveryimproved": tierNever, "thirdpartymodulelist64": tierNever, "thirdpartymodulelist": tierNever,
	"pnacl": tierNever, "swiftshader": tierNever, "gpupersistentcache": tierNever, "webrtcvideostats": tierNever,
	"segmentation_platform": tierFull, "variations": tierNever, "crash reports": tierNever,
	"pepper data": tierNever, "certificate revocation lists": tierNever, "local traces": tierNever,
	"deferred-crash-reports": tierNever, "chromecomponents": tierNever, "captchaproviders": tierNever,
	"launch.log": tierNever, "firefox-launch.log": tierNever, // our launcher logs, rewritten on every start
	// Firefox profile directory.
	"cache2": tierNever, "startupcache": tierNever, "thumbnails": tierNever, "safebrowsing": tierNever,
	"crashes": tierNever, "minidumps": tierNever, "datareporting": tierNever, "saved-telemetry-pings": tierNever,
	"shader-cache": tierNever, "jumplistcache": tierNever, "security_state": tierNever,
	"favicons.sqlite": tierStandard, "favicons.sqlite-wal": tierStandard, "favicons.sqlite-shm": tierStandard,
	"sessionstore-backups": tierStandard, "sessionstore.jsonlz4": tierStandard,
}

// syncProfileTier: entries inside a Chromium profile folder (Default,
// Profile N), keyed by their path below it.
var syncProfileTier = map[string]int{
	// rebuildable / throwaway
	"download service": tierNever, "feature engagement tracker": tierNever,
	"optimization_guide_hint_cache_store": tierNever, "optimization_guide_prediction_model_downloads": tierNever,
	"optimization_guide_model_metadata_store": tierNever, "heavy_ad_intervention_opt_out.db": tierNever,
	"heavy_ad_intervention_opt_out.db-journal": tierNever, "commerce_subscription_db": tierNever,
	"chrome_cart_db": tierNever, "discounts_db": tierNever, "coupon_db": tierNever, "parcel_tracking_db": tierNever,
	"autofillaimodelcache": tierNever, "shared dictionary": tierNever, "budgetdatabase": tierNever,
	"videodecodestats": tierNever, "webrtcvideostats": tierNever, "site characteristics database": tierNever,
	"affiliation database": tierNever, "affiliation database-journal": tierNever,
	"previews_opt_out.db": tierNever, "previews_opt_out.db-journal": tierNever,
	"network/reporting and nel": tierNever, "network/reporting and nel-journal": tierNever,
	"reporting and nel": tierNever, "reporting and nel-journal": tierNever,
	"network/sct auditing pending reports": tierNever, "sct auditing pending reports": tierNever,
	"discount_infos_db": tierNever, "declarative_performance_observer.db": tierNever,
	"declarative_performance_observer.db-journal": tierNever,
	"gpupersistentcache": tierNever, "jumplisticons": tierNever, "jumplisticonsold": tierNever,
	"segmentation_platform": tierFull, "segmentation platform": tierFull, "shared_proto_db": tierFull,
	"servercertificate": tierFull, "servercertificate-journal": tierFull,
	// history, tabs, service workers (标准)
	"history": tierStandard, "history-journal": tierStandard, "history-wal": tierStandard,
	"favicons": tierStandard, "favicons-journal": tierStandard, "favicons-wal": tierStandard,
	"top sites": tierStandard, "top sites-journal": tierStandard, "top sites-wal": tierStandard,
	"visited links": tierStandard, "shortcuts": tierStandard, "shortcuts-journal": tierStandard,
	"sessions": tierStandard, "sessions_encrypted": tierStandard, "current session": tierStandard,
	"current tabs": tierStandard, "last session": tierStandard, "last tabs": tierStandard,
	"session storage": tierStandard, "service worker": tierStandard, "gcm store": tierStandard,
	"file system": tierStandard, "platform notifications": tierStandard,
	// only 完整: browser-internal state nobody misses
	"network action predictor": tierFull, "network action predictor-journal": tierFull,
	"media history": tierFull, "media history-journal": tierFull, "dips": tierFull, "dips-journal": tierFull,
	"dips-wal": tierFull, "dips-shm": tierFull, "sync data": tierFull, "extension state": tierFull,
	"autofillstrikedatabase": tierFull, "persistentorigintrials": tierFull, "trust tokens": tierFull,
	"network/trust tokens": tierFull, "network/trust tokens-journal": tierFull, "conversions": tierFull,
	"conversions-journal": tierFull, "interestgroups": tierFull, "interestgroups-journal": tierFull,
	"interestgroups-wal": tierFull, "sharedstorage": tierFull, "sharedstorage-wal": tierFull,
	"privateaggregation": tierFull, "privateaggregation-journal": tierFull,
	"browsingtopicssitedata": tierFull, "browsingtopicssitedata-journal": tierFull, "browsingtopicsstate": tierFull,
	"attribution reporting": tierFull, "privacy sandbox": tierFull, "aggregation service": tierFull,
	"ondevicesecuremodule": tierFull, "databases": tierFull, "web applications/temp": tierNever,
}

// syncAnyTier: directory names at any depth (CacheStorage also lives in
// WebStorage/<bucket>/ and Storage/ext/<id>/).
var syncAnyTier = map[string]int{"cachestorage": tierStandard, "scriptcache": tierStandard}

func isProfileFolder(name string) bool {
	l := strings.ToLower(name)
	return l == "default" || strings.HasPrefix(l, "profile ") || l == "guest profile" || l == "system profile"
}

// syncTier classifies a path relative to the profile root ("/" separated).
// SQLite companions (-journal, -wal, -shm) share their database's tier:
// after a clean exit they are empty, after a crash they hold data the
// database needs, so they travel exactly when the database does.
func syncTier(rel string) int {
	if t := syncTierExact(rel); t != tierLite {
		return t
	}
	for _, sfx := range []string{"-journal", "-wal", "-shm"} {
		if len(rel) > len(sfx) && strings.EqualFold(rel[len(rel)-len(sfx):], sfx) {
			return syncTierExact(rel[:len(rel)-len(sfx)])
		}
	}
	return tierLite
}

func syncTierExact(rel string) int {
	parts := strings.Split(rel, "/")
	low := strings.ToLower(parts[len(parts)-1])
	if syncExcludedDirs[low] {
		return tierNever
	}
	if t, ok := syncAnyTier[low]; ok {
		return t
	}
	if len(parts) == 1 {
		if t, ok := syncRootTier[low]; ok {
			return t
		}
		return tierLite
	}
	if isProfileFolder(parts[0]) && len(parts) <= 3 {
		if t, ok := syncProfileTier[strings.ToLower(strings.Join(parts[1:], "/"))]; ok {
			return t
		}
	}
	return tierLite
}

// contentScope decides, for one walk, which entries belong in the upload.
type contentScope struct{ level int }

func newSyncScope(scope string) contentScope { return contentScope{level: scopeLevel(scope)} }

// everything that is not tierNever: restore uses it to find reusable local
// pieces, and the legacy fallback when no mode applies.
var syncScopeAll = contentScope{level: tierFull}

func (sc contentScope) keep(rel string) bool { return syncTier(rel) <= sc.level }

// carryLocalOnly moves, from the profile directory being replaced (old) into
// the freshly restored one (dst), every entry the cloud copy left out on
// purpose: caches, component data and whatever the uploading mode did not
// sync. Entries the manifest restored are never overwritten, and entries the
// uploading mode did sync are not carried (they were deleted remotely).
func carryLocalOnly(old, dst string, scope contentScope) {
	var walk func(rel string)
	walk = func(rel string) {
		ents, e := os.ReadDir(filepath.Join(old, filepath.FromSlash(rel)))
		if e != nil {
			return
		}
		for _, en := range ents {
			r := en.Name()
			if rel != "" {
				r = rel + "/" + r
			}
			if en.Type()&os.ModeSymlink != 0 {
				continue
			}
			low := strings.ToLower(en.Name())
			if !en.IsDir() && syncSkipFile(r, low) {
				continue // locks, launch metadata
			}
			from, to := filepath.Join(old, filepath.FromSlash(r)), filepath.Join(dst, filepath.FromSlash(r))
			if !scope.keep(r) {
				if _, e := os.Lstat(to); os.IsNotExist(e) {
					_ = os.MkdirAll(filepath.Dir(to), 0o700)
					_ = os.Rename(from, to)
				}
				continue
			}
			if en.IsDir() {
				walk(r)
			}
		}
	}
	walk("")
}

// SyncSizeEstimate is what one push of the profile would carry in each mode
// (plain bytes of the local files; the upload itself only sends changed
// pieces, compressed) and what no mode syncs.
type SyncSizeEstimate struct {
	Lite     int64  `json:"lite"`
	Standard int64  `json:"standard"`
	Full     int64  `json:"full"`
	Never    int64  `json:"never"` // caches / component data, never synced
	Mode     string `json:"mode"`  // the profile's effective mode
	Exists   bool   `json:"exists"`
}

// estimateSyncSizes walks the profile directory once (stat only).
func estimateSyncSizes(dir string) SyncSizeEstimate {
	var est SyncSizeEstimate
	if _, e := os.Stat(dir); e != nil {
		return est
	}
	est.Exists = true
	var walk func(abs, rel string, inherited int)
	walk = func(abs, rel string, inherited int) {
		ents, e := os.ReadDir(abs)
		if e != nil {
			return
		}
		for _, en := range ents {
			r := en.Name()
			if rel != "" {
				r = rel + "/" + r
			}
			if en.Type()&os.ModeSymlink != 0 {
				continue
			}
			t := syncTier(r)
			if inherited > t {
				t = inherited
			}
			if en.IsDir() {
				walk(filepath.Join(abs, en.Name()), r, t)
				continue
			}
			info, e := en.Info()
			if e != nil || !info.Mode().IsRegular() {
				continue
			}
			if syncSkipFile(r, strings.ToLower(en.Name())) {
				t = tierNever
			}
			n := info.Size()
			switch t {
			case tierLite:
				est.Lite += n
				est.Standard += n
				est.Full += n
			case tierStandard:
				est.Standard += n
				est.Full += n
			case tierFull:
				est.Full += n
			default:
				est.Never += n
			}
		}
	}
	walk(dir, "", tierLite)
	return est
}

// SyncSizeEstimate (UI): sizes per 同步内容 mode for a local profile.
func (s *Store) SyncSizeEstimate(id string) SyncSizeEstimate {
	p, ok := s.GetProfile(id)
	if !ok {
		return SyncSizeEstimate{Mode: syncScopeFor(s.GetSettings(), nil)}
	}
	est := estimateSyncSizes(s.ProfileDir(p.ID))
	est.Mode = syncScopeFor(s.GetSettings(), p)
	return est
}
