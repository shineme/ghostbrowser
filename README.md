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

- 应用图标（v0.8.5）：Spectral 渐变幽灵 + 指纹纹路，透明背景；EXE、托盘、侧栏和启动画面统一使用（源文件 `frontend/src/assets/images/ghost-mark.svg`）。

### 环境
- 开源内核版本兼容：支持 Chrome/Chromium 110 及以上（推荐 120+，已在 110–155 与 ungoogled-chromium 上实测），自动识别内核版本（版本目录、chrome.exe 版本信息或 --version），UA / UA-CH 大版本始终跟随内核；设置页对过旧内核、品牌版 Chrome 137+（不支持命令行加载扩展）和 Edge 给出中文提示，低于 110 拒绝启动
- 新建、批量创建、复制、删除；分组、标签、启动网址、导入 Cookie
- Firefox 内核（可选，与定制内核、开源内核并存）：环境抽屉「指纹配置 → 内核」或批量创建时选择 Firefox。使用本机已安装的 Mozilla Firefox（自动检测 Program Files 与注册表 App Paths，也可在「设置 → Firefox 内核」手动选择 firefox.exe）；没有安装时可打开 Mozilla 官方下载页，或一键安装官方 ESR / 正式版（从 Mozilla 官方 HTTPS 地址下载，按官方 SHA512SUMS 校验后静默安装到程序数据目录，不随程序打包）。每个环境独立 Firefox 配置目录（-profile -no-remote），启动前写入 user.js：代理（http / socks5；带账号密码的代理经本地中转，密码不写入文件）、语言（intl.accept_languages / intl.locale.requested）、时区（TZ 环境变量）、WebRTC（代理 IP / 真实 / 禁用）、DNT 与 GPC、固定地理坐标、WebGL / WebGPU 关闭、字体可见范围、严格防指纹模式（resistFingerprinting，与「语言 / 时区跟随 IP」互斥）、隐藏语音列表、CPU 核心数上限、禁止图片、静音、启动页和窗口大小；可选覆盖 UA（会提示可能不一致）。通过 WebDriver BiDi 控制：切到前台、正常关闭、退出检测、Cookie 实时导入导出，以及与 Chromium 相同格式的会话快照（换电脑后自动恢复登录；开源内核写下的完整会话快照含 LocalStorage / IndexedDB，Firefox 只恢复和更新其中的 Cookie，不会把它降级覆盖）。环境自检对 Firefox 环境不提供噪声种子、字体列表、启动参数等 Chromium 专用修复。不伪装显卡、Canvas、系统平台；navigator.webdriver 为 true（Firefox 远程控制的固有行为）；RPA、录制、多窗口同步器、排列窗口和扩展中心暂不支持 Firefox 环境（界面中标注「Firefox 内核暂不支持」）
- 扩展中心：按商店 ID / 应用商店链接、CRX 链接或本地 .crx/.zip 添加扩展（保留原扩展 ID），分配给环境后启动时自动加载（两种内核均支持）；Cookie 支持 JSON 与 Netscape cookies.txt 导入导出，运行中直接写入浏览器，未运行时暂存到下次启动；可连接本地 Clash / Mihomo 控制器，查看策略组与节点、切换节点、测延迟，并一键生成指向 Mihomo 端口的代理
  - 界面：「扩展」页以卡片展示图标、版本、MV2/MV3、来源和使用环境数，可搜索、按来源筛选、删除（自动从环境中移除），并通过「分配到环境」弹窗批量勾选环境；环境编辑抽屉里也可逐个开关扩展
  - 界面：环境抽屉 →「Cookies · 管理」打开导入/导出弹窗：自动识别 JSON / Netscape，支持选择文件、拖入、剪贴板，导出可复制或下载；顶部显示环境是否运行（立即写入 / 下次启动写入、快照时间、待写入条数）
  - 界面：「代理」页工具栏「Clash / Mihomo」：填写控制器地址与密钥连接，显示版本、模式、监听端口；按策略组切换节点、单个或全部测速，一键「添加为代理」。注意：所有使用同一监听端口的环境共享当前节点
- **启动时打开指纹检测页**：「设置 → 指纹检测页」开启后（默认关闭，随全局设置云同步），从界面启动环境时会在环境自己的启动页（或恢复的会话）之后多开一个标签页并切到它，不替换原有页面；环境没有启动页时直接在第一个标签页打开。可选 BrowserScan、Pixelscan、CreepJS、BrowserLeaks、iphey、whoer、内置自检或自定义网址；打开时机可选「每次启动」或「仅首次启动」（每个环境只打开一次：新环境首次启动，或开启后的第一次启动，可在环境里重置）。环境抽屉「启动」里可单独设置 跟随全局 / 开 / 关，以及专属检测网站和打开时机。三种内核（定制 Chromium、开源内核、Firefox）都支持。本地 API、MCP、RPA 运行、录制和默认浏览器链接启动的环境不会打开检测页，不干扰自动化
  - 内置自检：GhostBrowser 在本机 127.0.0.1 随机端口提供的页面（每次启动一个随机链接），逐项对比浏览器实际暴露的值与这次启动生效的配置：UA、navigator.platform、语言、时区、分辨率、像素比、色深、CPU 核心数、内存、触控点、WebGL 厂商 / 渲染器、navigator.webdriver、DNT，以及出口 IP（与代理上次检测结果比对，通过 api.ipify.org 查询）。Canvas / 音频噪声和字体请用外部检测页查看；GhostBrowser 重启后旧链接失效
- 列表可按分组和标签筛选，按最近运行、名称或创建时间排序；搜索支持名称、分组、标签和出口 IP
- 批量操作（勾选后底部出现操作栏）：启动、停止、改分组或加标签、分配代理、立即同步、删除
- 「运行中」条显示每个环境的运行时长和出口 IP，可以一键把浏览器窗口切到前台，或直接停止
- 排列窗口：「运行中」条右侧一键把运行中的窗口按网格 / 层叠 / 重叠排列，弹窗里实时预览布局（按屏幕比例绘制，格子内显示环境名），可选全部运行中或仅已勾选的环境，失败的窗口会列在提示里
- 多窗口同步器（群控）：对话框里选一个主控窗口和多个受控窗口；同步点击 / 输入 / 按键 / 滚动 / 标签页可分别开关（同步中修改立即生效）；可设随机操作延迟区间（毫秒），每个受控窗口对每个操作独立随机等待，模拟真人节奏；标签页同步：主控新建、切换、关闭标签页会在受控窗口镜像执行，主控与受控标签页一一对应，操作落到对应的标签页（不再只限第一个标签页）；一键网格平铺、重叠排列、统一大小（受控窗口改成主控窗口大小，位置不变）、显示窗口（还原最小化并前置）、统一标签页（受控窗口打开与主控相同的网址、相同顺序和当前标签页）、关闭当前 / 其他 / 空白标签；批量访问网址（自动补全 https://，可选新标签页）。窗口和标签页操作不需要先开始同步；实时显示同步事件数、每个受控窗口的延迟、标签页数和错误；选项与所选窗口自动保存。iframe（同源与跨域）内的点击、输入、滚动和下拉选择同步到受控窗口的对应 iframe（按 iframe 路径匹配，找不到时按坐标换算）；输入法候选与文件选择框不同步
- 存储列显示同步状态：同步中、待上传、已同步（含云端版本号）、仅本地
- 多个环境共用同一个出口 IP 时会标黄提醒，代理失效时会标红
- 环境自检 / 一键修复：环境行的盾牌按钮打开「环境检测」，对照当前出口 IP 逐项检查并显示「当前 / 期望」：出口 IP 与国家、语言、时区（含 UTC 偏移比对）、地理位置（距出口距离）、WebRTC 防泄漏、navigator.webdriver（自定义启动参数里的 --enable-automation）、字体数量与是否混入其他系统字体、Canvas/WebGL 噪声种子、UA / navigator.platform / 浏览器版本 / WebGL 渲染器一致性，以及系统时钟（与 HTTPS 服务器 Date 头比对，偏差超过 2 分钟标红，连不上时显示「尚未确认」）
  - 可自动修复的项目：语言 / 时区 / 地理位置改为跟随 IP 并写入当前出口的值、WebRTC 改为替换为代理 IP、重新生成噪声种子、按系统字体库重新生成字体列表、移除自动化启动参数；支持一键修复全部或单项修复
  - 只能检测、不会自动修复：出口 IP（请检查或更换代理）、UA 一致性（请重新生成指纹）、系统时钟（请开启系统自动对时，本程序不修改系统时间）；webdriver 按配置判断，不读取运行中的浏览器
  - 修复前自动备份原值，「修复记录」逐条显示时间和每个字段的 旧值 → 新值，可一键恢复（只改回该次修复动过的字段，不影响之后的其他修改）；记录保存在 %APPDATA%/GhostBrowser/repairlog/
- 接口与审计页：MCP 服务一键启停（可改端口，显示运行状态、调用次数、最近客户端和最近调用，端口被占用时给出原因）；本地 API 地址与 Key（默认隐藏，可显示 / 复制）；按 Cursor（HTTP）、Claude Desktop（stdio）、Claude Code（命令行）一键复制接入配置；列出全部 MCP 工具并区分只读和会修改数据的工具
- 操作日志：界面、本地 API、MCP 和云同步的每次操作（启动 / 停止 / 新建 / 编辑 / 删除 / 克隆环境、分配代理、分组、同步、MCP 调用）都会记录，可按分类、来源、环境筛选，搜索，只看失败，每 5 秒自动刷新，并按当前筛选导出 CSV / JSON 或清空

### 指纹
- 按操作系统生成相互匹配的 UA、平台、屏幕、DPR、CPU 核数、内存、WebGL、字体
- Canvas、WebGL、Audio、ClientRects 噪声按种子固定
- 时区、语言、地理位置可以跟随代理出口 IP；启动前会重新检测出口
- 内置设备库 + 自定义指纹库（`library.json`），下拉框联动并支持保存自定义值
- **防护与启动选项**（环境编辑「防护」「启动」页，开源内核生效）：Canvas / WebGL 图像 / AudioContext / ClientRects 各自可选 噪声、真实、关闭；WebRTC、媒体设备（可自定义麦克风/扬声器/摄像头数量）、语音列表、WebGPU、Do Not Track；端口扫描保护（白名单端口）、每次启动随机指纹；窗口大小（自动/自定义宽高位置/最大化）、硬件加速、禁止图片/视频、静音、自定义启动参数（user-data-dir、remote-debugging、proxy-server、user-agent 等由程序接管）。定制内核下会提示哪些项不生效
- **指纹附加项**（开源内核）：媒体设备 ID 别名——媒体设备选「真实」时可打开「设备 ID 别名」，网站看到的 deviceId 按环境种子 + 网站来源生成（同一环境同一网站永远一致，换网站 / 换种子就不同），getUserMedia、轨道 getSettings、setSinkId 等自动换回真实 ID，摄像头和麦克风照常可用；未授权时为空的 ID 保持为空；groupId 与 Chrome 一样每次加载重新加盐。WebGPU 关闭会移除 navigator.gpu 并加 --disable-blink-features=WebGPU；语音列表可关闭。环境编辑「防护」页可「重新生成指纹种子」（确认后立即保存，只换噪声种子，Cookie、登录状态和其他参数不变），并显示种子短 ID（前 8 位十六进制）
- **终端**（侧栏「终端」）：一键打开 Claude Code（`claude`）、Codex CLI（`codex`）、Gemini CLI（`gemini`）或 Windows PowerShell 纯终端；自动检测 PATH 中是否已安装并显示版本，未安装时给出安装命令和官方安装指南。可选工作目录（文件夹选择或手动输入，留空为用户目录），可继承某个环境的时区（TZ）、语言（LANG / LC_ALL）和代理（HTTP_PROXY / HTTPS_PROXY / ALL_PROXY），也可自定义时区和语言；带账号密码或 SOCKS 的代理经本地中转变成 `http://127.0.0.1` 代理，中转在终端关闭时一起停止。新窗口可选 PowerShell 控制台或 Windows Terminal（已安装时）；记住上次的选择，列出从这里打开的终端并可关闭。PowerShell 控制台窗口直接拥有自己的控制台输入输出（修复 0.8.3 及之前 PowerShell / Codex 等独立窗口一闪就关的问题）；窗口在启动 3 秒内退出时会提示退出码，不再无声消失

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
- 批量分配给环境：顺序轮流、随机（数量够时不重复）、统一用一个；可只选「可用」或「未被占用」的代理；可按订阅分组一键选中，把一个订阅的节点轮流分给选中的环境
- **代理订阅**（代理 → 订阅）：可添加多个机场 / 服务商订阅，每个订阅的节点在代理列表里单独成组（分组筛选、分组标题、「检测本组」），环境编辑里的代理选择也按订阅分组、可搜索
  - 支持 Clash / Mihomo YAML、Base64 和分享链接列表：`http(s)://` `socks5://` `ss://`（含 obfs / v2ray-plugin） `vmess://` `vless://`（tls / reality / ws / grpc / h2） `trojan://` `hysteria2://` `hy2://` `tuic://` `anytls://`；`ssr://` 与 sing-box JSON 暂不支持
  - 默认以 `clash-verge/v1.7` 的 User-Agent 下载（可改），30 秒超时、20 MB 上限；直连失败时经已连接的 Mihomo 端口重试
  - 添加前先解析预览：节点总数、各协议数量、需要内核的节点数、流量（subscription-userinfo）、到期时间、建议更新间隔
  - 订阅列表显示节点数、已用 / 总流量、到期、上次更新与错误；自动更新间隔默认取订阅的 profile-update-interval（否则 24 小时，0 = 关闭），失败 30 分钟后重试；名称默认取 content-disposition 文件名或域名
  - 更新时按节点指纹（类型 + 地址 + 端口 + 凭据）合并：已有节点 ID 不变，绑定的环境继续可用；订阅里消失的节点如仍有环境在用会标记「已失效」保留，否则删除；节点 ID 由订阅 + 指纹决定，两台电脑同步后不会冲突
  - 删除订阅时可选择同时删除节点（在用的环境改为直连）或保留为手动代理
  - 订阅随「软件全局数据」云同步（更新结果、流量不参与冲突判断）
- **代理内核 (mihomo)**（设置 → 代理内核）：浏览器只支持 HTTP / SOCKS5，VLESS / VMess / SS / Trojan / Hysteria2 / TUIC / AnyTLS / HTTPS 等订阅节点由 [mihomo](https://github.com/MetaCubeX/mihomo)（GPL-3.0，独立下载运行，GhostBrowser 不包含其代码）在本机转换
  - 一键从 GitHub Releases 下载最新版（Windows amd64 compatible 版），用 `-v` 校验；也可选择本地已有的 mihomo / Clash Meta 程序
  - 只为正在使用的节点各开一个 127.0.0.1 混合端口（`listeners` + `proxy` 直连该节点），不开 LAN、不改系统代理和 DNS；多个环境共用一个内核进程，节点变化通过 REST API 热重载，已运行的环境端口不变；没有环境使用时自动停止，退出软件时结束
  - 启动环境、出口检测、IP 匹配时区 / 语言、测速都经过本地端口；未安装内核时启动会提示去设置下载，检测会标记「需要内核」
  - 代理列表显示协议标签和「内核」徽标；Windows 下内核进程不弹黑窗口

### 云同步
- 客户端加密：AES-256-GCM，密钥由本机的同步密码派生，服务器只存密文
- **增量同步**：每个文件切成最大 4 MB 的加密块，只上传服务器上没有的块；拉取时本地已有的块直接复用。首次同步后，关闭浏览器通常只传几 MB。单个环境最大 2 GB（不含缓存）
- 自动同步：启动时以服务器为准对比；启动环境前拉取云端新版本；关闭浏览器后上传（失败自动重试）；退出软件前等待上传完成
- 云端版本较新时弹窗倒计时询问，默认动作可在设置里改
- **同步网站登录状态（会话存储，默认开启）**：浏览器运行时，登录快照（`ghost-session.json`，随环境一起加密同步）除普通 Cookie 外还包含分区 Cookie（CHIPS，保留 partitionKey / SameSite / 过期时间 / Secure / HttpOnly / 域名 / 路径）、每个网站的 LocalStorage 和 IndexedDB；在另一台电脑启动时自动写回，网页应用（SPA）换设备后也保持登录。两种内核（定制内核 / 开源 Chromium）都通过 CDP 实现：
  - 采集：每 15 秒刷新 Cookie 和已打开网站的 LocalStorage，IndexedDB 每 60 秒及关闭浏览器前采集一次；在独立的隔离环境中读取，不经过网页自己的脚本。没打开的网站沿用上次快照（90 天未访问自动清理）；只采集 http/https 网站，不含扩展页面和 chrome:// 页面
  - 恢复：在隐藏标签页里打开对应网站，导航请求被直接拦截返回空白页（不访问网络、不运行网站脚本），再写入 LocalStorage 和 IndexedDB（按原版本号建库、补建表和索引）；之前已打开的同站标签会自动刷新
  - IndexedDB 为尽力恢复：支持字符串、数字、BigInt、日期、正则、ArrayBuffer/TypedArray、Blob/File、Map/Set、嵌套对象和复合主键；无法序列化的值（CryptoKey、文件句柄等）会跳过并标记「部分」
  - 设置 → 云端同步可关闭（关闭后只同步普通 Cookie，与旧版格式相同）并设置单个环境的快照上限（默认 32 MB）：超出时保留全部 Cookie 和最近访问网站的 LocalStorage，较早网站的 IndexedDB 先舍弃；Cookie 弹窗显示快照包含的内容和大小
  - 快照格式升级为 schema 2：新版仍能读取旧快照；旧版客户端遇到 schema 2 不会只恢复一半（会当作没有快照），本版遇到更新的 schema 会拒绝读取和覆盖。建议所有设备升级到同一版本
- 全局数据同步：设置、代理列表（含凭据、标签、换 IP 链接）、自定义指纹库
- **历史版本**：「云同步 → 历史版本」可以恢复服务器保留的任意版本，恢复后会作为最新版本上传，其他电脑也会同步到这个版本
- **同步日志**：最近 300 次同步的时间、流量、耗时和错误，失败的可以一键重试
- **修复云端（v0.8.8）**：拉取时提示「云端缺少该环境的部分数据分块」说明服务器上丢了这个环境引用的数据块（v0.8.7 及更早的 `ghostsync.php` 存在数据块被误清理的问题，见下文）。在**上传它的那台设备**上点 环境列表 更多 → 修复云端（强制重新上传），或「云同步 → 全部环境」每行的 ⋯ → 修复云端：客户端重新向服务器确认缺少哪些块、全部补传，并提交一个新版本（即使本机没有改动），其他设备再拉取即可。同一菜单的「检查云端完整性」会下载并解密清单逐块核对（旧服务端也可用）。「重新上传全局数据」用于强制重传设置 / 代理 / 指纹库这条全局记录
- **上传 / 修复时浏览器还开着（v0.8.9）**：环境正在运行时「上传」「修复云端」和自动同步会直接拒绝（「环境正在运行，请先关闭后再修复/上传」）；窗口关了但还有浏览器进程在后台写这个环境目录（残留的 chrome.exe / 崩溃处理进程等，Windows 上通过环境目录里被独占的 `lockfile` / `parent.lock` 判断）时提示「检测到该环境仍有浏览器进程在后台运行（可能是残留进程）」；如果这些进程是 GhostBrowser 为这个环境启动的，提示里会有「结束残留进程」按钮，确认后一键结束（只结束命令行指向该环境目录的浏览器进程）。关闭浏览器后的自动上传会先等残留进程退出（最多 20 秒）
- **同步中变化的文件（v0.8.9）**：扫描后到上传前有文件被改写（比如缓存索引），不再整次失败（旧版报 `index changed during sync`），而是把这个文件重新读入内存（两次读取一致才算稳定）、重新切块并上传新内容，清单和块列表都以最终内容为准；同一文件重试 2 次仍在变化才报错「文件 <路径> 在同步过程中一直在变化，说明浏览器或残留进程仍在写入，请关闭后重试」。「修复云端」不再使用本地分块缓存，所有文件重新计算（内存映射的文件可能内容变了而修改时间不变）
- **不同步的纯缓存目录（v0.8.9 起）**：Cache、Code Cache、Media Cache、GPUCache、ShaderCache、GrShaderCache、DawnCache、DawnGraphiteCache、GraphiteDawnCache、DawnWebGPUCache、component_crx_cache、extensions_crx_cache、optimization_guide_model_store、BrowserMetrics（及 BrowserMetrics-spare.pma）、blob_storage、Crashpad，以及根目录的 `lockfile` / `parent.lock`，浏览器会自己重建。网站数据（Cookie、Local/Session Storage、IndexedDB、Service Worker 及其 ScriptCache、CacheStorage）照常同步（v0.8.10 起按「同步内容」模式决定，见下）。旧版本上传的清单里如果还带着这些文件，恢复时照样还原，不影响兼容
- **同步内容（v0.8.10，设置 → 云端同步，随设置同步到其他设备；每个云端环境可在编辑窗口「基本信息 → 同步内容」单独设置，并显示每种模式的预计大小）**：
  - **精简（默认）**：只同步登录状态和环境配置：Cookie（含 `Network/Cookies`、Device Bound Sessions）、LocalStorage、IndexedDB、WebStorage（存储桶）、已存密码（Login Data）、自动填充（Web Data）、扩展及其数据（Extensions、Local Extension Settings、Extension Rules/Scripts、Storage/ext）、书签、Preferences / Secure Preferences / Local State、MediaDeviceSalts、指纹配置和登录快照 `ghost-session.json`
  - **标准**：精简 + 浏览历史（History、Favicons、Top Sites、Visited Links、Shortcuts）、上次的标签页（Sessions、Session Storage）、Service Worker（含 CacheStorage 离线缓存和 ScriptCache）、网页推送订阅（GCM Store）、File System、通知数据库
  - **完整**：除下面「从不同步」的内容外全部同步（另含 Sync Data、DIPS、Network Action Predictor、Extension State、隐私沙盒 / Trust Tokens / Interest Groups 等浏览器内部状态、segmentation_platform、shared_proto_db 等）
  - **从不同步（所有模式）**：网页缓存、代码缓存、GPU / 着色器缓存（见下一条）；用户数据根目录下组件更新器下载的数据，浏览器会自己重新下载：screen_ai、WidevineCdm、OnDeviceHeadSuggestModel、Safe Browsing、ZxcvbnData、hyphen-data、Subresource Filter、TrustTokenKeyCommitments、MEIPreload、FileTypePolicies、SSLErrorAssistant、CertificateRevocation、Crowd Deny、OriginTrials、PKIMetadata、AutofillStates、CookieReadinessList、FirstPartySetsPreloaded / first_party_sets.db、PrivacySandboxAttestationsPreloaded、SafetyTips、OptimizationHints、OptimizationGuideModelsManifest、WasmTtsEngine、GPUPersistentCache 等；环境里可重建的数据库：Download Service、Feature Engagement Tracker、optimization_guide_*、heavy_ad_intervention_opt_out.db、commerce / cart / coupon / discount / parcel 数据库、AutofillAiModelCache、Shared Dictionary、Reporting and NEL、SCT Auditing、BudgetDatabase、VideoDecodeStats、Site Characteristics Database、Affiliation Database；我们自己的 `launch.log`
  - SQLite 的 `-journal` / `-wal` / `-shm` 跟随所属数据库一起同步或一起排除：浏览器正常退出后它们是空的（不占流量），异常退出时里面是数据库需要的数据，不能单独丢掉
  - 兼容：模式只决定新上传的内容。旧版本或其他模式上传的清单 / 旧 zip 格式恢复时照样还原其中所有文件；拉取时本机已有、但云端副本按设计没带的数据（缓存、组件、精简模式下的历史记录等）会保留在本机，不会因为拉取而丢失。分享给其他用户的环境始终按精简打包（不带历史记录）
  - 实测（Linux Chromium 156，打开 YouTube、TikTok、WhatsApp Web、Telegram Web、Discord、Reddit、Bilibili、Amazon 等 11 个网站后关闭）：旧版（相当于完整）首次上传 39–71 MB（大部分是 Service Worker 的 CacheStorage + ScriptCache），精简只有 2.5 MB；再次打开这些网站后关闭，完整上传 8.6 MB、精简 2.7 MB（压缩前；网络实际传输 3.0 MB 对 0.7 MB），剩下的主要是登录快照和 IndexedDB 的变化。Windows 上的 Chrome / Undetectable 内核还会在根目录下载 screen_ai（100 MB 以上）、WidevineCdm 等组件，以前它们和网站离线缓存一起被同步，是「每次关闭上传几百 MB」的主要来源
- **传输面板显示实际流量（v0.8.10）**：上传时进度条按「变化数据（压缩前）」计算；完成后显示「实际上传 X」（压缩加密后真正经过网络的字节，含清单），以及「跳过 Y（未变化）」——跳过的部分没有上传，旧版的「省 Y」容易被看成上传量
- 普通上传也更稳：提交时服务器发现有数据块不在（409 `chunk_missing`），客户端会补传缺的块并自动重试一次提交
- 错误提示：服务器返回网页而不是 JSON 时（nginx / CDN 的错误页），错误里会带上请求的方法和接口（如 `POST /api/chunks/get`），并提示「服务器前端（nginx/CDN）替换了错误页，真实错误被隐藏」；宝塔请在网站配置里把 `fastcgi_intercept_errors` 改为 `off`，或删掉 `error_page 404 ...` 这类行后重载 nginx
- **删除环境可选「只删本地」或「同时删云端」**（单个和批量删除共用一个确认框，记住上次的选择；未配置同步时不显示选项）：
  - 只删本地：删除本机的环境和数据目录，云端副本保留且**不会再被自动下载回来**；「云同步 → 仅云端」列出这些环境（标记「仅云端（已在本机删除）」，其他设备上传了新版本时显示「云端有更新」），可「恢复到本机」或「删除云端」
  - 同时删云端：服务器删除该环境的数据、全部历史版本和增量块引用，只留一条不含数据的删除记录（墓碑，名称 / 时间 / 设备，默认保留 90 天）。其他设备在启动对比、立即同步、启动该环境前会看到墓碑并自动删除本机副本（提示「环境 X 已在 某设备 删除，已同步删除本机副本」）；正在运行的会在浏览器关闭后再删除；旧设备的上传会被服务器拒绝（HTTP 410），不会把环境“复活”
  - 服务器连不上时先删除本机副本，云端删除排队保存，联网后（启动、立即同步或每分钟）自动重试；另一台设备正在上传该环境时服务器返回 423，可选择强制删除
  - 两种删除都会写入操作日志（删除环境、同步 · 删除云端、同步 · 删除本机副本）。需要把新版 `ghostsync.php` 上传到服务器，旧版服务端删除后其他设备不会自动删除
- **最近删除（回收站）**：在本机删除的环境（只删本地、同时删云端，或因其他设备删除了云端而被同步删除）不会立刻销毁，环境记录和数据目录（移到 `profiles/.trash/<ID>`）先保留在「最近删除」，默认 30 天（设置项 `trashDays`），到期在启动时自动彻底删除；别人分享给你的环境（只读副本）删除时直接彻底删除，不进回收站
  - 环境列表工具栏的回收站按钮（带数量角标）打开「最近删除」：显示名称、删除时间、数据大小、剩余天数和云端状态（仅本机 / 云端副本保留 / 云端已删除），可单个「恢复」「彻底删除」或「清空回收站」（彻底删除会二次确认）
  - 回收站里的环境不会出现在环境列表、启动、批量操作、多窗口同步器和本地 API 中，自动同步不会上传它，也不会把它从云端重新下载回来
  - 恢复后回到环境列表，数据目录移回原处，自动同步继续跟踪；如果云端副本已被删除（墓碑），恢复为一个新 ID 的本地环境，自动同步会把它作为新环境上传
  - 彻底删除会删除数据目录和环境记录（包括环境自带的代理账号密码）；代理列表中的共享代理不受影响。删除 / 恢复 / 彻底删除都写入操作日志；本地 API `GET /trash` 只读列出回收站
- 显示服务器信息：已用空间、磁盘剩余、单次上传上限、保留的版本数
- 同步地址使用 http 时会提示：令牌会明文传输，建议改用 https
- 旧版服务端（不支持增量）会自动改用整包同步
- **账号安全与分享**（「云同步 → 设备与密钥 / 分享」，需要新版 `ghostsync.php`）：
  - 设备管理：每台电脑首次连接时自动登记（名称、系统、首次 / 最近连接）；可开启「新设备需要批准」，批准前该设备只能看到自己的状态；**吊销**后它的所有请求被服务器拒绝（HTTP 403，中文提示），可重新批准或移除记录。不能吊销当前设备
  - 恢复密钥：云端数据改用「账号主密钥」加密，同步密码和恢复密钥（8 组 4 位，只显示一次，可复制 / 下载 .txt，必须勾选「我已保存」）都能解开它。新设备可用同步密码或恢复密钥解锁；修改同步密码、轮换恢复密钥都只重新包装主密钥，不重新加密云端数据。**老账号无缝迁移**：已有云端数据时主密钥就是当前同步密码（会先用它解密云端全局数据核对），旧数据保持可读
  - 环境分享：把环境分享给同一服务器上的其他用户，可选「仅配置」或「包含登录状态」、有效期 1 / 7 / 30 / 90 天或自定义（≤366 天）和备注。分享包在本机用对方的 X25519 公钥加密，服务器只存密文。对方在「收到的」里接受 / 拒绝，接受后环境列表出现带「共享 · 发送者」标记的只读环境（不能编辑、复制、上传到对方自己的云端，浏览数据只留在对方本机）；发送者可撤销 / 续期，接收者可退出。撤销、过期或退出后，对方本机副本在下次分享同步（每 5 分钟、打开分享页、启动该环境前）自动删除
  - 管理页：`ghostsync.php?r=/admin`（设置 `GHOST_ADMIN_TOKEN` 后启用）只显示用户、存储、设备数、分享数和最近错误，从不显示令牌、密码或任何解密内容

### 界面
- 左侧边栏可收起为图标栏（侧栏底部按钮或 Ctrl+B），悬停显示名称和快捷键，数量变为圆点；状态会记住，下次启动保持；系统开启「减少动态效果」时不播放动画
- 所有下拉菜单、弹出面板和悬停提示（启动按钮旁的 ▾ 网站快捷入口、「⋯」更多操作、筛选/排序、表单下拉、可输入下拉、排列窗口、同步器的窗口选择、RPA 添加步骤、导出、公告）统一浮在最上层：不会被表格、滚动区域或对话框裁剪遮挡，下方空间不够时自动向上展开并限制在窗口内；点击外部、按 Esc、窗口缩放或列表滚动时自动关闭；在对话框或抽屉里打开的菜单按 Esc 只关闭菜单

### 应用外壳（托盘 / 更新 / 快捷入口 / 公告）
- **待重启**：环境运行中修改代理、指纹、内核、启动参数或语言时区后，环境列表、运行条和编辑抽屉显示「待重启」，一键重启生效；只改名称、分组、标签、备注不会提示
- **托盘常驻**（Windows）：关闭主窗口后隐藏到托盘，运行中的环境和代理继续运行；托盘菜单：显示主窗口、运行中的环境（点击切到前台）、全部关闭、退出。可在「设置 → 应用与更新」关闭
- **在线更新**：启动时和每 6 小时检查一次（也可手动「检查更新」），有新版本时顶栏出现提示，查看更新说明后「立即更新」，带下载进度。更新源默认是你的同步服务器（`update/latest.json`），也可填自定义地址或公开 GitHub 仓库。安装前校验 SHA-256 和 Ed25519 签名，签名不符拒绝安装；替换 EXE 后自动重启（旧文件改名为 `.old`，下次启动清理）
- **默认浏览器（按环境）**（Windows）：环境名称旁的 ☆ 设为默认浏览器（同一时间只有一个，再点一次取消）。在「设置 → 默认浏览器」点「注册并打开系统设置」：GhostBrowser 只在当前用户注册表（HKCU，不需要管理员）登记为浏览器（http / https 以及 .htm .html .shtml .xht .xhtml .svg），随后打开系统「默认应用」页面（Windows 11 直接跳到 GhostBrowser 页面）——Windows 10/11 不允许程序自己改默认浏览器，需要你在那里选择一次。之后微信、邮件、文档里点开的链接都交给星标环境：运行中的环境新开标签页并切到前台；未运行的环境走正常启动流程（启动前云同步拉取、IP 地理匹配、指纹注入），第一个标签页直接打开该链接（不再停在空白页）；启动过程中点开的多个链接排队，启动后全部打开。没有默认环境（或已删除）时，主窗口弹出小窗选择用哪个环境打开，可勾选「记住为默认」。由链接冷启动 GhostBrowser 时主窗口隐藏到托盘，只打开链接；程序改为单实例运行，再次打开 EXE 会显示已运行的窗口。只接受 http / https / file 链接。便携版 EXE 移动位置后启动时自动重新注册；「取消注册」删除全部注册表项。默认环境只保存在本机，不随全局设置同步。每次打开链接记入操作日志
- **网站快捷入口**：环境列表「启动」旁的箭头，启动环境并直接打开常用网站（运行中的环境在新标签页打开），也可输入任意网址；列表由同步服务器的 `catalog.json` 下发，离线时用缓存
- **公告中心**：顶栏铃铛显示未读数，公告支持 Markdown（安全渲染，不执行 HTML），按开始 / 结束时间显示
- **导入导出版本号**：导出的环境 JSON 带 `schema` 字段（当前 1）；旧版本客户端导入更新版本导出的文档时会明确提示先升级，而不是悄悄丢掉字段

#### 发布更新（维护者）
```bash
go run ./cmd/sign-release keygen -out keys        # 只需一次；私钥 keys/ghostbrowser-update.key 务必离线保管、不要提交
# 把 keys/ghostbrowser-update.pub 的内容写进 updater.go 的 updatePubKey（或 -ldflags "-X main.updatePubKey=..."）
wails build -platform windows/amd64 -clean -trimpath -ldflags "-s -w"
go run ./cmd/sign-release sign -key keys/ghostbrowser-update.key -file build/bin/GhostBrowser.exe \
  -version 0.8.0 -url GhostBrowser-0.8.0.exe -notes-file CHANGELOG.md -out latest.json
# 上传到服务器 ghostsync.php 同目录：update/latest.json 和 update/GhostBrowser-0.8.0.exe（文件名与 -url 一致）
```
版本号取自 `mcp.go` 的 `mcpAppVersion()`，发布前记得改。私钥丢失后只能换新密钥对，旧客户端将无法自动更新到用新密钥签名的版本。

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
- **v0.8.8 必须更新 `ghostsync.php`**：v0.8.7 及更早的服务端在上传清单时把刚收到的数据块列表（refs）删掉了，服务器因此认为这些块“没人引用”，超过 2 天的块会在下一次清理（每 6 小时随提交触发，或删除环境时立即触发）被删除，表现为拉取时 `chunks/get` 404。新版：保留块列表；`chunks/missing` 报告“已存在”的块会刷新时间；清理时把正在上传中的块列表也算作引用；任何保留版本没有块列表（旧服务端上传的）时暂停清理，直到这些环境重新上传；提交前逐个确认块都在，否则拒绝提交（409 `chunk_missing`），云端不会再出现指向缺失数据的版本；缺块返回 409 JSON（不再是会被 nginx `error_page` 换成网页的 404）；新增 `GET /api/profiles/{id}/check`，管理页「云端数据」列标出缺块的环境
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
    GET  /api/profiles/{id}/check           {refs, chunks, missing, sample}: chunk health of the current version (feature "chunkcheck")
    POST /api/chunks/missing                {"ids": [...]} -> {"missing": [...]}; refreshes the mtime of every chunk reported present
    PUT  /api/chunks                        framed: (64-hex id, uint32 BE length, GBC1 data)*
    POST /api/chunks/get                    {"ids": [...]} -> framed, same layout; 409 {error, code:"chunk_missing", missing:[...], count} if any is gone
Commit refuses (409 code "chunk_missing", same body) when a chunk named in the pending refs is not stored, so a committed manifest never
points at absent chunks; the client then re-checks, uploads what is missing and commits once more. PUT blob keeps the pending refs.
Chunk GC (every 6 h on commit, forced on delete) treats tmp/*.refs as live and skips chunks younger than 48 h; it does nothing while
any kept version lacks refs/<sha>.txt (uploaded through ghostsync.php <= 0.8.7).
All requests carry Authorization: Bearer <token> (and X-Ghost-Token for hosts that strip it).
Upload, refs, commit and unlock also send X-Device-ID and X-Lock-Token.
Deletion leaves a tombstone: the index entry keeps id/name, its version is bumped, data/history/refs are released (blob and chunk GC), and
deleted/deletedAt/deletedBy are set. The plain list (old clients) omits tombstones; ping lists the "tombstones" feature and tombstoneDays.
Blob/history GET, lock and commit on a tombstone return 410, so a stale device cannot resurrect it; an explicit lock {"restore": true}
re-creates it. Tombstones are purged after GHOST_TOMBSTONE_DAYS (default 90, env GHOST_SYNC_TOMBSTONE_DAYS).
Accounts & security (ping features "devices", "keys", "directory", "shares"; every route is additive, old clients are unaffected):
    GET  /api/devices/self                  {id, status: approved|pending|revoked, requireApproval} (allowed while pending)
    GET  /api/devices                       {self, requireApproval, devices:[{id,name,os,firstSeen,lastSeen,status}]}
    POST /api/devices/{id}/approve|revoke|forget ; POST /api/devices/settings {"requireApproval": bool}
    GET  /api/keys | PUT /api/keys          {epoch, envelope}; PUT {"envelope": {...}, "epoch": <read epoch>} is compare-and-swap (409)
    GET  /api/directory | GET /api/directory/{user} | POST /api/directory {"pub": base64 X25519}
    GET  /api/shares                        {self, out:[...], in:[... with wrap]}  (statuses pending|accepted|declined|revoked|left|expired)
    POST /api/shares                        {to, name, mode: session|config, note, days 1..366, os, wrap:{eph,nonce,ct}} -> {id, expiresAt}
    PUT|GET /api/shares/{sid}/blob          GBE1 share package (PUT by sender once; GET by recipient while pending/accepted, else 410)
    POST /api/shares/{sid}/accept|decline|leave (recipient), revoke|extend {"days"} (sender)
Every request registers X-Device-ID (+ X-Device-Name, X-Device-OS); revoked devices get 403 everywhere, pending ones everywhere except ping and devices/self.
Key envelope (opaque to the server): v=1, kid, legacy, pw = AES-256-GCM(PBKDF2-SHA256(password, salt, 200000), M), rk = AES-256-GCM(HKDF-SHA256(160-bit recovery key, salt), M),
id = X25519 private key sealed with HKDF(M, "GBK1 identity"). M replaces the password as the input of the existing per-blob PBKDF2 (GBE1) and is M = password bytes for accounts
that already had cloud data, so nothing is re-encrypted. Share key K (32 random bytes) is wrapped with ephemeral X25519 -> HKDF-SHA256 -> AES-256-GCM; the package is encrypt(zip, K).
Chunk IDs are HMAC-SHA256(per-profile chunk key, plaintext); chunk bodies are "GBC1" + nonce + AES-256-GCM(deflate(data)). The chunk key lives only inside the encrypted manifest.

### Session snapshot (ghost-session.json) schema
Lives in the profile folder and is synced inside the encrypted manifest like any other file (no server change; chunked, so the 413/body-cap logic applies unchanged).
- Schema 1 (v0.7 and earlier): `{"device","savedAt","cookies":[CDP Cookie...]}`; partitioned cookies were not restored.
- Schema 2: `{"schema":2,"device","savedAt","cookies2":[full CDP Cookie incl. partitionKey],"origins":[{"origin","savedAt","local":{k:v},"idb":[{"name","version","stores":[{"name","keyPath","autoIncrement","indexes":[{"name","keyPath","unique","multiEntry"}],"records":[{"k","v"}],"truncated","skipped"}]}],"idbAt","idbPartial","idbDropped"}],"dropped":[origin...]}`.
  IndexedDB keys/values use a tagged-JSON codec: `{"$t":"d"|"re"|"ab"|"ta"|"blob"|"map"|"set"|"bi"|"n"|"u"|"err"|"o",...}` (`"o"` wraps a plain object that has its own `$t` key).
- Guard: cookies moved from `cookies` to `cookies2`, so a schema-1-only client sees an empty snapshot and restores nothing (instead of half a session). A client refuses to read, restore or overwrite a schema newer than its own.
- Restore happens only when `device` differs from this device; afterwards the snapshot is re-taken under this device ID.
- Cap: `Settings.sessionStoreCapMB` (default 32, max 256); all cookies first, then LocalStorage of the most recently seen origins, then their IndexedDB. Toggle: `Settings.syncSessionStorage` (absent = on; false = write schema 1).
