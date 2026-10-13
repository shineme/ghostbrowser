# GhostBrowser 定制 Chromium 与本地指纹包

交付日期：2026-10-13。适用系统：Windows x64。

此目录包含本机 Undetectable 随附的定制 Chromium 153.0.8010.37、指纹快照、二进制模板、独立 Go 启动器和分析证据。启动器直接执行 `runtime/chrome/chrome.exe`，不启动 Undetectable 主程序。GhostBrowser GUI 使用同一套 Go 原生启动代码。

GitHub 源码仓库通过 `.gitignore` 排除了约 505 MiB 的 `runtime/`、生成的 `bin/` 程序、主程序许可证副本和运行时校验清单；当前本机交付目录仍保留这些文件。克隆源码后，stock 模式可以使用系统浏览器；要启用原生指纹模式，需要自行放入可用且兼容的 Chromium 运行目录。

浏览器程序来自现有安装文件的完整复制。本包包含启动器源码；不包含该厂商 Chromium 的 C++ 源码，因此这里的 Go 构建命令只构建启动器或 GUI。

## 1. 文件存放与用途

```text
ghostbrowser/
  build/bin/GhostBrowser.exe           新构建的 GUI
  internal/undetectable/               GUI 与命令行共用的原生启动引擎
  cmd/ghost-launcher/                  独立启动器入口
  undetectable.go                      GhostBrowser 的指纹库、JSON 与引擎适配
  undetectable_integration_test.go     实际 Chromium 的可选集成检查
  undetectable-kit/
    README.zh-CN.md                    本文
    bin/
      ghost-launcher.exe               独立 JSON 启动器
      GhostBrowser.exe                 GUI 副本，可双击运行
    runtime/chrome/                    完整 Chromium 运行目录
    runtime-manifest.json              463 个浏览器文件的长度与 SHA-256
    fingerprints/
      library.json                    4 条去重快照，不含 Cookies 或账号令牌
      library.sqlite                  同一批指纹的 SQLite 版本
      field-index.json                明确统计口径的 JSON 字段索引
      templates/
        windows-intel-a.bin            10539 字节，旧布局模板
        windows-intel-a2.bin           10554 字节，旧布局另一份模板
        windows-intel-b.bin            10812 字节，带品牌区块的模板
    examples/
      native-profile.json             完整、可直接启动的配置
    cloud-server/                     云同步 PHP 服务端、路由和部署文档
    source/                           独立启动器源码快照及 go.mod/go.sum
    reference/
      profile-inspector/              前序检查器源码和 EXE，保留作研究参考
      host/                           安装主程序与 .orig 备份，仅用于对照分析
      host-manifest.json              主程序快照的哈希
    evidence/                         反汇编、REA 输出与本次验证记录
    licenses/                         随本机安装复制的许可证文本
    backup-source/                    修改前 GhostBrowser 源码备份
```

`chrome.dll` 包含浏览器及原生指纹读取逻辑。`chrome.exe` 负责进程入口。`chrome_elf.dll`、`icudtl.dat`、`resources.pak`、两个缩放资源包、`v8_context_snapshot.bin`、图形 DLL、`locales/` 和 `WidevineCdm/` 都保持原目录结构。不要只移动 `chrome.exe`。

完整运行文件共 529,814,860 字节，约 505 MiB。用于启动的二进制没有进行补丁修改。

云同步服务端现已提供，服务端为单文件 `cloud-server/ghostsync.php`（无需伪静态、无需数据库），详见 [PHP 云同步部署文档](cloud-server/README.zh-CN.md)。新版 GUI 会列出其它设备的云端环境，支持加密上传与元数据恢复；请使用随包更新后的 GUI 与服务端。

## 2. 直接使用 JSON 启动

在 PowerShell 中运行：

```powershell
Set-Location 'C:\Users\dhl\Desktop\ghostbrowser\undetectable-kit'
.\bin\ghost-launcher.exe -config .\examples\native-profile.json
```

启动器输出一行 JSON，包含 `profileId`、`pid`、`port`、`websocket` 和 `supervising`。它保持运行，持续为新标签页配置 UA/UA-CH。按 Ctrl+C 会关闭它启动的浏览器并清理自己的指纹桥接文件。浏览器关闭时，启动器也会退出。

只把 JSON 转成 `startInfo`，不启动浏览器：

```powershell
.\bin\ghost-launcher.exe -config .\examples\native-profile.json -prepare
```

启动、读取新标签页里的指纹、比对，然后关闭：

```powershell
.\bin\ghost-launcher.exe -config .\examples\native-profile.json -verify
```

验证页使用本机 DevTools HTTP 地址。`localhost` 是安全上下文，能读取 `navigator.userAgentData` 和 `deviceMemory`，无需访问外部检测网站。`mismatches` 为空或 `null` 表示此次检查的字段一致；不会因此声称所有模板字段都完成验证。

默认随机分配空闲 CDP 端口。可加 `-port 9227` 指定端口；已被占用时启动器报错，避免连上其它浏览器。

示例中三个文件路径均相对于 JSON 所在目录解析：

```json
{
  "schemaVersion": 1,
  "profileId": "ghost-demo",
  "name": "Ghost Native Demo",
  "chromePath": "../runtime/chrome/chrome.exe",
  "profilesRoot": "../profiles",
  "templatePath": "../fingerprints/templates/windows-intel-a.bin"
}
```

以上只是顶层路径摘录；真正可运行的完整配置是 `examples/native-profile.json`，包含必需的 `data` 和 `configMeta`。

## 3. JSON 的配置方式

顶层契约如下：

| 字段 | 含义 |
|---|---|
| `schemaVersion` | 当前固定为 1 |
| `profileId` | 1–64 位字母、数字、下划线或连字符；决定 Ghost 环境目录 |
| `name` | 显示名称，写入 `startInfo` 的 UTF-16 名称槽 |
| `chromePath` | 定制 Chromium 的 EXE 路径 |
| `profilesRoot` | Ghost 的浏览器用户数据根目录 |
| `templatePath` | 已识别布局的 `startInfo` 模板文件 |
| `data` | 指纹基础值、运行选项，沿用原软件 JSON 命名 |
| `configMeta` | 原生模板的扩展配置；未理解字段完整保存 |
| `runtime` | 明确指定 DPR、平台、颜色深度和 WebGL Vendor |
| `startUrls` | 启动后打开的 URL 数组 |
| `proxy` | 可选的无鉴权代理地址，如 `http://127.0.0.1:7890` |
| `extraArgs` | 附加 Chromium 参数，不能覆盖 profile ID、数据目录和调试端口 |
| `headless` | 可选；为 true 时传 `--headless=new` |

常用指纹参数：

| JSON 路径 | 示例值 | 本次实现的作用 |
|---|---|---|
| `data.UA` | 完整 Mozilla UA，含 `Chrome/153.0.8010.36` | 写入两份 UA 槽，再通过 CDP 设置 |
| `data.broVers` | `153.0.8010.36` | 浏览器版本记录；应与 UA/UA-CH 一致 |
| `data.osVers` | `11` | 写入系统版本槽 |
| `data.CPUCores` | `12` | `navigator.hardwareConcurrency` |
| `data.memory` | `16` | 安全上下文中的 `navigator.deviceMemory` |
| `data.resConfig` | `1536x864` | 原生屏幕宽高；可用高度默认减 48 |
| `data.langs` | `en-US#en` | 语言列表，使用 `#` 分隔 |
| `data.langHeader` | `en-US,en;q=0.9` | CDP 的语言来源；下发前去除 q 值，避免 q 值进入 JS 语言列表 |
| `data.tzId` | `America/New_York` | 原生时区；必须是有效 IANA 标识 |
| `data.webGL` | 完整 ANGLE Renderer 字符串 | WebGL 的 unmaskedRenderer |
| `data.publicRTC` | 可选 IP 字符串 | 写入已恢复的 RTC IP 槽 |
| `runtime.pixelRatio` | `1.25` | 原生 DPR |
| `runtime.platform` | `Win32` | `navigator.platform`；当前仅验证 Windows |
| `runtime.colorDepth` | `24` | 原生颜色/像素深度 |
| `runtime.webglVendor` | `Google Inc. (Intel)` | WebGL 的 unmaskedVendor |
| `configMeta.pixel1/pixel2` | 整数 | 已恢复的画布参数 |
| `configMeta.CanvasWebglRandomParameter/2` | 整数 | 已恢复的两个噪声参数 |
| `configMeta.acc` | 数值 | 原生画布累加参数 |
| `configMeta.additional_info.userAgentDataObj` | 对象 | UA-CH 品牌与高熵信息，通过 CDP 下发 |
| `configMeta.additional_info.gpuDataObj.adapterInfo` | 对象 | 写入已恢复的 GPU architecture/vendor 字符串 |
| `configMeta.additional_info.storage` | 字节数 | 写入原生 storage quota 槽 |
| `configMeta.additional_info.performance.jsHeapSizeLimit` | 字节数 | 写入原生 JS heap limit 槽 |

`additional_info` 接受对象，也兼容原数据库的 JSON 字符串格式。包内已转换为对象，便于编辑。CPU/内存等原字段兼容数值及十进制字符串。

`userAgentDataObj` 应包含 `architecture`、`bitness`、`brands`、`formFactors`、`fullVersionList`、`mobile`、`model`、`platform`、`platformVersion`、`uaFullVersion`、`wow64`。`uaFullVersion` 必须与 UA 的 Chrome 版本一致。B 模板的原生品牌槽按现有布局固定为 3 个品牌和 3 个完整版本条目。

修改最常见的值，可以直接编辑示例：

```powershell
$path = '.\examples\native-profile.json'
$config = Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
$config.profileId = 'ghost-shanghai'
$config.name = '上海环境'
$config.data.CPUCores = 6
$config.data.memory = 8
$config.data.resConfig = '1440x900'
$config.data.tzId = 'Asia/Shanghai'
$config.data.langs = 'zh-CN#zh'
$config.data.langHeader = 'zh-CN,zh;q=0.9'
$config.runtime.pixelRatio = 1.5
$config | ConvertTo-Json -Depth 100 | Set-Content -Encoding utf8 '.\examples\shanghai.json'
.\bin\ghost-launcher.exe -config .\examples\shanghai.json -verify
```

## 4. JSON 如何落到浏览器

```text
JSON data + configMeta + runtime
  → 配置校验、路径解析、语言及版本归一化
  → 加载二进制模板并检查往返编码一致
  → 替换已恢复的字符串槽，保留未知区块
  → 序列化后重新定位 CPU / 屏幕标量块
  → 写入 Ghost 的 <profilesRoot>/<profileId>/startInfo
  → 在浏览器原生读取目录放置带 owner 标记的 ghost_* 指纹副本
  → chrome.exe --profile-id=ghost_<hash> --user-data-dir=<Ghost 环境目录>
  → 等待 CDP，先为 about:blank 安装覆盖，再导航启动页
  → Target.setAutoAttach + 每个页面/子框架会话的 UA / UA-CH 设置
```

该定制内核实际读取的是二进制 `startInfo`，不会因为参数里有 `--fingerprint-config=...` 就自动理解 JSON。原 GhostBrowser 的 `custom` 模式留给实现了该参数的其它内核；新增的 `undetectable` 模式负责转换和原生读取流程。

`startInfo` 不能描述成纯 JSON，也不能完整描述成纯字符串流。它包含 `<u64 小端长度><字符串负载>` 与夹杂其中的原始标量区块。分析器识别字符串记录与 RAW 区块，在写入后根据锚点重新定位数值区，避免字符串长度变化破坏偏移。

旧模板识别为约 358 个槽，B 模板约 372 个槽；这里的槽数是当前启发式解析器的表示，不是语义上相互独立的指纹数量。

已恢复的 CPU RAW 区偏移：`+2` 内存、`+6` CPU、`+11/+15/+19/+23` 四个画布参数。屏幕区偏移：`+0/+4` 宽高、`+8/+12` 可用宽高、`+16` f64 DPR、`+28/+32` 色深、`+54` f64 acc、`+63` u64 quota、`+72` u64 heap limit。

`appVersion` 是独立槽，不随 CDP userAgent 自动变化。本实现根据最终 UA 重写它，修复前序 profile 中版本号为空的情况。

## 5. 原生配置路径的实际限制

本次额外验证发现：指纹读取器通过 Chromium `PathService` 取得系统 Local AppData，然后追加 `Undetectable/settings.json`。`--user-data-dir` 和修改子进程 `LOCALAPPDATA` 不能替代该读取路径。本次复制的浏览器在找不到新 profile 指纹时曾以 `0xC000001D` 退出；完成下面的指纹桥接后启动正常。

启动器读取本机 `%LOCALAPPDATA%\Undetectable\settings.json` 的 `profilesPath`，在此根目录内创建 `ghost_<hash>` 子目录，仅存放自己的 `startInfo` 和 `ghost-owner.txt`。普通用户浏览器数据、Cookies 和缓存都保存在 JSON 指定的 Ghost 环境目录。Ghost 不往 Undetectable 的 SQLite profiles 表添加记录，也不启动其主程序。

现有 settings.json 不会被改写。若复制到未安装过 Undetectable 的机器，启动器只在该 settings 文件不存在时创建最小配置，默认原生根目录为 `%LOCALAPPDATA%\Undetectable\Profiles`。因此浏览器主程序可独立启动，但现有内核仍要求这个固定系统读取入口。

桥接目录的名称根据 Ghost 数据根目录及 profileId 计算，以避免与原环境 ID 冲突。若已有同名目录且 owner 标记不同，拒绝覆盖。正常关闭时只删除自己生成的两个文件；不递归删除原软件目录。Windows 文件锁防止多个启动器同时启动同一 Ghost 环境，进程异常结束后锁会由系统释放。

## 6. GhostBrowser GUI 的使用

运行 `bin/GhostBrowser.exe`，或项目的 `build/bin/GhostBrowser.exe`。

1. 进入「设置」，点击「使用项目自带 Chromium 与指纹库」。若移动了项目，可先修改「本地指纹包目录」。
2. 返回环境列表，点击「指纹库 / JSON」。可以从 4 条去重快照中选择一条创建环境，或导入 `examples/native-profile.json`。
3. 点击环境的「启动」。原生模式由 GhostBrowser 保持 CDP 监督；关闭 GhostBrowser 会关闭它启动的环境。
4. 「导出启动 JSON」导出当前环境的完整配置，包含当前机器的内核/模板/数据路径，可直接供命令行启动器读取。
5. 「验证运行中的指纹」临时创建本地检测标签页、检查实际值，然后关闭该检测标签页。结果显示在 JSON 文本框中。

导入 JSON 时，GhostBrowser 生成新的环境 ID，并使用设置里的运行文件及数据根目录；导入文件中的 `chromePath`/`profilesRoot` 不会改变本机设置。已知模板依据文件名选择，未匹配时使用 A 模板。`extraArgs` 和 `headless` 保存在环境 native 配置中。

GUI 常规字段是 UA、CPU、内存、屏幕、DPR、时区、语言、WebGL 字符串的编辑入口；导出时这些值会覆盖 native.data 对应值，并同步 Chrome 品牌版本。扩展 GPU/UA-CH 参数通过 JSON 编辑。当前原生 GPU 下拉只提供已打包的 Intel 配置，避免随机切换 GPU 却仍沿用另一套参数。

标准 Chrome 的原有扩展模式、代理管理、Cookies 导入和自有云同步保留。原生模式当前不支持带用户名/密码的代理，选择这类代理会明确报错；可以使用本机无鉴权代理转发。该限制避免 GUI 显示已配置代理而浏览器实际未完成鉴权。

## 7. 从本地 HTTP API 调用

GUI 需保持运行并启用本地 API，默认地址 `http://127.0.0.1:25325`。设置中的内核模式需为 `undetectable`。

| 方法和路径 | 作用 |
|---|---|
| `GET /fingerprints` | 列出本地 4 条去重快照及配置 ID |
| `POST /profile/launch-json` | 请求体为完整启动 JSON，导入为新环境并启动 |
| `GET /profile/json?profile=<ID>` | 导出已有环境启动 JSON |
| `POST /profile/verify?profile=<ID>` | 检查运行中环境的新标签页指纹 |
| `GET /profile/start?profile=<ID>` | 启动已有环境，返回端口、WS 和 PID |
| `GET /profile/stop?profile=<ID>` | 停止环境 |
| `GET /list`、`GET /status` | 环境列表及运行状态 |

```powershell
$config = Get-Content -LiteralPath '.\examples\native-profile.json' -Raw
$result = Invoke-RestMethod -Method Post -ContentType 'application/json' `
  -Uri 'http://127.0.0.1:25325/profile/launch-json' -Body $config
$result.profileId
$result.data.wsUrl
Invoke-RestMethod -Method Post `
  -Uri ('http://127.0.0.1:25325/profile/verify?profile=' + $result.profileId)
```

## 8. 指纹库数量和已验证范围

`library.json` 当前保留 **4 条去重后的配置快照**，都引用 `configID=3285798`，所以本地可确认的基础配置 ID 仍为 **1 个**。原来 10 条记录归一后只有 3 套；本机历史库补入一条有效变体：洛杉矶时区、UA 使用 Chrome 缩减版本 `153.0.0.0`，完整 UA-CH 版本仍保存在 `configMeta`。按 `data` 与 `configMeta` 规范化内容去重后共 4 套；克隆名称和记录 ID 不参与比较，PID、调试端口、DevTools 链接、备注、起始页和公开 RTC 字段不打包，嵌套 JSON 字符串在比较前先解析。每条快照的 `fingerprintHash` 是规范化配置的 SHA-256。3 个 `.bin` 是不同序列化布局/版本的模板，不是额外 JSON 配置，也不能当成新的上游配置 ID。

本包对原 `data` 去掉了 `pid`、`debugPort`、`devToolsLink`、`note`、`startPage`、`publicRTC` 六个运行/环境字段，并修正了空的 UA 版本。`configMeta` 完整保存，其中 JSON 字符串被展开为对象。

统计口径见 `fingerprints/field-index.json`：每个快照 data 顶层 40 个键，configMeta 顶层 21 个键，additional_info 顶层 7 个键，UA-CH 对象顶层 11 个键，WebGL 参数顶层 72 个键。全部快照的规范化叶子路径去重后为 **190**，数组标量值按一个数组字段计数。旧报告的 203 使用了不同数组展开口径并含运行字段，不能解释成 203 套指纹或 203 个已验证原生开关。

| 类型 | 当前处理 |
|---|---|
| UA、appVersion、CPU/内存、屏幕/DPR/色深、时区、语言、WebGL Vendor/Renderer | 写入已恢复的原生槽；实际浏览器检查通过 |
| UA-CH 品牌、完整版本、平台版本、架构、bitness 等 | 每个页面会话通过 CDP 下发；新标签页检查通过 |
| GPU architecture/vendor、quota、heap limit、已恢复的四个画布参数与 acc | 写入已恢复槽；不是完整图形/噪声模型生成器 |
| 字体列表、媒体设备、语音列表、WebGL 参数矩阵、未知原生开关 | 继承二进制模板；JSON 中保留，暂未实现任意数组重建或全部开关写入 |
| `maxTouchPoints`、DNT、任意 noiseSeed 等 Ghost 通用字段 | 原生模式未映射；不声称这些 GUI 字段已原生生效 |
| WebRTC disabled 的额外 Chromium 参数 | 限制非代理 UDP，不等于移除全部 WebRTC API |
| macOS/Linux 原生模板 | 当前未验证，原生模式拒绝使用 |

该范围表也解释了为什么只带 JSON 不足以重建所有指纹：未知二进制字段仍需通过兼容模板保留。

## 9. 上游来源：本次校正的事实

本包的 Ghost 启动器只读本地指纹文件，不向 Undetectable 的指纹服务器请求配置。浏览器正常网页请求与其自身网络行为不属于指纹库下载。

对照备份主程序发现，`Undetectable.original.exe` 的配置 URL 是 `api.undetectnews.com` 下的 `/v2/configs`、`/v2/free-configs`、`/cnf/configs` 及其 `/%1` 变体；当前安装主程序对应位置已经替换为 `shops.chestchain.com`。二者不是相同哈希的二进制，所以不能凭前序请求就把 chestchain 域名认定为厂商官方镜像。

前序从 chestchain 地址看到的 3 条目录只是当次响应范围。原备份域名返回 401，也只能说明该次请求未获授权，不能据此断言 token 一定过期，更不能计算厂商全量指纹池规模。当前可证明的是：本地实际配置 ID 有 1 个；厂商全量可下载配置数量仍未确认。

直接字节比对、两个主程序哈希及 URL 列表存放于 `reference/host-manifest.json` 和 `evidence/host-config-endpoints.json`。旧 REA Evidence 内含它当时分析对象的独立 SHA-256；引用前必须核对对象哈希，不能把旧结果当作本次主程序快照的同一文件证据。

## 10. 证据、源码与复现

E1：`evidence/fn_ini.asm` 的 IniConnector 读取 `settings.json` / `profilesPath`；`evidence/inspect_data_root.py` 对 `0x1832fad60` 的只读反汇编确认系统目录入口。

E2：`evidence/fn_readprofile.asm` 对 `chrome.dll` 中 `0x1832fd110` 的读取路径分析；`internal/undetectable/startinfo.go` 为恢复的编解码实现。

E3：`evidence/json-launch-verification.jsonl` 为独立启动器实际启动和新标签页指纹读回结果，匹配项包含 UA、appVersion、品牌/完整版本、CPU/内存、语言、时区、屏幕和 WebGL。

E4：`evidence/ghost-integration.txt` 为 GUI 共用的 App/Store 路径集成记录。A、A2、B 三个模板都通过了 JSON 导入/导出、未知 metadata 保留、重复启动复用、跨进程启动锁，以及修改 CPU、内存、屏幕、DPR、时区、语言后的新标签页比对。

构建独立启动器：

```powershell
Set-Location 'C:\Users\dhl\Desktop\ghostbrowser'
go build -o undetectable-kit/bin/ghost-launcher.exe ./cmd/ghost-launcher
```

可单独复制 `undetectable-kit`。在 `source` 内使用随包 go.mod 构建：

```powershell
Set-Location '.\source'
go build -o ..\bin\ghost-launcher.exe ./cmd/ghost-launcher
```

构建 GUI：安装满足 `go.mod` 版本要求的 Go、Node.js/npm 和 Wails CLI v2.16.0。项目根目录执行：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
Set-Location 'C:\Users\dhl\Desktop\ghostbrowser'
go mod download
wails build -platform windows/amd64
```

Wails 会按 `wails.json` 调用前端安装和构建命令；如果提示找不到 `node` 或 `npm`，先将 Node.js 安装目录加入当前 PowerShell 的 `PATH`。GUI 输出至 `build/bin/GhostBrowser.exe`。发布时把 `undetectable-kit` 文件夹和 EXE 放在同一目录；设置页默认路径指向随包的 `runtime/chrome/chrome.exe`。用户可用「选择 EXE」指定其它浏览器程序并保存，选择「恢复默认」会回到当前模式自动检测出的浏览器路径。定制指纹模式要求兼容的定制 Chromium；普通 Chrome / Edge 仅用于 stock 模式。本包 `bin/GhostBrowser.exe` 是构建副本。

宝塔面板 + Nginx 部署 PHP 服务端无需编译，直接上传 `cloud-server/`。详细步骤和可合并的站点配置片段见 [PHP 云同步部署文档](cloud-server/README.zh-CN.md) 与 `cloud-server/deploy/baota-nginx.conf.example`。

重复本次实际浏览器检查：

```powershell
$env:GHOST_NATIVE_TEST = '1'
go test -run TestNativeJSONIntegration -v -count=1 .
```

如果浏览器退出，先看启动器报出的退出码，并检查：模板是否兼容、原生 settings.json 是否可解析、profilesPath 是否为可写绝对路径、运行 DLL 是否完整、CDP 端口是否被占用。附带文件锁错误表示同一 Ghost 环境已经有启动者。
