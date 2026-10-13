# GhostBrowser 云同步 PHP 服务端

这里补齐了桌面客户端此前缺少的服务端。服务端不需要 Undetectable 主程序、Chromium、Go 或 Composer，只需要 PHP 8.2 及以上版本和 PDO SQLite。部署文件包含 `public/index.php`、`server.php`、配置示例、Apache 路由、Nginx 示例和本机启动脚本。

本次已经使用本机 PHP 8.5.11 与实际 SQLite 完成 Go 客户端联调。PHP 工具安装于 `C:\Users\dhl\Tools\php-8.5.11`，下载包来自 [PHP 官方 Windows 下载页面](https://www.php.net/downloads.php?os=windows)，下载后核对了官方 SHA-256。PHP 工具本身不需要上传到 Linux VPS。

## 1. 文件对应关系

```text
cloud-server/
  public/
    index.php                 唯一 HTTP 入口，Web 根目录设为此 public 文件夹
    .htaccess                 Apache 路由及 Authorization 转发
  server.php                  鉴权、数据库、锁、上传、下载、版本提交逻辑
  config.example.php          复制为 config.php 并填随机令牌
  router.php                  PHP 内置开发服务器路由
  start-local.ps1             Windows 本机开发启动脚本
  deploy/nginx.conf.example   Nginx + PHP-FPM 配置示例
  README.zh-CN.md             本文
  data/                      运行时自动生成，位于 public 外
    sync.sqlite               SQLite 环境索引与锁状态
    sync.sqlite-wal/-shm      SQLite WAL 文件，运行时可能存在
    blobs/<sha256>.gbe        已提交的加密数据包
    tmp/<随机名>.part          未提交上传的临时文件
```

项目根目录的 `cloud-server` 是服务端源码。交付包中的 `undetectable-kit/cloud-server` 放置同一份文件，便于一起复制给服务器。

## 2. 本机启动

先生成随机令牌，再保持终端运行：

```powershell
Set-Location 'C:\Users\dhl\Desktop\ghostbrowser\cloud-server'
$php = 'C:\Users\dhl\Tools\php-8.5.11\php.exe'
$env:GHOST_SYNC_TOKEN = & $php -r 'echo bin2hex(random_bytes(32));'
$env:GHOST_SYNC_DATA_DIR = Join-Path $PWD 'data'
# 将上面生成的令牌保存到你自己的密码管理器或客户端设置中。
Write-Output $env:GHOST_SYNC_TOKEN
.\start-local.ps1 -PhpPath $php -Port 8787
```

该脚本只监听 `127.0.0.1`，用于本机测试。PHP 内置服务器用于开发；VPS 部署使用下面的 Nginx/PHP-FPM 或 Apache。

新版 GhostBrowser 中填写：

| 设置项 | 本机示例 |
|---|---|
| 同步服务器地址 | `http://127.0.0.1:8787`，不要加 `/api` |
| 访问令牌 | 上面生成的 `GHOST_SYNC_TOKEN` |
| 同步密码 | 你自己选择的加密密码，不是访问令牌 |
| 设备名 | 当前电脑名称 |

点击「测试连接」，然后进入「云端同步」。上传和拉取前需停止环境。服务器上存在、本机还没有的环境也会列出，可以直接拉取；新客户端会恢复完整 Profile JSON，包括原生指纹配置、代理配置及环境名称。

Windows 服务端运行数据默认保存到此目录的 `data`；可通过环境变量改到其它位置。同步密码不会发送给 PHP。

## 3. 宝塔面板 + Nginx 部署

下面以目录 `/www/server/ghostbrowser-sync`、PHP 8.2 为例。路径和 PHP-FPM socket 需要按你的站点实际值调整。宝塔官方文档说明可在「网站 -> 站点设置 -> PHP」为站点选择运行版本；修改前先备份现有 Nginx 配置。[宝塔站点 PHP 配置](https://docs.bt.cn/user-guide/site/php/site-config/php/)

1. 在宝塔「软件商店」安装 Nginx 和仍受支持的 PHP 版本（例如 PHP 8.2/8.3），在该 PHP 版本的「设置 -> 安装扩展」中启用 SQLite/PDO SQLite 扩展。然后到「网站 -> 添加站点」创建域名站点，并在站点设置里选同一个 PHP 版本。生产环境分支支持周期见 [PHP 官方支持版本](https://www.php.net/supported-versions.php)。
2. 把整个 `cloud-server` 目录上传至 `/www/server/ghostbrowser-sync`，**不要**放进公开网站目录。站点运行目录 / Nginx `root` 指向 `/www/server/ghostbrowser-sync/public`，这样 `server.php`、`config.php` 和 `data` 位于网站根目录之外。
3. 在宝塔终端中创建配置和数据目录：

```bash
cd /www/server/ghostbrowser-sync
cp config.example.php config.php
mkdir -p data
chmod 700 data
php -r 'echo bin2hex(random_bytes(32)), PHP_EOL;'
```

把最后一条命令输出的随机令牌填进 `config.php` 的 `token` 项。宝塔默认 PHP-FPM 用户通常是 `www`，将数据目录授权给它：

```bash
chown -R www:www /www/server/ghostbrowser-sync/data
chown root:www /www/server/ghostbrowser-sync/config.php
chmod 640 /www/server/ghostbrowser-sync/config.php
```

如果 PHP-FPM 池使用其他用户/组，以实际 PHP 设置为准。检查所选 PHP 版本是否加载 SQLite：

```bash
/www/server/php/82/bin/php -m | grep -Ei '^(PDO|pdo_sqlite|sqlite3)$'
```

命令里的 `82` 要与安装的 PHP 版本对应。输出至少包含 `PDO` 和 `pdo_sqlite`；CLI 和 PHP-FPM 使用同一版本时检查才有参考意义。

4. 在「网站 -> 站点设置 -> 配置文件」中合并 [宝塔 Nginx 配置片段](deploy/baota-nginx.conf.example)。先把示例 `root` 改成你的 `.../cloud-server/public`，并把 `fastcgi_pass` 的 socket 后缀改为站点 PHP 版本（常见 PHP 8.2 为 `/tmp/php-cgi-82.sock`）。如果面板已生成 `location /` 和 PHP 处理块，编辑现有块，不要添加重复 `location`。重点保留：

   - `try_files $uri /index.php?$query_string;`，将 `/api/*` 转发给 PHP 入口。
   - `fastcgi_param HTTP_AUTHORIZATION $http_authorization;`，将 Bearer 令牌传给 PHP。
   - `client_max_body_size 260m;` 和 `fastcgi_read_timeout 300s;`，允许同步大包。

5. 保存后在宝塔 Nginx 服务中执行配置检查/重载，为域名配置 HTTPS。用下列命令验证入口（替换域名和令牌）：

```bash
curl -i -H 'Authorization: Bearer 你的随机令牌' https://sync.example.com/api/ping
```

应返回 HTTP 200 和 `{"ok":true,"service":"ghostbrowser-sync","protocol":1}`。未带令牌访问返回 401 是预期行为。客户端设置中填写 `https://sync.example.com`，不要在地址末尾加 `/api`；访问令牌填 `config.php` 中的值，同步密码则是你自行设置的另一串密码。

Nginx 完整服务器示例在 `deploy/nginx.conf.example`；它适合自行管理 Nginx 的 VPS。宝塔站点优先合并 `deploy/baota-nginx.conf.example`，避免覆盖面板生成的 SSL、日志和 PHP-FPM 设置。

Apache 部署同样把 DocumentRoot 指向 `public`，启用 mod_rewrite，并允许此目录的 `.htaccess` 生效。无需把整个项目目录设置成公开 Web 根目录。

```bash
mkdir -p /var/www/ghostbrowser-sync/data
chown -R www-data:www-data /var/www/ghostbrowser-sync/data
chmod 700 /var/www/ghostbrowser-sync/data
```

5. 使用 `deploy/nginx.conf.example`，替换域名、证书路径、项目路径和当前 PHP-FPM socket。宝塔用户通常需要改成宝塔实际的 PHP socket，并保留 `Authorization` 转发和 `/api/*` 转到 index.php 的规则。
6. 为站点配置 HTTPS，然后客户端填入 `https://sync.你的域名`。两台电脑使用同一令牌和同一同步密码。

生成令牌：

```bash
php -r 'echo bin2hex(random_bytes(32)), PHP_EOL;'
php -m
```

`php -m` 中应包含 `PDO`、`pdo_sqlite`。SQLite 驱动说明见 [PHP PDO_SQLITE 文档](https://www.php.net/manual/en/ref.pdo-sqlite.php)。

Apache 部署同样把 DocumentRoot 指向 `public`，启用 mod_rewrite，并允许此目录的 `.htaccess` 生效。无需把整个项目目录设置成公开 Web 根目录。

## 4. 配置文件和环境变量

默认配置：

```php
<?php
return [
    'token' => '填写自己生成的随机令牌',
    'data_dir' => __DIR__ . '/data',
    'max_blob_bytes' => 268435456,
    'lease_seconds' => 600,
    'base_path' => '',
];
```

实际使用时填入至少 32 字符的随机令牌；留空会返回 503，服务端没有默认通用密码。环境变量非空时覆盖同名文件配置：

| 环境变量 | 默认值/作用 |
|---|---|
| `GHOST_SYNC_TOKEN` | 无默认值，Bearer 访问令牌 |
| `GHOST_SYNC_DATA_DIR` | `cloud-server/data`，必须位于 public 外 |
| `GHOST_SYNC_MAX_BLOB_BYTES` | 268435456，即 256 MiB |
| `GHOST_SYNC_LEASE_SECONDS` | 600 秒；限定为 60–3600 秒 |
| `GHOST_SYNC_BASE_PATH` | 默认空；子目录部署可设 `/ghostsync` |

当前桌面客户端同步包上限同样为 256 MiB；只调高 PHP 限制不会改变客户端限制。Nginx 示例允许 260 MiB 请求体，为包头等留出余量。客户端网络超时为 5 分钟。

子目录部署时，客户端地址填 `https://域名/ghostsync`，PHP 的 `base_path` 填 `/ghostsync`，Web 服务器也必须把此前缀转给本服务入口。单独域名部署无需设置前缀。

## 5. API 协议

所有请求都需要 `Authorization: Bearer <token>`。Profile ID 仅接受 1–64 位字母、数字、下划线或连字符。

| 接口 | 请求/响应 |
|---|---|
| `GET /api/ping` | `{"ok":true,"service":"ghostbrowser-sync","protocol":1}` |
| `GET /api/profiles` | 数组；包含 id、name、version、sha256、meta、lockedBy、updatedAt |
| `POST /api/profiles/{id}/lock` | 请求 `{"device":"设备ID"}`，返回 lockToken、version、expiresAt |
| `PUT /api/profiles/{id}/blob?version=N` | 二进制 GBE1 加密包，成功返回摘要与长度 |
| `POST /api/profiles/{id}/commit` | 提交 id、name、version=N+1、sha256、meta |
| `POST /api/profiles/{id}/unlock` | 失败时释放自己的租约及临时上传 |
| `GET /api/profiles/{id}/blob` | 最新已提交加密包；含 `X-Profile-Version`、`X-Blob-SHA256` |

取得租约后的 PUT、commit、unlock 请求额外携带 `X-Device-ID` 与 `X-Lock-Token`。服务端将租约令牌的哈希存入 SQLite。另一个同步操作在有效租约期间取得同一环境锁时返回 423。

上传不是直接覆盖正式文件：PHP 先保存到 tmp，验证基准版本、包头、长度与 SHA-256；commit 通过后才更新 SQLite 当前版本并释放锁。412 表示版本冲突，409 表示上传与提交摘要不匹配，413 表示超过大小上限。下载始终读取已提交版本，因此中断上传不会替换原云端环境。

已提交包按摘要存放，旧包保留，便于备份和恢复。此版本未加入远端删除、版本历史浏览和自动垃圾清理功能。

## 6. 加密与数据恢复

新版客户端将完整环境信息写入 ZIP 的 `ghost-profile.json`，与浏览器文件一起加密。完整指纹、代理密码、待导入 Cookies 等不会再出现在明文 commit metadata 中。服务器能看到环境 ID、名称、版本、摘要、设备 ID 和大小，不能看到加密包里的内容。

格式保持为：

```text
GBE1（4 字节）
salt（16 字节）
nonce（12 字节）
AES-256-GCM ciphertext + tag（tag 为 16 字节）
```

密钥由同步密码和随机 salt 经 PBKDF2-HMAC-SHA256、120000 次迭代派生。每次上传生成新的 salt 和 nonce。PHP 不持有同步密码，也不尝试解密。

上传时排除缓存、生成的指纹扩展、Crashpad、运行时 launch.json 和锁文件。若代理配置原本引用本地代理 ID，会将实际代理记录放入加密元数据，方便另一台机器恢复。

拉取先检查下载摘要并完成 GCM 解密，在独立临时目录解包，检查路径与 Profile ID，然后替换环境目录。错误密码、损坏数据或不合法 ZIP 不会先删除现有环境。已有环境保留为 profiles 根目录下的 `.before-pull-<ID>-<随机串>` 备份目录。

浏览器 Cookies 数据库可能使用绑定本机的系统加密；云端加密封包不会改变该绑定。需要跨机器恢复登录态时，应使用客户端的 Cookies JSON 导出/导入能力，而不能保证直接复制 Chromium 数据库即可在其它机器解密。

此前客户端上传的 ZIP 如果没有 `ghost-profile.json`，新客户端会提示重新上传，避免只恢复数据文件却丢失指纹与环境配置。使用本服务应使用此次更新后的 GhostBrowser.exe，旧客户端缺少租约头和提交结果检查。

## 7. 本次联调结果与复现

检查使用独立临时数据目录及真实 PHP/SQLite：

- Bearer 鉴权成功与失败。
- 上传、列出、下载；完整环境、代理记录和浏览器文件往返恢复。
- 服务器索引及文件中不出现测试 Cookie/代理密码明文。
- 第二台设备的旧版本上传被拒绝，拉取后版本更新。
- 有效租约被占用时拒绝另一个操作；错误版本 PUT 返回 412。
- 错误提交摘要被拒绝，原版本保持不变。
- 服务端 commit 返回 500 时，客户端不推进版本，并释放租约。
- 错误密码和损坏下载不能破坏本机已有数据。
- 原生浏览器三个模板的集成检查仍然通过。

在项目根目录复现：

```powershell
$env:GHOST_PHP_TEST = 'C:\Users\dhl\Tools\php-8.5.11\php.exe'
$env:GHOST_NATIVE_TEST = '1'
go test -run 'TestPHPSyncRoundTrip|TestNativeJSONIntegration' -v -count=1 .
```

测试记录存放于 `undetectable-kit/evidence/php-sync-integration.txt`。Windows 测试为 PHP 自动启用 ext 目录里的 PDO SQLite，VPS 的 PHP-FPM 需自行在实际运行配置中启用扩展。

## 8. 备份与常见错误

备份内容是 `config.php` 和整个 `data`，并在自己的密码管理器保存同步密码。SQLite 采用 WAL；做静态文件备份时先停 PHP-FPM 写入或使用 SQLite 一致性备份工具，不要只复制主数据库文件而漏掉未合并的 WAL。

| 情况 | 处理 |
|---|---|
| 401 | 检查客户端令牌，以及 Nginx/Apache 是否转发 Authorization |
| 503 token 未配置 | 设置 config.php 或 GHOST_SYNC_TOKEN |
| 503 PDO SQLite 未启用 | 在实际 PHP-FPM 的扩展配置中启用 pdo_sqlite |
| 404 /api 路由 | 检查 Web 根目录与重写规则，客户端地址不要加 /api |
| 412/版本冲突 | 先拉取云端最新环境；本机原环境会保留备份 |
| 423/锁占用 | 等正在进行的操作完成；崩溃遗留租约到期后可重试 |
| 413 | 环境包超过上限，检查缓存排除情况与服务器限制 |
| 解密失败 | 两台电脑必须使用相同同步密码 |
| 无法替换环境目录 | 关闭浏览器及占用该目录的其它程序再拉取 |
| 500 | 查看 PHP 错误日志，检查 data 目录权限及 SQLite/Disk 状态 |
