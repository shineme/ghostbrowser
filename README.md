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

云同步 PHP 服务端不参与桌面 GUI 编译。宝塔 Nginx 部署步骤见 [PHP 云同步部署文档](cloud-server/README.zh-CN.md)。服务端只有一个文件 `cloud-server/ghostsync.php`，不需要数据库，也不需要伪静态。

## 功能

### 环境
- 新建、批量创建、复制、删除；分组、标签、启动网址、导入 Cookie
- 扩展中心：按商店 ID / 应用商店链接、CRX 链接或本地 .crx/.zip 添加扩展（保留原扩展 ID），分配给环境后启动时自动加载（两种内核均支持）；Cookie 支持 JSON 与 Netscape cookies.txt 导入导出，运行中直接写入浏览器，未运行时暂存到下次启动；可连接本地 Clash / Mihomo 控制器，查看策略组与节点、切换节点、测延迟，并一键生成指向 Mihomo 端口的代理
  - 界面：「扩展」页以卡片展示图标、版本、MV2/MV3、来源和使用环境数，可搜索、按来源筛选、删除（自动从环境中移除），并通过「分配到环境」弹窗批量勾选环境；环境编辑抽屉里也可逐个开关扩展
  - 界面：环境抽屉 →「Cookies · 管理」打开导入/导出弹窗：自动识别 JSON / Netscape，支持选择文件、拖入、剪贴板，导出可复制或下载；顶部显示环境是否运行（立即写入 / 下次启动写入、快照时间、待写入条数）
  - 界面：「代理」页工具栏「Clash / Mihomo」：填写控制器地址与密钥连接，显示版本、模式、监听端口；按策略组切换节点、单个或全部测速，一键「添加为代理」。注意：所有使用同一监听端口的环境共享当前节点
- 列表可按分组和标签筛选，按最近运行、名称或创建时间排序；搜索支持名称、分组、标签和出口 IP
- 批量操作（勾选后底部出现操作栏）：启动、停止、改分组或加标签、分配代理、立即同步、删除
- 「运行中」条显示每个环境的运行时长和出口 IP，可以一键把浏览器窗口切到前台，或直接停止
- 存储列显示同步状态：同步中、待上传、已同步（含云端版本号）、仅本地
- 多个环境共用同一个出口 IP 时会标黄提醒，代理失效时会标红

### 指纹
- 按操作系统生成相互匹配的 UA、平台、屏幕、DPR、CPU 核数、内存、WebGL、字体
- Canvas、WebGL、Audio、ClientRects 噪声按种子固定
- 时区、语言、地理位置可以跟随代理出口 IP；启动前会重新检测出口
- 内置设备库 + 自定义指纹库（`library.json`），下拉框联动并支持保存自定义值

### 设备类型与双内核
- 设备类型：台式机、笔记本、安卓手机、安卓平板、折叠屏、Chromebook、电视。设备库里有对应的真实机型（Galaxy S24/S23/A 系列、Pixel 6a~9 Pro、小米/Redmi/POCO、OnePlus、OPPO、vivo、Motorola、Sony；Galaxy Tab、Pixel Tablet、小米平板、联想、OnePlus Pad；Z Fold6 折叠/展开、Z Flip6、Pixel 9 Pro Fold；Acer/HP/ASUS/三星/联想 Chromebook；Chromecast、BRAVIA、Tizen、webOS 电视）
- 选择设备类型后，系统、机型、UA（手机带 Mobile，平板不带）、平台、分辨率与 DPR、GPU、CPU、内存、触控点、字体全部联动
- **两个内核**：Windows 台式机和笔记本可选「定制内核」（Undetectable）或「开源内核」；手机、平板、折叠屏、Chromebook、电视、macOS、Linux 只能用开源内核
- **开源内核**（设置 → 开源内核）：Chromium、Chrome for Testing、ungoogled-chromium、Chrome、Edge 或自己编译的 Chromium，留空自动检测
  - 指纹通过 CDP 注入，不用扩展（Chrome 137 起正式版禁用了 `--load-extension`）：Chrome 开发者工具同款的设备模拟（UA 与 UA-CH 含 model/mobile/platformVersion、视口、DPR、触控、屏幕方向）、时区、语言、地理位置、CPU 核数；页面脚本在文档开始前补上 WebGL、内存、屏幕（桌面）、噪声、字体、电池、WebRTC
  - UA-CH 的品牌列表直接读内核自身，UA 大版本自动跟随内核版本（电视除外）
  - 新标签页、iframe 和 worker 自动接管；需要保持 GhostBrowser 运行
  - 不支持 iPhone / iPad：iOS 上的浏览器都是 WebKit，Chromium 无法可信地伪装
  - 已知限制：Worker 里的部分属性、真实系统字体测量、电视浏览器的老内核特性无法完全对应
- 批量创建也可以选择设备类型

### 代理
- 批量导入：粘贴或拖入 txt/csv/json，支持常见格式、协议前缀、换 IP 链接、`# 名称` 备注，自动去重
- 检测出口 IP、国家/地区、ISP、IP 类型（住宅/机房/移动）和延迟，支持测速
- 编辑代理：名称、标签、类型、认证、换 IP 链接
- 「换 IP」：调用服务商的换 IP 链接，然后重新检测，确认出口确实已经变了
- 可设置「每次启动环境前自动换 IP」（换 IP 失败不会阻止启动）
- 后台定时检测全部代理（设置 → 代理自动检测间隔，0 = 关闭），失效时显示横幅并列出受影响的环境
- 提醒多个代理出口相同、多个环境共用同一个代理
- 批量分配给环境：顺序轮流、随机（数量够时不重复）、统一用一个；可只选「可用」或「未被占用」的代理

### 云同步
- 客户端加密：AES-256-GCM，密钥由本机的同步密码派生，服务器只存密文
- **增量同步**：每个文件切成最大 4 MB 的加密块，只上传服务器上没有的块；拉取时本地已有的块直接复用。首次同步后，关闭浏览器通常只传几 MB。单个环境最大 2 GB（不含缓存）
- 自动同步：启动时以服务器为准对比；启动环境前拉取云端新版本；关闭浏览器后上传（失败自动重试）；退出软件前等待上传完成
- 云端版本较新时弹窗倒计时询问，默认动作可在设置里改
- 全局数据同步：设置、代理列表（含凭据、标签、换 IP 链接）、自定义指纹库
- **历史版本**：「云同步 → 历史版本」可以恢复服务器保留的任意版本，恢复后会作为最新版本上传，其他电脑也会同步到这个版本
- **同步日志**：最近 300 次同步的时间、流量、耗时和错误，失败的可以一键重试
- 显示服务器信息：已用空间、磁盘剩余、单次上传上限、保留的版本数
- 同步地址使用 http 时会提示：令牌会明文传输，建议改用 https
- 旧版服务端（不支持增量）会自动改用整包同步

### 本地自动化 API
- 地址 `127.0.0.1:25325`：`/list`、`/profile/start?profile=ID`（返回 CDP 端口和 websocket 地址）、`/profile/stop?profile=ID`、`/status`
- 除 `/status` 外，每个请求都要带 API Key：请求头 `X-API-Key: <key>`，或网址参数 `?key=<key>`。Key 在「设置 → 数据与本地 API」中复制或重新生成
- 网页无法调用：要求 Host 是本机地址，带 Origin 或跨站 `Sec-Fetch-Site` 的请求一律拒绝
- 设置里可以关闭「要求 API Key」（不建议）

```bash
curl -H "X-API-Key: gb_xxx" "http://127.0.0.1:25325/profile/start?profile=<环境ID>"
```

## 云同步 PHP 服务端（已提供）

单文件 `cloud-server/ghostsync.php`（PHP 7.4+，无需数据库和伪静态），部署方式见 [云同步部署文档](cloud-server/README.zh-CN.md)。`undetectable-kit/cloud-server/` 附带相同的文件。

- 多用户：在 `GHOST_USERS` 里加 `'令牌' => '名字'`，每个用户的数据目录相互独立
- 防暴力破解：同一个 IP 15 分钟内输错令牌 10 次，封禁 15 分钟（`GHOST_MAX_FAILS`）
- 版本保留：默认 3 个（`GHOST_KEEP_VERSIONS`，最多 50）；不再被引用的数据块 2 天后自动清理
- 每次请求最大约 12 MB，PHP 默认的 `post_max_size = 8M` 也能用（客户端会自动按服务器上限缩小批次）
- 上传/拉取有实时进度：右下角传输面板显示阶段（扫描、上传、提交、下载、还原）、已传/总量、实时速度与速度曲线、预计剩余时间，以及增量同步省下的流量；环境列表和云同步列表的状态列也会变成进度条（事件 `sync:progress`，绑定 `Transfers()`）
- 前面有 nginx / CDN 时，它们的上传上限 PHP 看不到：收到 HTTP 413 后客户端会自动对半拆包重试，并记住更小的批次。单个分块约 4 MB，所以 nginx 至少要设 `client_max_body_size 8m;`，建议 `64m`（nginx 默认只有 1m）

## Sync server contract
    GET  /api/ping                          features, keepVersions, maxBody, usedBytes, freeBytes
    GET  /api/profiles
    POST /api/profiles/{id}/lock            {"device": "..."}  -> lockToken
    PUT  /api/profiles/{id}/refs            newline-separated chunk IDs of this upload (under lease)
    PUT  /api/profiles/{id}/blob?version=N  encrypted manifest (GBE1); 412 if server version != N
    POST /api/profiles/{id}/commit          {id,name,version,sha256,meta}
    POST /api/profiles/{id}/unlock
    GET  /api/profiles/{id}/blob[?version=N]
    GET  /api/profiles/{id}/history
    POST /api/profiles/{id}/delete
    POST /api/chunks/missing                {"ids": [...]} -> {"missing": [...]}
    PUT  /api/chunks                        framed: (64-hex id, uint32 BE length, GBC1 data)*
    POST /api/chunks/get                    {"ids": [...]} -> framed, same layout
All requests carry Authorization: Bearer <token> (and X-Ghost-Token for hosts that strip it).
Upload, refs, commit and unlock also send X-Device-ID and X-Lock-Token.
Chunk IDs are HMAC-SHA256(per-profile chunk key, plaintext); chunk bodies are "GBC1" + nonce + AES-256-GCM(deflate(data)). The chunk key lives only inside the encrypted manifest.
