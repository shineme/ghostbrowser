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
- 开源内核版本兼容：支持 Chrome/Chromium 110 及以上（推荐 120+，已在 110–155 与 ungoogled-chromium 上实测），自动识别内核版本（版本目录、chrome.exe 版本信息或 --version），UA / UA-CH 大版本始终跟随内核；设置页对过旧内核、品牌版 Chrome 137+（不支持命令行加载扩展）和 Edge 给出中文提示，低于 110 拒绝启动
- 新建、批量创建、复制、删除；分组、标签、启动网址、导入 Cookie
- 扩展中心：按商店 ID / 应用商店链接、CRX 链接或本地 .crx/.zip 添加扩展（保留原扩展 ID），分配给环境后启动时自动加载（两种内核均支持）；Cookie 支持 JSON 与 Netscape cookies.txt 导入导出，运行中直接写入浏览器，未运行时暂存到下次启动；可连接本地 Clash / Mihomo 控制器，查看策略组与节点、切换节点、测延迟，并一键生成指向 Mihomo 端口的代理
  - 界面：「扩展」页以卡片展示图标、版本、MV2/MV3、来源和使用环境数，可搜索、按来源筛选、删除（自动从环境中移除），并通过「分配到环境」弹窗批量勾选环境；环境编辑抽屉里也可逐个开关扩展
  - 界面：环境抽屉 →「Cookies · 管理」打开导入/导出弹窗：自动识别 JSON / Netscape，支持选择文件、拖入、剪贴板，导出可复制或下载；顶部显示环境是否运行（立即写入 / 下次启动写入、快照时间、待写入条数）
  - 界面：「代理」页工具栏「Clash / Mihomo」：填写控制器地址与密钥连接，显示版本、模式、监听端口；按策略组切换节点、单个或全部测速，一键「添加为代理」。注意：所有使用同一监听端口的环境共享当前节点
- 列表可按分组和标签筛选，按最近运行、名称或创建时间排序；搜索支持名称、分组、标签和出口 IP
- 批量操作（勾选后底部出现操作栏）：启动、停止、改分组或加标签、分配代理、立即同步、删除
- 「运行中」条显示每个环境的运行时长和出口 IP，可以一键把浏览器窗口切到前台，或直接停止
- 排列窗口：「运行中」条右侧一键把运行中的窗口按网格 / 层叠 / 重叠排列，弹窗里实时预览布局（按屏幕比例绘制，格子内显示环境名），可选全部运行中或仅已勾选的环境，失败的窗口会列在提示里
- 多窗口同步器（群控）：对话框里选一个主控窗口和多个受控窗口；同步点击 / 输入 / 按键 / 滚动 / 标签页可分别开关（同步中修改立即生效）；可设随机操作延迟区间（毫秒），每个受控窗口对每个操作独立随机等待，模拟真人节奏；标签页同步：主控新建、切换、关闭标签页会在受控窗口镜像执行，主控与受控标签页一一对应，操作落到对应的标签页（不再只限第一个标签页）；一键网格平铺、重叠排列、统一大小（受控窗口改成主控窗口大小，位置不变）、显示窗口（还原最小化并前置）、统一标签页（受控窗口打开与主控相同的网址、相同顺序和当前标签页）、关闭当前 / 其他 / 空白标签；批量访问网址（自动补全 https://，可选新标签页）。窗口和标签页操作不需要先开始同步；实时显示同步事件数、每个受控窗口的延迟、标签页数和错误；选项与所选窗口自动保存。iframe（同源与跨域）内的点击、输入、滚动和下拉选择同步到受控窗口的对应 iframe（按 iframe 路径匹配，找不到时按坐标换算）；输入法候选与文件选择框不同步
- 存储列显示同步状态：同步中、待上传、已同步（含云端版本号）、仅本地
- 多个环境共用同一个出口 IP 时会标黄提醒，代理失效时会标红
- 接口与审计页：MCP 服务一键启停（可改端口，显示运行状态、调用次数、最近客户端和最近调用，端口被占用时给出原因）；本地 API 地址与 Key（默认隐藏，可显示 / 复制）；按 Cursor（HTTP）、Claude Desktop（stdio）、Claude Code（命令行）一键复制接入配置；列出全部 MCP 工具并区分只读和会修改数据的工具
- 操作日志：界面、本地 API、MCP 和云同步的每次操作（启动 / 停止 / 新建 / 编辑 / 删除 / 克隆环境、分配代理、分组、同步、MCP 调用）都会记录，可按分类、来源、环境筛选，搜索，只看失败，每 5 秒自动刷新，并按当前筛选导出 CSV / JSON 或清空

### 指纹
- 按操作系统生成相互匹配的 UA、平台、屏幕、DPR、CPU 核数、内存、WebGL、字体
- Canvas、WebGL、Audio、ClientRects 噪声按种子固定
- 时区、语言、地理位置可以跟随代理出口 IP；启动前会重新检测出口
- 内置设备库 + 自定义指纹库（`library.json`），下拉框联动并支持保存自定义值
- **防护与启动选项**（环境编辑「防护」「启动」页，开源内核生效）：Canvas / WebGL 图像 / AudioContext / ClientRects 各自可选 噪声、真实、关闭；WebRTC、媒体设备（可自定义麦克风/扬声器/摄像头数量）、语音列表、WebGPU、Do Not Track；端口扫描保护（白名单端口）、每次启动随机指纹；窗口大小（自动/自定义宽高位置/最大化）、硬件加速、禁止图片/视频、静音、自定义启动参数（user-data-dir、remote-debugging、proxy-server、user-agent 等由程序接管）。定制内核下会提示哪些项不生效
- **指纹附加项**（开源内核）：媒体设备 ID 别名——媒体设备选「真实」时可打开「设备 ID 别名」，网站看到的 deviceId 按环境种子 + 网站来源生成（同一环境同一网站永远一致，换网站 / 换种子就不同），getUserMedia、轨道 getSettings、setSinkId 等自动换回真实 ID，摄像头和麦克风照常可用；未授权时为空的 ID 保持为空；groupId 与 Chrome 一样每次加载重新加盐。WebGPU 关闭会移除 navigator.gpu 并加 --disable-blink-features=WebGPU；语音列表可关闭。环境编辑「防护」页可「重新生成指纹种子」（确认后立即保存，只换噪声种子，Cookie、登录状态和其他参数不变），并显示种子短 ID（前 8 位十六进制）
- **终端**（侧栏「终端」）：一键打开 Claude Code（`claude`）、Codex CLI（`codex`）、Gemini CLI（`gemini`）或 Windows PowerShell 纯终端；自动检测 PATH 中是否已安装并显示版本，未安装时给出安装命令和官方安装指南。可选工作目录（文件夹选择或手动输入，留空为用户目录），可继承某个环境的时区（TZ）、语言（LANG / LC_ALL）和代理（HTTP_PROXY / HTTPS_PROXY / ALL_PROXY），也可自定义时区和语言；带账号密码或 SOCKS 的代理经本地中转变成 `http://127.0.0.1` 代理，中转在终端关闭时一起停止。新窗口可选 PowerShell 控制台或 Windows Terminal（已安装时）；记住上次的选择，列出从这里打开的终端并可关闭

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
  - **一键安装开源内核**：设置 → 开源内核 →「一键安装」，自动下载 Chromium 官方 Windows x64 快照（`chrome-win.zip`）到 `%APPDATA%\GhostBrowser\kernels\chromium-<版本>\` 并设为默认开源内核。下载源可选「国内镜像」（npmmirror，默认）、「官方源」（Google 快照服务器）或「自定义地址」（快照目录或 .zip 直链，如 ungoogled-chromium 便携版）。支持断点续传、取消、自动/手动重试、磁盘空间检查、防 zip-slip 解压、校验 chrome.exe 并读取版本；进度条显示 解析版本/下载/解压/校验/完成、速度和剩余时间。已安装的内核可切换使用或删除。启动开源内核环境时若找不到内核，会直接弹出安装窗口，环境抽屉里也有提示
- 批量创建也可以选择设备类型

### 自动化
- 「自动化」页可视化编排流程：打开网址、点击、输入、等待元素、延时（可随机）、滚动、提取文本、执行 JS、截图、上传文件、切换/关闭标签页，以及可嵌套的条件判断（则 / 否则）和循环（按次数或按条件、跳出循环）；选择器支持 CSS、`xpath:` 和 `text:`
- 每个步骤可停用、设置「出错继续」；提取的值存为变量，后续步骤用 `{{变量}}` 引用，编辑器会列出当前可用变量并实时校验，有问题的步骤直接标红
- 选择多个环境批量运行，可设并发数（1/3/5/10）、自动启动未打开的环境、结束后关闭；运行抽屉实时显示每个环境的步骤进度、耗时日志、提取到的变量和截图，可随时停止，历史记录可回看
- 流程可复制、导出为 JSON 文件或导入（文件或粘贴），空白页提供一键「新建示例流程」
- RPA iframe 支持：点击、输入、等待、提取文本、滚动到元素、上传、执行 JS、元素截图等步骤都可填写「所在框架」（如 `iframe#pay >> iframe[name=card]`，多层用 >> 连接，可用 {{变量}}），同源、跨域（独立进程）和嵌套 iframe 都能用，鼠标点击按 iframe 位置换算成真实坐标；新增「切换框架」步骤（进入框架 / 上一级 / 返回主页面），之后的步骤都在该框架里执行；会等待 iframe 出现和加载。辅助连接自动沿用指纹的 UA 覆盖，跨域 iframe 内的 navigator / UA / 时区与主页面一致
- 操作录制：点「录制」选择环境（未打开会自动启动），在浏览器里正常操作即可自动生成步骤：地址栏输入网址、点击/双击、输入（连续按键合并为一次输入）、下拉选择、勾选、回车提交、上传文件、新开/切换/关闭标签页，链接或表单引起的跳转不会重复记录；新出现的元素自动先加「等待元素」。选择器优先用 id、data-testid、name、aria-label、按钮文字，并附带备用 XPath。录制面板实时显示步骤（可编辑、删除、暂停），浏览器右下角有录制浮窗（可关闭），还能点选元素添加「提取文本」「等待元素」；停止后可选择保留停顿时间、合并输入、密码转为 `{{password}}` 变量，再生成新流程或插入到当前步骤之后。录制脚本运行在隔离环境里，网页脚本看不到（不启用 Runtime 调试域）；iframe 内的操作（同源、跨域、嵌套、后加载的 iframe）也会录制，步骤自动带上「所在框架」路径；暂不支持封闭 Shadow DOM 内部和 canvas 绘制的界面

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
- **删除环境可选「只删本地」或「同时删云端」**（单个和批量删除共用一个确认框，记住上次的选择；未配置同步时不显示选项）：
  - 只删本地：删除本机的环境和数据目录，云端副本保留且**不会再被自动下载回来**；「云同步 → 仅云端」列出这些环境（标记「仅云端（已在本机删除）」，其他设备上传了新版本时显示「云端有更新」），可「恢复到本机」或「删除云端」
  - 同时删云端：服务器删除该环境的数据、全部历史版本和增量块引用，只留一条不含数据的删除记录（墓碑，名称 / 时间 / 设备，默认保留 90 天）。其他设备在启动对比、立即同步、启动该环境前会看到墓碑并自动删除本机副本（提示「环境 X 已在 某设备 删除，已同步删除本机副本」）；正在运行的会在浏览器关闭后再删除；旧设备的上传会被服务器拒绝（HTTP 410），不会把环境“复活”
  - 服务器连不上时先删除本机副本，云端删除排队保存，联网后（启动、立即同步或每分钟）自动重试；另一台设备正在上传该环境时服务器返回 423，可选择强制删除
  - 两种删除都会写入操作日志（删除环境、同步 · 删除云端、同步 · 删除本机副本）。需要把新版 `ghostsync.php` 上传到服务器，旧版服务端删除后其他设备不会自动删除
- 显示服务器信息：已用空间、磁盘剩余、单次上传上限、保留的版本数
- 同步地址使用 http 时会提示：令牌会明文传输，建议改用 https
- 旧版服务端（不支持增量）会自动改用整包同步

### 界面
- 左侧边栏可收起为图标栏（侧栏底部按钮或 Ctrl+B），悬停显示名称和快捷键，数量变为圆点；状态会记住，下次启动保持；系统开启「减少动态效果」时不播放动画

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
    GET  /api/profiles[?tombstones=1]       tombstones=1 also returns deleted profiles: {id,name,version,deleted:true,deletedAt,deletedBy,deletedByName}
    POST /api/profiles/{id}/lock            {"device": "...", "restore": false}  -> lockToken; 410 for a tombstone unless restore:true
    PUT  /api/profiles/{id}/refs            newline-separated chunk IDs of this upload (under lease)
    PUT  /api/profiles/{id}/blob?version=N  encrypted manifest (GBE1); 412 if server version != N
    POST /api/profiles/{id}/commit          {id,name,version,sha256,meta}
    POST /api/profiles/{id}/unlock
    GET  /api/profiles/{id}/blob[?version=N]
    GET  /api/profiles/{id}/history
    POST /api/profiles/{id}/delete          {"force": false} -> {ok, state: deleted|already|absent, version}; 423 if another device holds the lease
    POST /api/chunks/missing                {"ids": [...]} -> {"missing": [...]}
    PUT  /api/chunks                        framed: (64-hex id, uint32 BE length, GBC1 data)*
    POST /api/chunks/get                    {"ids": [...]} -> framed, same layout
All requests carry Authorization: Bearer <token> (and X-Ghost-Token for hosts that strip it).
Upload, refs, commit and unlock also send X-Device-ID and X-Lock-Token.
Deletion leaves a tombstone: the index entry keeps id/name, its version is bumped, data/history/refs are released (blob and chunk GC), and
deleted/deletedAt/deletedBy are set. The plain list (old clients) omits tombstones; ping lists the "tombstones" feature and tombstoneDays.
Blob/history GET, lock and commit on a tombstone return 410, so a stale device cannot resurrect it; an explicit lock {"restore": true}
re-creates it. Tombstones are purged after GHOST_TOMBSTONE_DAYS (default 90, env GHOST_SYNC_TOMBSTONE_DAYS).
Chunk IDs are HMAC-SHA256(per-profile chunk key, plaintext); chunk bodies are "GBC1" + nonce + AES-256-GCM(deflate(data)). The chunk key lives only inside the encrypted manifest.
