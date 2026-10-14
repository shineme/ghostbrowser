# GhostBrowser 云同步服务端（单文件 PHP）

整个服务端只有一个文件：`ghostsync.php`。

- 不需要伪静态 / URL 重写，不需要数据库（不需要 SQLite / MySQL）
- 支持 PHP 7.4 及以上版本，只用 PHP 默认自带的功能
- 宝塔、虚拟主机、Nginx、Apache、IIS、`php -S` 都能直接用

## 部署（3 步）

1. 把 `ghostsync.php` 上传到网站任意目录，例如 `https://你的域名/ghostsync.php`。
2. 用浏览器打开这个地址。页面会生成一个随机令牌；用编辑器打开 `ghostsync.php`，把顶部的
   `const GHOST_TOKEN = '';` 改成这个令牌，然后保存。
3. 在 GhostBrowser 中打开「设置 → 云同步」，按下面填写，再点「测试连接」：

| 设置项 | 填写 |
|---|---|
| 同步服务器地址 | `https://你的域名/ghostsync.php`（填完整的 .php 地址） |
| 访问令牌 | 第 2 步填入的令牌 |
| 同步密码 | 自己设一个加密密码。每台电脑都要填同一个；这个密码不会发送到服务器 |

## 上传大小（增量同步）

新版客户端按文件切成最大 4 MB 的加密块上传，只传服务器上还没有的块：

- 首次上传是完整的环境，之后关闭浏览器通常只传几 MB
- 每次请求最大约 12 MB，PHP 默认的 `post_max_size = 8M` 也能用（客户端会自动按服务器上限缩小批次）
- 单个环境最大 2 GB（不含缓存）
- 如果遇到 413，把 Nginx 的 `client_max_body_size` 设为 `20m` 以上，PHP 的 `post_max_size` 设为 `16M` 以上
- 旧版客户端仍可用整包方式同步（最大约 256 MB，需要 `post_max_size` 和 `client_max_body_size` 约 `300M`）

## 多用户

在文件顶部的 `GHOST_USERS` 里加一行 `'令牌' => '名字'`，就多了一个用户：

- 每个用户的数据放在各自独立的目录，互相看不到
- 每个令牌至少 32 个字符
- 用户各自在客户端填自己的令牌和自己的同步密码

- 也可以用环境变量 `GHOST_SYNC_USERS="令牌1=名字1;令牌2=名字2"` 添加用户（与 `GHOST_USERS` 合并）
- 主令牌的用户名是 `owner`，环境分享时用这个名字指定接收者

## 设备管理、恢复密钥与环境分享

新版服务端额外提供（`ping` 的 `features` 里出现 `devices`、`keys`、`directory`、`shares`），旧客户端不受影响：

- **设备管理**：每个请求带 `X-Device-ID`，服务端在用户目录的 `devices.json` 里登记设备。账号的第一台设备自动批准；客户端可开启「新设备需要批准」（默认值见 `GHOST_REQUIRE_APPROVAL`）。被吊销的设备所有请求返回 403；待批准的设备除 `ping`、`devices/self` 外返回 403。注意：持有令牌的人可以换一个设备 ID 重新登记，所以设备丢失时请开启「新设备需要批准」或更换令牌
- **恢复密钥**：客户端把加密后的“密钥包”存到用户目录的 `keys.json`（`GET/PUT /api/keys`，PUT 带读到的 `epoch` 做并发保护）。密钥包里只有用同步密码和恢复密钥分别加密的主密钥，服务器无法解开
- **环境分享**：共享目录 `_shared/`（在主数据目录下）保存公钥目录 `directory.json`、分享记录 `shares.json` 和加密分享包 `shares/*.gbe`。分享包用接收者公钥加密，服务端只校验格式（GBE1）。分享结束（撤销 / 拒绝 / 退出 / 过期）后分享包立即删除，记录 30 天后清理。单个分享包上限 `GHOST_SHARE_MAX_MB`（默认 256，环境变量 `GHOST_SYNC_SHARE_MAX_MB`），同时受 `post_max_size` 和 nginx `client_max_body_size` 限制；包含登录状态的大环境建议调大，或改用「仅配置」分享

## 管理页（只看元数据）

在文件顶部把 `GHOST_ADMIN_TOKEN` 设为 32 位以上的随机字符串（或环境变量 `GHOST_SYNC_ADMIN_TOKEN`），然后用浏览器打开 `https://你的域名/ghostsync.php?r=/admin`，输入管理令牌即可（登录状态 12 小时，Cookie 为 HttpOnly + SameSite=Strict）。

- 显示：用户、存储占用、云端环境数、设备数（已批准 / 待批准 / 已吊销）、是否设置密钥、分享数、最近活动、最近 40 条错误（4xx/5xx，不含 404）
- 从不显示：令牌、密码、密钥、环境名称或任何解密内容
- 输错管理令牌同样计入防暴力破解
- 未设置管理令牌时管理页处于关闭状态

## 防暴力破解

同一个 IP 在 15 分钟内输错令牌 10 次（`GHOST_MAX_FAILS`），这个 IP 会被封 15 分钟，返回 429。计数保存在 `.ghostsync-guard` 目录。

## 版本历史

- 每个环境默认保留 3 个版本（`GHOST_KEEP_VERSIONS`，或环境变量 `GHOST_SYNC_KEEP_VERSIONS`，最多 50）
- 客户端「云同步 → 历史版本」可以把任意一个保留的版本恢复到本机，并作为最新版本上传
- 不再被任何版本引用的数据块，会在 2 天后自动清理

## 数据存放位置

数据默认存放在 `ghostsync.php` 旁边一个隐藏目录中，目录名形如 `.ghostsync-<20位随机字符>`：

- 目录名由令牌计算得出，不知道令牌就猜不到
- 目录里会自动写入 `.htaccess` 和 `web.config`，禁止网页直接访问
- 用 Nginx 时，建议再加一条：`location ~ /\.ghostsync- { deny all; }`
- 也可以把 `GHOST_DATA_DIR` 改到网站根目录以外的路径
- 上传的内容是客户端用 AES-256-GCM 加密后的数据，服务器上没有明文
- 每个环境默认保留最近 3 个版本（`GHOST_KEEP_VERSIONS`），更早的版本会自动清理

## 同步流程（客户端）

- **启动软件时**：先对比云端和本地
  - 本地才有的环境，或本地有改动的环境：直接上传
  - 本地没有、云端有的环境：直接下载
  - 云端比本地新：弹窗询问是否拉取云端版本覆盖本地，默认倒计时 3 秒后执行默认动作
  - 倒计时秒数和默认动作可以在「设置 → 云同步」里修改；设为 0 表示不弹窗，直接执行默认动作
  - 选择「保留本地」时，会用本地版本覆盖云端版本
- **每次关闭浏览器后**：自动上传该环境，登录状态也会一起同步
  - Chrome 的 Cookie 文件是按 Windows 用户加密的，换一台电脑就解不开。所以软件会在浏览器运行期间把 Cookie 另存一份快照，并放进加密包一起上传
  - 在另一台电脑上启动这个环境时，软件会自动把快照中的 Cookie 写回浏览器
- **软件全局数据**：设置、代理管理列表、自定义指纹库（library.json）作为一条特殊记录 `_global` 加密同步
  - 改动后约 20 秒内自动上传；退出软件时也会上传一次
  - 拉取时保留本机自己的：设备名和设备 ID、同步地址/令牌/密码、环境存放目录；云端的内核路径在本机不存在时，也保留本机的
  - 代理的检测结果（延迟、出口 IP 等）会跟着同步，但只是检测结果变化不会触发上传
- **启动某个环境前**：如果云端版本更新，并且本地没有未上传的改动，会先拉取云端版本

## 接口

两种写法都可以：`ghostsync.php?r=/api/...` 或 `ghostsync.php/api/...`。

- 鉴权：`Authorization: Bearer <令牌>`
  - 部分主机会把 `Authorization` 请求头过滤掉，所以客户端还会同时发送 `X-Ghost-Token`
- 接口与旧版服务端保持一致：
  - `ping`
  - `profiles`
  - `lock` / `unlock`
  - `blob`（PUT 上传、GET 下载）
  - `commit`
  - `delete`：留下“墓碑”（只有名称、删除时间和设备，不含数据），数据、历史版本和数据块引用随即释放；其他设备看到墓碑会删除自己的副本。墓碑默认保留 90 天（`GHOST_TOMBSTONE_DAYS` 或环境变量 `GHOST_SYNC_TOMBSTONE_DAYS`）。另一台设备正在上传时返回 423，可带 `{"force": true}` 强制删除
  - `profiles?tombstones=1` 同时列出墓碑；不带参数（旧客户端）时不列出。对已删除的环境上传 / 下载返回 410，旧设备无法把它“复活”
  - `history`：列出保留的版本；`blob?version=N` 下载某个旧版本
  - `refs`：上传本版本引用的数据块列表（用于清理）
  - `chunks/missing`、`chunks`（批量上传）、`chunks/get`（批量下载）
- `ping` 会返回 `features`（含 `tombstones`）、`tombstoneDays`、`keepVersions`、`maxBody`、`usedBytes`、`freeBytes`
- 版本冲突返回 412
- 设备、密钥、公钥目录、分享接口见上文「设备管理、恢复密钥与环境分享」，完整列表在 `ghostsync.php` 文件头注释和项目 README 的 Sync server contract

## 在线更新与快捷入口（静态文件）

以下接口同样需要令牌（开启「设备批准」时还要求设备已获批准），只是读取 `ghostsync.php` 同目录下的静态文件，不涉及加密数据：

- `GET ?r=/api/update/latest`：返回 `update/latest.json`；没有发布时返回 404
- `GET ?r=/api/update/file/<文件名>`：下载 `update/<文件名>`（只允许字母、数字、`.`、`_`、`-`）
- `GET ?r=/api/catalog`：返回 `catalog.json`（网站快捷入口 + 公告）；文件不存在时返回空列表

目录结构：

```
ghostsync.php
catalog.json              ← 可选，示例见同目录
update/
  latest.json             ← sign-release 生成
  GhostBrowser-0.8.0.exe
```

`latest.json`：`{"version","notes","url","sha256","sig"}`。`url` 写文件名时从 `update/` 下载，也可以写完整的 https 地址（例如 CDN）。客户端会校验 SHA-256 和 Ed25519 签名（公钥内置在客户端，私钥只在维护者手里），服务器被入侵也无法推送恶意更新。

`catalog.json`：

```json
{
  "revision": 1,
  "groups": [{ "id": "ai", "name": "AI", "sites": [{ "id": "chatgpt", "name": "ChatGPT", "url": "https://chatgpt.com/", "icon": "" }] }],
  "announcements": [{ "id": "notice-1", "title": "标题", "body": "支持 **Markdown**", "level": "info", "start": "2026-10-01T00:00:00+08:00", "end": "" }]
}
```

- 网站只接受 http / https 地址；`icon` 可留空（客户端显示首字母）
- 公告 `level`：`info` / `warn` / `critical`；`start` / `end` 可写时间字符串或 Unix 秒，留空表示不限
- 改 `id` 会被当作新公告（重新显示未读）

