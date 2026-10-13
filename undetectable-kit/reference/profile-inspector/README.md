# profile-inspector

Standalone inspector and launcher for the Chromium build bundled with
Undetectable. It reads the profile database and the binary `startInfo`
fingerprint records, and starts `chrome.exe` with a profile's fingerprint
without running the Undetectable main application.

This is the piece a custom GUI launcher needs: everything the main app does
before the browser is useful is reproduced here, with no Qt dependency.

## Build

```
go build -o profile-inspector.exe .
```

Requires Go 1.21+ and network access on the first build (module cache).

## Commands

```
profile-inspector list                        list profiles in the database
profile-inspector dump   <profile-id>         startInfo slots + resolved fields
profile-inspector config <profile-id>         the fingerprint the database asks for
profile-inspector sync   <profile-id>         rewrite startInfo from the database
profile-inspector launch <profile-id>         start the fingerprint browser
profile-inspector verify <profile-id>         launch, read back, compare
```

Flags may appear before or after the subcommand.

| flag | meaning |
|---|---|
| `-port N` | DevTools port, `0` picks a free one |
| `-timeout 60s` | DevTools startup timeout |
| `-chrome PATH` | `chrome.exe` path |
| `-uddir PATH` | user data dir, defaults to the profile directory |
| `-proxy URL` | `--proxy-server` value |
| `-url URL` | page to open, defaults to `about:blank` |
| `-window-size WxH` | window size |
| `-maximized`, `-minimized`, `-headless` | window mode |
| `-no-cdp` | skip the UA / UA-CH overrides |
| `-detach` | `launch`: return immediately instead of supervising |
| `-log PATH` | append the browser's stdout/stderr to a file |
| `-json` | machine readable output (`list`, `dump`, `config`, `verify`) |
| `-out PATH` | `sync`: write elsewhere instead of in place |
| `-force` | `sync`: overwrite in place without writing `startInfo.bak` |

## What actually carries the fingerprint

There is one source of truth: `<profilesPath>/<profile-id>/startInfo`, a flat
stream of `<u64 little-endian length><payload>` records. The browser locates it
from `--profile-id`, not from `--user-data-dir`. A missing or malformed file
makes `chrome.exe` exit with `0xC0000409`.

The Undetectable main application does not hand the browser any fingerprint
arguments. It reads the profile row, serializes `startInfo`, and starts
`chrome.exe`. The database is therefore the configuration interface:

| file | contents |
|---|---|
| `%LOCALAPPDATA%\Undetectable\Profiles\storage` | SQLite, table `profiles` |
| `profiles.data` | JSON: UA, CPU cores, memory, timezone, languages, resolution, WebGL string, public RTC IP |
| `profiles.configMeta` | JSON: canvas noise, fonts, WebGL config, and `additional_info` |
| `configMeta.additional_info` | GPU adapter, media devices, `performance.jsHeapSizeLimit`, `speechSynthesis`, `storage`, `userAgentDataObj`, `webglParameters` |
| `%LOCALAPPDATA%\Undetectable\settings.json` | `profilesPath`, and the CDP port the main app serves |

`profile-inspector sync` reads those two JSON documents and writes the values
back into `startInfo`, keeping the existing file as the template so every field
that is not understood survives untouched.

### Field map

Two layouts exist. The older one (`layout A`, 358 slots) predates UA-CH; the
newer one (`layout B`, 372 slots) carries the client hint brand lists.

| field | layout A | layout B |
|---|---|---|
| name (UTF-16) | 0 | 0 |
| chrome version (UTF-16) | - | 2 |
| OS version `"11"` | 3 | 4 |
| packed CPU/memory/canvas record | 4 | 5 |
| GPU architecture | 5 | 6 |
| GPU vendor | 7 | 8 |
| user agent (two copies) | 17, 18 | 18, 19 |
| `navigator.platform` | 19 | 20 |
| `productSub` | 20 | 21 |
| vendor | 21 | 22 |
| UA-CH platform `"Windows"` | 24 | 25 |
| UA-CH platform version | 25 | 26 |
| architecture | 26 | 27 |
| bitness | 29 | 30 |
| `appVersion` | 31 | 32 |
| form factor | 33 | 34 |
| brand count, brands, full version list | - | 35..49 |
| public RTC IP | 36 | 50 |
| WebGL renderer | 43 | 57 |
| WebGL vendor | 44 | 58 |
| fonts list | 55..252 | 69..266 |
| timezone | 253 | 267 |
| language | 255 | 269 |
| speech voices | 257.. | 271.. |
| `zh-TW` anchor | 344 | 358 |

Scalar fields are packed into RAW runs. The screen block starts eight bytes
after an all-zero run that follows the `zh-TW` record:

```
+0   u32  screen.width
+4   u32  screen.height
+8   u32  screen.availWidth
+12  u32  screen.availHeight
+16  f64  devicePixelRatio
+28  u32  screen.colorDepth
+32  u32  screen.pixelDepth
+54  f64  canvas noise accumulator
+63  u64  navigator.storage.estimate() quota
+72  u64  performance.jsHeapSizeLimit
```

The packed CPU/memory record at slot 4 (layout A) or 5 (layout B):

```
+0   u8   1
+2   u32  navigator.deviceMemory (GB)
+6   u32  navigator.hardwareConcurrency
+11  u32  CanvasWebglRandomParameter
+15  u32  CanvasWebglRandomParameter2
+19  u32  pixel1
+23  u32  pixel2
```

## What startInfo cannot do

`startInfo` has no effect on the user agent client hints of a layout A profile:
the brand lists are empty and `uaFullVersion` is blank. The Undetectable main
app fills them at runtime over DevTools, not through the file. `launch` and
`verify` do the same:

1. `Target.setAutoAttach` with `flatten` on the browser endpoint, so every page
   target is attached before its first script runs.
2. `Emulation.setUserAgentOverride` with `userAgentMetadata` on each page
   session, which sets `navigator.userAgent` and `navigator.userAgentData`
   including the high entropy values.
3. `Emulation.setLocaleOverride` for the profile language.

This is also why a one-shot override on a single page is not enough: a tab
opened later gets a fresh session with the default hints.

The browser's own `Browser.und_updateProfileData` and
`Browser.und_getProfileId` commands exist in `chrome.dll`, but
`und_updateProfileData` does not set the brand lists on a layout A profile, so
the launcher uses the standard Emulation domain instead.

## Layout of the code

| file | responsibility |
|---|---|
| `main.go` | subcommands, `sync` writer, `verify` comparison |
| `startinfo.go` | record parser/serializer, layout resolver, scalar patch |
| `db.go` | SQLite reader, `data` / `configMeta` decoding, expectations |
| `launch.go` | command line, process start, CDP override watcher, fingerprint probe |
| `cdp.go` | minimal DevTools Protocol client with event dispatch |

## Notes

- `sync` writes `startInfo.bak` next to the original unless `-force` is used.
- `verify` opens `https://example.com/` because `navigator.userAgentData` and
  `navigator.deviceMemory` only exist in a secure context.
- `launch` supervises the browser by default so the override watcher stays
  connected; a tab opened later still receives the fingerprint. Press Ctrl-C to
  stop watching without closing the browser. `-detach` skips supervision and is
  the right mode for a GUI that owns the process itself.
- `launch` and `verify` work for profile directories that have no row in the
  database. `verify` then compares the live browser against the values read from
  `startInfo` instead of the database.
- The bundled `chrome.exe` is `Chrome/153.0.8010.37`.
