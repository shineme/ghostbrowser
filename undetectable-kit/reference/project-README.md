# GhostBrowser

## Undetectable 原生内核与 JSON 启动

完整运行文件、10 条指纹快照、3 个二进制模板及独立 Go 启动器存放在 `undetectable-kit/`。
详细使用及字段支持范围见 [中文文档](undetectable-kit/README.zh-CN.md)。

GUI 设置中选择「使用项目自带 Chromium 与指纹库」，环境列表提供「指纹库 / JSON」导入、导出、创建与实际浏览器验证。
原生模式直接启动 `chrome.exe`，通过二进制 `startInfo` 与持续 CDP 连接配置指纹。
命令行：`undetectable-kit/bin/ghost-launcher.exe -config undetectable-kit/examples/native-profile.json`。

原有 `custom` 参数模式需要浏览器自行实现参数解析；本机 Undetectable Chromium 应使用新增的 `undetectable` 模式。

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

生成文件为 `build/bin/GhostBrowser.exe`。发布时将整个 `undetectable-kit` 文件夹与 EXE 放在一起；设置页默认优先使用该包内的 `runtime/chrome/chrome.exe`，用户也可以点「选择 EXE」指定其它浏览器程序，再点「保存设置」保存路径。定制指纹模式需要与模板兼容的 Undetectable 定制 Chromium；标准 Chrome/Edge 只适用于 stock 模式。

云同步 PHP 服务端不参与桌面 GUI 编译。宝塔 Nginx 部署步骤见 [PHP 云同步部署文档](cloud-server/README.zh-CN.md)，可直接使用 `cloud-server/deploy/baota-nginx.conf.example` 作为站点配置片段。

## Features
- Profiles: create / bulk create / clone / delete, groups, tags, start URLs, cookie import (CDP Storage.setCookies)
- Fingerprint: OS-consistent generator (UA, platform, screen, DPR, cores, memory, WebGL vendor/renderer, fonts), canvas / WebGL / audio / ClientRects noise by seed, timezone / language / geolocation from proxy IP, WebRTC, media devices, DNT
- Kernel modes: stock Chrome + generated MV3 extension (MAIN world), or custom patched Chromium via --fingerprint-config=<profile>/fingerprint.json
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
