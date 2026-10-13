# GhostBrowser

## Undetectable 原生内核与 JSON 启动

完整运行文件、10 条指纹快照、3 个二进制模板及独立 Go 启动器存放在 `undetectable-kit/`。
详细使用及字段支持范围见 [中文文档](undetectable-kit/README.zh-CN.md)。

GitHub 源码仓库不包含 `undetectable-kit/runtime/`、`bin/` 下的生成程序和浏览器运行二进制。缺失部分由首次启动时的自动安装补齐（见「环境自动安装」）。

GUI 设置中只保留「定制 Chromium」一种内核。环境列表提供「指纹库 / JSON」导入、导出、创建与实际浏览器验证。
原生模式直接启动 `chrome.exe`，通过二进制 `startInfo` 与持续 CDP 连接配置指纹。
命令行启动器：`undetectable-kit/bin/ghost-launcher.exe -config undetectable-kit/examples/native-profile.json`。

Go + Wails v2 fingerprint browser manager. UI effects from Libraries.dev (border-beam, thinking-orbs, metal-fx).

## Build

Windows 图形界面需要 Go（版本不低于 `go.mod` 的要求）、Node.js/npm 和 Wails v2。安装一次 Wails CLI：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
```

在项目根目录编译：

```powershell
Set-Location 'C:\Users\dhl\Desktop\ghostbrowser'
go mod download
wails build -platform windows/amd64
```

Wails 会按 `wails.json` 调用前端安装和构建命令；如果提示找不到 `node` 或 `npm`，先把 Node.js 安装目录加入当前 PowerShell 的 `PATH`。

生成文件为 `build/bin/GhostBrowser.exe`。**只需分发这一个 EXE**：首次启动会自动检测环境，缺失时从服务器下载定制 Chromium 与指纹包并校验、解压（详见下方「环境自动安装」）。若 EXE 同目录已存在完整的 `undetectable-kit`，则优先就地使用，不再下载。

## 环境自动安装

| 项目 | 值 |
|---|---|
| 定制 Chromium | `http://shop.chestchain.com/chrome.zip`（约 213 MB，解压后是顶层 `chrome/`） |
| 指纹包 | `http://shop.chestchain.com/undetectable-kit.zip`（顶层 `undetectable-kit/fingerprints/`） |
| 校验清单 | `http://shop.chestchain.com/manifest.json`（可选，见下） |

安装位置按优先级：EXE 同目录的 `undetectable-kit/` → `%APPDATA%\GhostBrowser\undetectable-kit\`。
`chrome.zip` 会被重映射到 `<kit>/runtime/chrome/`，因为内核与指纹库都按该布局查找；直接盲解压会导致找不到 `chrome.exe`。

下载会读取 `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY`，并支持 `socks5://` 与 `socks5h://`。

### 发布前必须做的两件事

1. **上传 `manifest.json`**。当前服务器上没有这个文件（返回 404），所以只能用 EXE 内置的校验值兜底。内置值对应现在的两个包；一旦你在服务器上替换包而没有改内置值，客户端会判定校验失败并拒绝安装。清单格式：

```json
{
  "updated": "2026-10-14",
  "chrome":    { "sha256": "<64位小写十六进制>", "size": 222926646 },
  "fpkit":     { "sha256": "<64位小写十六进制>", "size": 187537 }
}
```

   有清单时以清单为准，换包无需重新编译 EXE。生成命令（在包所在目录）：

```powershell
Get-FileHash .\chrome.zip,.\undetectable-kit.zip -Algorithm SHA256 | ForEach-Object { "{0}  {1}" -f $_.Hash.ToLower(), $_.Path }
```

2. **改用 HTTPS**。`shop.chestchain.com` 的 80 端口不监听，443 提供的是不受信证书，因此明文 http 下载存在被中间人替换包的风险——而这里下载的是一个可执行浏览器内核。建议在服务器上部署受信证书，并把 `envsetup.go` 顶部的 `chromeZipURL` / `fpKitZipURL` 改为 `https://`。sha256 校验能发现内容被篡改，但无法抵御主动攻击者同时替换包与哈希的场景，HTTPS 才是根治手段。

更新包后若要同步内置兜底值，修改 `envsetup.go` 里的 `builtInSHA`。

验证整条链路（真实下载 + 启动内核 + 指纹生效），需先把包放到 `%TEMP%\gbprobe\`：

```powershell
$env:ALL_PROXY='socks5://127.0.0.1:2080'; $env:GB_LIVE_SHOP='1'
go test -run TestEnsureAgainstRealShop -v .
```

云同步 PHP 服务端不参与桌面 GUI 编译。宝塔 Nginx 部署步骤见 [PHP 云同步部署文档](cloud-server/README.zh-CN.md)，可直接使用 `cloud-server/deploy/baota-nginx.conf.example` 作为站点配置片段。

## Features
- Profiles: create / bulk create / clone / delete, groups, tags, start URLs, cookie import (CDP Storage.setCookies)
- Fingerprint: OS-consistent generator (UA, platform, screen, DPR, cores, memory, WebGL vendor/renderer, fonts), canvas / WebGL / audio / ClientRects noise by seed, timezone / language / geolocation from proxy IP, WebRTC, media devices, DNT
- Kernel: Undetectable 定制 Chromium only. Fingerprints are applied natively from the binary `startInfo` template; standard Chrome/Edge cannot run the templates.
- Proxies: bulk import (host:port[:user:pass], user:pass@host:port, scheme://), check exit IP / geo / latency, proxy auth via extension
- Cloud sync client: zip (caches excluded) -> AES-256-GCM (PBKDF2 key from local sync password) -> PHP server on your VPS
- Local API on 127.0.0.1:25325: /list, /profile/start?profile=ID (returns CDP port + websocket), /profile/stop?profile=ID, /status

## 云同步 PHP 服务端（已提供）

源码在 `cloud-server/`，部署方式见 [云同步部署文档](cloud-server/README.zh-CN.md)。
新版客户端检查锁定、上传、提交结果，将完整 Profile JSON 放入加密包，并在拉取时恢复元数据。
交付包 `undetectable-kit/cloud-server/` 同步附带相同服务端文件。

## Sync server contract
    GET  /api/ping
    GET  /api/profiles
    POST /api/profiles/{id}/lock     {"device": "..."}
    PUT  /api/profiles/{id}/blob?version=N   (412 if server version != N)
    GET  /api/profiles/{id}/blob
    POST /api/profiles/{id}/commit   {id,name,version,sha256,meta}
    POST /api/profiles/{id}/unlock
All requests carry Authorization: Bearer <token>.
The lock response returns lockToken; upload/commit/unlock also send X-Device-ID and X-Lock-Token.
