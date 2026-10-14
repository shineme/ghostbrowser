<?php
/**
 * GhostBrowser sync server: one PHP file, no database, no URL rewrite.
 *
 *  1. Put this file anywhere PHP runs (宝塔 / any host / php -S).
 *  2. Set GHOST_TOKEN below to a random string of 32+ characters
 *     (open the file in a browser once: it suggests one).
 *  3. In GhostBrowser → Settings → Cloud sync, set the server to the file URL,
 *     e.g. https://example.com/ghostsync.php, and the same token.
 *
 * Profiles are encrypted on the client (AES-256-GCM) before upload; this
 * server only ever stores opaque "GBE1" blobs plus a small JSON index.
 * Requires PHP 7.4+ with no extensions beyond the defaults.
 *
 * Routes (either ghostsync.php?r=/api/... or ghostsync.php/api/...):
 *   GET  /api/ping
 *   GET  /api/profiles[?tombstones=1]      (tombstones=1 also lists deleted profiles)
 *   POST /api/profiles/{id}/lock      {"device": "...", "restore": false}  (410 for a deleted profile unless restore)
 *   POST /api/profiles/{id}/unlock
 *   PUT  /api/profiles/{id}/blob?version=N   (body = encrypted blob)
 *   POST /api/profiles/{id}/commit    {"id","name","version","sha256","meta"}
 *   GET  /api/profiles/{id}/blob[?version=N]   (N = current or a kept older version)
 *   GET  /api/profiles/{id}/history
 *   PUT  /api/profiles/{id}/refs      (under lease: newline-separated chunk IDs of this upload)
 *   POST /api/profiles/{id}/delete    {"force": false} -> leaves a tombstone so other devices delete their copy
 *   POST /api/chunks/missing          {"ids": [...]}  -> {"missing": [...]}
 *   PUT  /api/chunks                  framed: (64-hex id, uint32 BE length, GBC1 data)*
 *   POST /api/chunks/get              {"ids": [...]}  -> framed, same layout
 *
 * Accounts & security (feature "devices", "keys", "directory", "shares"; see the block near the end):
 *   GET  /api/devices | /api/devices/self ; POST /api/devices/{id}/approve|revoke|forget ; POST /api/devices/settings
 *   GET|PUT /api/keys                 (opaque, client-encrypted key envelope; PUT is compare-and-swap on "epoch")
 *   GET|POST /api/directory[/{user}]  (X25519 public keys of the users on this server)
 *   GET|POST /api/shares, PUT|GET /api/shares/{sid}/blob, POST /api/shares/{sid}/accept|decline|leave|revoke|extend
 *   GET|POST ?r=/admin                (metadata-only admin page, needs GHOST_ADMIN_TOKEN)
 */

// ------------------------------------------------------------- settings
const GHOST_TOKEN = '';                 // 32+ random characters (or env GHOST_SYNC_TOKEN)
const GHOST_DATA_DIR = '';              // '' = a hidden folder next to this file (or env GHOST_SYNC_DATA_DIR)
const GHOST_MAX_BLOB_MB = 512;          // largest profile upload
const GHOST_LEASE_SECONDS = 600;        // how long an upload lock lives
const GHOST_KEEP_VERSIONS = 3;          // versions kept per profile (current + older ones you can restore)
const GHOST_USERS = [                   // extra users, each with an isolated data folder: 'token' => 'name'
    // 'another-random-token-of-32-or-more-characters' => 'alice',
];
const GHOST_MAX_FAILS = 10;             // wrong tokens per IP within 15 minutes before a 15-minute block
const GHOST_TOMBSTONE_DAYS = 90;        // how long a deleted profile's tombstone is kept so every device learns about it
const GHOST_ADMIN_TOKEN = '';           // 32+ characters enables the metadata-only admin page ?r=/admin (or env GHOST_SYNC_ADMIN_TOKEN)
const GHOST_REQUIRE_APPROVAL = false;   // default for "new devices need approval" (each user can change it in the app)
const GHOST_SHARE_MAX_MB = 256;         // largest shared-profile upload
// ----------------------------------------------------------------------

error_reporting(E_ALL);
ini_set('display_errors', '0');
@set_time_limit(0);

final class GsFail extends Exception
{
    public $status;
    public function __construct(int $status, string $message) { parent::__construct($message); $this->status = $status; }
}

function gs_reply(int $status, $value): void
{
    http_response_code($status);
    header('Content-Type: application/json; charset=utf-8');
    header('Cache-Control: no-store');
    header('X-Content-Type-Options: nosniff');
    echo json_encode($value, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function gs_cfg(): array
{
    $env = function (string $k, $def) { $v = getenv($k); return ($v === false || $v === '') ? $def : $v; };
    $token = (string)$env('GHOST_SYNC_TOKEN', GHOST_TOKEN);
    $dir = (string)$env('GHOST_SYNC_DATA_DIR', GHOST_DATA_DIR);
    if ($dir === '') {
        // Unguessable without the token, and denied by .htaccess / web.config.
        $dir = __DIR__ . '/.ghostsync-' . substr(hash('sha256', 'ghostsync|' . $token), 0, 20);
    }
    return [
        'token' => $token,
        'dir' => rtrim($dir, '/\\'),
        'max' => max(1, (int)$env('GHOST_SYNC_MAX_BLOB_MB', GHOST_MAX_BLOB_MB)) * 1048576,
        'ttl' => max(60, min(3600, (int)$env('GHOST_SYNC_LEASE_SECONDS', GHOST_LEASE_SECONDS))),
        'keep' => max(1, min(50, (int)$env('GHOST_SYNC_KEEP_VERSIONS', GHOST_KEEP_VERSIONS))),
        'tomb' => max(1, min(3650, (int)$env('GHOST_SYNC_TOMBSTONE_DAYS', GHOST_TOMBSTONE_DAYS))) * 86400,
        'base' => $dir,
    ];
}

function gs_setup_page(): void
{
    header('Content-Type: text/html; charset=utf-8');
    $t = bin2hex(random_bytes(32));
    echo "<!doctype html><meta charset=utf-8><title>GhostBrowser Sync</title><body style='font:15px system-ui;max-width:680px;margin:40px auto;line-height:1.6'>"
        . "<h2>GhostBrowser 同步服务器</h2><p>还没有设置令牌。请用文本编辑器打开 <code>" . htmlspecialchars(basename(__FILE__)) . "</code>，把</p>"
        . "<pre>const GHOST_TOKEN = '';</pre><p>改成（这是随机生成的，可直接用）：</p><pre>const GHOST_TOKEN = '$t';</pre>"
        . "<p>然后在 GhostBrowser「设置 → 云同步」里填写服务器地址为本文件的网址，令牌填同一个值。</p></body>";
    exit;
}

function gs_init_dir(array $c): void
{
    foreach ([$c['dir'], $c['dir'] . '/blobs', $c['dir'] . '/tmp', $c['dir'] . '/chunks', $c['dir'] . '/refs'] as $d) {
        if (!is_dir($d) && !@mkdir($d, 0700, true) && !is_dir($d)) throw new GsFail(500, 'Cannot create data directory; check folder permissions.');
    }
    $deny = [
        '.htaccess' => "Require all denied\nDeny from all\n",
        'web.config' => '<?xml version="1.0"?><configuration><system.webServer><authorization><deny users="*" /></authorization></system.webServer></configuration>',
        'index.html' => '',
    ];
    foreach ($deny as $f => $body) {
        if (!is_file($c['dir'] . '/' . $f)) @file_put_contents($c['dir'] . '/' . $f, $body);
    }
}

/** Run $fn(&$index) under an exclusive lock and save the index afterwards. */
function gs_tx(array $c, callable $fn)
{
    $lock = fopen($c['dir'] . '/index.lock', 'c');
    if (!$lock || !flock($lock, LOCK_EX)) throw new GsFail(500, 'Cannot lock index.');
    try {
        $file = $c['dir'] . '/index.json';
        $idx = is_file($file) ? json_decode((string)file_get_contents($file), true) : null;
        if (!is_array($idx)) $idx = ['profiles' => []];
        $before = json_encode($idx);
        $result = $fn($idx);
        if (json_encode($idx) !== $before) {
            $tmp = $file . '.' . bin2hex(random_bytes(6)) . '.tmp';
            if (file_put_contents($tmp, json_encode($idx, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES)) === false || !rename($tmp, $file)) {
                @unlink($tmp);
                throw new GsFail(500, 'Cannot write index.');
            }
        }
        return $result;
    } finally {
        flock($lock, LOCK_UN);
        fclose($lock);
    }
}

function gs_body(): array
{
    $b = file_get_contents('php://input', false, null, 0, 131073);
    if ($b === false || strlen($b) > 131072) throw new GsFail(413, 'JSON body exceeds 128 KiB.');
    if ($b === '') return [];
    $v = json_decode($b, true, 64);
    if (!is_array($v)) throw new GsFail(400, 'JSON body must be an object.');
    return $v;
}

function gs_header(string $name): string
{
    $k = 'HTTP_' . strtoupper(str_replace('-', '_', $name));
    return (string)($_SERVER[$k] ?? $_SERVER['REDIRECT_' . $k] ?? '');
}

function gs_lease(array &$idx, string $id): array
{
    $r = $idx['profiles'][$id] ?? null;
    $ticket = gs_header('X-Lock-Token');
    if (!$r || (int)$r['lock_until'] <= time() || $r['lock_device'] !== gs_header('X-Device-ID') ||
        $ticket === '' || !hash_equals((string)$r['lock_hash'], hash('sha256', $ticket))) {
        throw new GsFail(423, 'A valid profile lease is required.');
    }
    return $r;
}

function gs_clear_pending(array $c, array &$r): void
{
    $p = $r['pending_path'] ?? '';
    if ($p && preg_match('/^[a-f0-9]{48}\.part$/D', $p) && is_file($c['dir'] . '/tmp/' . $p)) @unlink($c['dir'] . '/tmp/' . $p);
    $f = $r['pending_refs'] ?? '';
    if ($f && preg_match('/^[a-f0-9]{48}\.refs$/D', $f) && is_file($c['dir'] . '/tmp/' . $f)) @unlink($c['dir'] . '/tmp/' . $f);
    $r['pending_path'] = $r['pending_sha'] = null;
    $r['pending_size'] = $r['pending_base'] = null;
    $r['pending_refs'] = null;
}

/** Folder holding the per-IP failure counters (outside every user's data). */
function gs_guard_dir(): string
{
    $d = __DIR__ . '/.ghostsync-guard';
    if (!is_dir($d)) {
        @mkdir($d, 0700, true);
        @file_put_contents($d . '/.htaccess', "Require all denied\nDeny from all\n");
        @file_put_contents($d . '/index.html', '');
    }
    return $d;
}

function gs_client_ip(): string { return (string)($_SERVER['REMOTE_ADDR'] ?? 'unknown'); }

function gs_guard_file(): string { return gs_guard_dir() . '/' . substr(hash('sha256', gs_client_ip()), 0, 32) . '.json'; }

function gs_guard_check(): void
{
    $f = gs_guard_file();
    $g = is_file($f) ? json_decode((string)@file_get_contents($f), true) : null;
    if (is_array($g) && (int)($g['until'] ?? 0) > time()) {
        header('Retry-After: ' . ((int)$g['until'] - time()));
        throw new GsFail(429, 'Too many wrong tokens from this IP; try again later.');
    }
}

function gs_guard_fail(): void
{
    $f = gs_guard_file();
    $g = is_file($f) ? json_decode((string)@file_get_contents($f), true) : null;
    $now = time();
    if (!is_array($g) || (int)($g['since'] ?? 0) < $now - 900) $g = ['since' => $now, 'fails' => 0, 'until' => 0];
    $g['fails'] = (int)$g['fails'] + 1;
    if ($g['fails'] >= max(1, (int)GHOST_MAX_FAILS)) $g['until'] = $now + 900;
    @file_put_contents($f, json_encode($g), LOCK_EX);
    // Old counters
    if (mt_rand(1, 50) === 1) foreach (glob(gs_guard_dir() . '/*.json') ?: [] as $o) if (filemtime($o) < $now - 86400) @unlink($o);
}

/** Bytes PHP accepts in one request body (smallest of the limits). */
function gs_ini_bytes(string $k): int
{
    $v = trim((string)ini_get($k));
    if ($v === '' || $v === '0' || $v === '-1') return 0;
    $n = (float)$v;
    switch (strtolower(substr($v, -1))) {
        case 'g': $n *= 1024;
        case 'm': $n *= 1024;
        case 'k': $n *= 1024;
    }
    return (int)$n;
}

function gs_dir_size(string $d): int
{
    $n = 0;
    if (!is_dir($d)) return 0;
    $it = new RecursiveIteratorIterator(new RecursiveDirectoryIterator($d, FilesystemIterator::SKIP_DOTS));
    foreach ($it as $f) if ($f->isFile()) $n += $f->getSize();
    return $n;
}

function gs_chunk_path(array $c, string $id): string { return $c['dir'] . '/chunks/' . substr($id, 0, 2) . '/' . $id; }

/** Reads a request body of newline- or JSON-listed chunk IDs. */
function gs_ids(array $b): array
{
    $ids = $b['ids'] ?? null;
    if (!is_array($ids) || count($ids) > 1000) throw new GsFail(400, 'ids must be an array of at most 1000 chunk IDs.');
    foreach ($ids as $id) if (!is_string($id) || !preg_match('/^[a-f0-9]{64}$/D', $id)) throw new GsFail(400, 'Invalid chunk ID.');
    return array_values(array_unique($ids));
}

function gs_hist_entry($h): array { return is_array($h) ? $h : ['sha256' => (string)$h, 'version' => 0, 'size' => 0, 'updated_at' => 0, 'device_name' => '']; }

function gs_view(array $r): array
{
    return ['id' => $r['id'], 'name' => $r['name'], 'version' => (int)$r['version'], 'sha256' => $r['sha256'],
        'meta' => $r['meta'] ?: new stdClass(), 'lockedBy' => (int)$r['lock_until'] > time() ? $r['lock_device'] : '',
        'updatedAt' => (int)$r['updated_at'], 'size' => (int)($r['size'] ?? 0), 'device' => (string)($r['device'] ?? ''), 'deviceName' => (string)($r['device_name'] ?? '')];
}

function gs_is_tomb(?array $r): bool { return is_array($r) && !empty($r['deleted']); }

function gs_tomb_view(array $r): array
{
    return ['id' => $r['id'], 'name' => $r['name'], 'version' => (int)$r['version'], 'deleted' => true,
        'deletedAt' => (int)($r['deleted_at'] ?? 0), 'deletedBy' => (string)($r['deleted_by'] ?? ''),
        'deletedByName' => (string)($r['deleted_by_name'] ?? ''), 'updatedAt' => (int)$r['updated_at'], 'lockedBy' => '', 'sha256' => '', 'size' => 0, 'meta' => new stdClass()];
}

/** 410 for a profile that was deleted from the cloud (tombstone). */
function gs_tomb_fail(array $r): GsFail
{
    $who = (string)($r['deleted_by_name'] ?? '') ?: (string)($r['deleted_by'] ?? '') ?: '另一台设备';
    return new GsFail(410, '该环境已在「' . $who . '」于 ' . gmdate('Y-m-d H:i', (int)($r['deleted_at'] ?? 0)) . ' (UTC) 从云端删除。');
}

/** Drop tombstones older than the configured retention. */
function gs_purge_tombs(array $c, array &$idx): void
{
    foreach ($idx['profiles'] as $k => $r) {
        if (gs_is_tomb($r) && (int)($r['deleted_at'] ?? 0) < time() - $c['tomb'] && (int)$r['lock_until'] <= time()) unset($idx['profiles'][$k]);
    }
}

function gs_blank(string $id): array
{
    return ['id' => $id, 'name' => '', 'version' => 0, 'sha256' => '', 'meta' => [], 'updated_at' => 0, 'size' => 0, 'device' => '',
        'history' => [], 'lock_device' => '', 'lock_hash' => '', 'lock_until' => 0,
        'pending_path' => null, 'pending_sha' => null, 'pending_size' => null, 'pending_base' => null];
}

/** Remove blobs no profile (current or kept history) references any more. */
function gs_gc(array $c, array $idx): void
{
    $live = [];
    foreach ($idx['profiles'] as $r) {
        if ($r['sha256']) $live[$r['sha256']] = true;
        foreach ($r['history'] ?? [] as $h) $live[gs_hist_entry($h)['sha256']] = true;
    }
    foreach (glob($c['dir'] . '/blobs/*.gbe') ?: [] as $f) {
        if (!isset($live[basename($f, '.gbe')])) @unlink($f);
    }
    foreach (glob($c['dir'] . '/refs/*.txt') ?: [] as $f) {
        if (!isset($live[basename($f, '.txt')])) @unlink($f);
    }
    foreach (glob($c['dir'] . '/tmp/*.refs') ?: [] as $f) {
        if (filemtime($f) < time() - 86400) @unlink($f);
    }
    foreach (glob($c['dir'] . '/tmp/*.part') ?: [] as $f) {
        if (filemtime($f) < time() - 86400) @unlink($f);
    }
}

/** Delete chunks no kept version references. Runs at most every 6 hours. */
function gs_chunk_gc(array $c, bool $force = false): void
{
    $stamp = $c['dir'] . '/chunks/.gc';
    if (!$force && is_file($stamp) && filemtime($stamp) > time() - 21600) return;
    @touch($stamp);
    $live = [];
    foreach (glob($c['dir'] . '/refs/*.txt') ?: [] as $f) {
        $h = fopen($f, 'rb');
        if (!$h) return; // never delete on a partial view
        while (($line = fgets($h)) !== false) { $line = trim($line); if ($line !== '') $live[$line] = true; }
        fclose($h);
    }
    $old = time() - 172800; // chunks of an upload still in progress are young
    foreach (glob($c['dir'] . '/chunks/??/*') ?: [] as $f) {
        if (!isset($live[basename($f)]) && filemtime($f) < $old) @unlink($f);
    }
}

function gs_route(): string
{
    if (isset($_GET['r']) && is_string($_GET['r'])) return '/' . ltrim($_GET['r'], '/');
    if (!empty($_SERVER['PATH_INFO'])) return (string)$_SERVER['PATH_INFO'];
    $path = (string)parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH);
    $self = (string)($_SERVER['SCRIPT_NAME'] ?? '');
    if ($self !== '' && strpos($path, $self) === 0) $path = substr($path, strlen($self));
    return $path === '' ? '/' : $path;
}

function gs_serve(): void
{
    $c = gs_cfg();
    $method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
    $path = gs_route();
    if ($path === '/admin') gs_admin_page($c);
    if ($c['token'] === '' && $method === 'GET' && ($path === '/' || $path === '')) gs_setup_page();
    if (strlen($c['token']) < 32) throw new GsFail(503, 'Set GHOST_TOKEN (32+ characters) at the top of ' . basename(__FILE__) . '.');
    if ($method === 'GET' && ($path === '/' || $path === '')) gs_reply(200, ['ok' => true, 'service' => 'ghostbrowser-sync', 'hint' => 'Use this URL as the sync server in GhostBrowser.']);

    // Bearer token; X-Ghost-Token as a fallback for hosts that strip Authorization.
    gs_guard_check();
    $given = '';
    if (preg_match('/^Bearer\s+(.+)$/i', gs_header('Authorization'), $m)) $given = trim($m[1]);
    if ($given === '') $given = gs_header('X-Ghost-Token');
    $user = '';
    if ($given !== '' && hash_equals($c['token'], $given)) {
        $user = 'owner';
    } elseif ($given !== '') {
        foreach (gs_users() as $tok => $name) {
            if (is_string($tok) && strlen($tok) >= 32 && hash_equals($tok, $given)) {
                $user = (string)$name;
                // Each extra user gets a folder derived from their own token.
                $c['dir'] = (getenv('GHOST_SYNC_DATA_DIR') || GHOST_DATA_DIR !== '')
                    ? $c['base'] . '/u-' . substr(hash('sha256', 'ghostsync-user|' . $tok), 0, 20)
                    : __DIR__ . '/.ghostsync-' . substr(hash('sha256', 'ghostsync|' . $tok), 0, 20);
                break;
            }
        }
    }
    if ($user === '') {
        gs_guard_fail();
        header('WWW-Authenticate: Bearer');
        throw new GsFail(401, 'Invalid token.');
    }
    gs_init_dir($c);
    $GLOBALS['gs_sec_ctx'] = ['user' => $user, 'path' => $path, 'base' => $c['base']];
    gs_sec_gate($c, $user, $path);
    gs_sec_routes($c, $user, $path, $method);

    if ($path === '/api/ping' && $method === 'GET') {
        $limits = array_filter([$c['max'], gs_ini_bytes('post_max_size')]);
        $free = @disk_free_space($c['dir']);
        gs_reply(200, ['ok' => true, 'service' => 'ghostbrowser-sync', 'protocol' => 2, 'single' => true, 'time' => time(),
            'user' => $user, 'features' => array_merge(['chunks', 'history', 'multiuser', 'tombstones'], gs_sec_features()), 'keepVersions' => $c['keep'],
            'tombstoneDays' => (int)($c['tomb'] / 86400),
            'maxBody' => $limits ? min($limits) : 0, 'usedBytes' => gs_dir_size($c['dir']),
            'freeBytes' => $free === false ? -1 : (int)$free]);
    }
    if ($path === '/api/chunks/missing' && $method === 'POST') {
        $ids = gs_ids(gs_body());
        $miss = [];
        foreach ($ids as $id) if (!is_file(gs_chunk_path($c, $id))) $miss[] = $id;
        gs_reply(200, ['missing' => $miss]);
    }
    if ($path === '/api/chunks' && $method === 'PUT') {
        if (isset($_SERVER['CONTENT_LENGTH']) && (int)$_SERVER['CONTENT_LENGTH'] > 64 * 1048576) throw new GsFail(413, 'Chunk batch is too large.');
        $in = fopen('php://input', 'rb');
        if (!$in) throw new GsFail(500, 'Cannot open upload stream.');
        $stored = 0;
        $readn = function (int $n) use ($in): string {
            $buf = '';
            while (strlen($buf) < $n && !feof($in)) { $x = fread($in, $n - strlen($buf)); if ($x === false) break; $buf .= $x; }
            return $buf;
        };
        while (true) {
            $id = $readn(64);
            if ($id === '') break;
            if (!preg_match('/^[a-f0-9]{64}$/D', $id)) throw new GsFail(400, 'Bad chunk frame.');
            $len = $readn(4);
            if (strlen($len) !== 4) throw new GsFail(400, 'Truncated chunk frame.');
            $n = unpack('N', $len)[1];
            if ($n < 32 || $n > 8 * 1048576) throw new GsFail(400, 'Bad chunk length.');
            $data = $readn($n);
            if (strlen($data) !== $n || substr($data, 0, 4) !== 'GBC1') throw new GsFail(400, 'Expected an encrypted GBC1 chunk.');
            $dst = gs_chunk_path($c, $id);
            if (!is_file($dst)) {
                if (!is_dir(dirname($dst))) @mkdir(dirname($dst), 0700, true);
                $tmp = $dst . '.' . bin2hex(random_bytes(6)) . '.part';
                if (@file_put_contents($tmp, $data) !== $n) { @unlink($tmp); throw new GsFail(500, 'Chunk write failed (disk full?).'); }
                if (!@rename($tmp, $dst)) @unlink($tmp);
            } else {
                @touch($dst);
            }
            $stored++;
        }
        fclose($in);
        gs_reply(200, ['ok' => true, 'stored' => $stored]);
    }
    if ($path === '/api/chunks/get' && $method === 'POST') {
        $ids = gs_ids(gs_body());
        foreach ($ids as $id) if (!is_file(gs_chunk_path($c, $id))) throw new GsFail(404, 'Chunk ' . $id . ' is missing.');
        while (ob_get_level() > 0) ob_end_clean();
        header('Content-Type: application/octet-stream');
        header('Cache-Control: no-store');
        foreach ($ids as $id) {
            $data = (string)file_get_contents(gs_chunk_path($c, $id));
            echo $id, pack('N', strlen($data)), $data;
            flush();
        }
        exit;
    }
    if ($path === '/api/profiles' && $method === 'GET') {
        $tombs = ($_GET['tombstones'] ?? '') === '1';
        $list = gs_tx($c, function (array &$idx) use ($c, $tombs) {
            gs_purge_tombs($c, $idx);
            $out = [];
            // Old clients never ask for tombstones and so never see deleted profiles.
            foreach ($idx['profiles'] as $r) {
                if (gs_is_tomb($r)) { if ($tombs) $out[] = gs_tomb_view($r); }
                elseif ((int)$r['version'] > 0) $out[] = gs_view($r);
            }
            usort($out, function ($a, $b) { return [$b['updatedAt'], $a['id']] <=> [$a['updatedAt'], $b['id']]; });
            return $out;
        });
        gs_reply(200, $list);
    }
    if (!preg_match('~^/api/profiles/([a-zA-Z0-9_-]{1,64})/(lock|unlock|blob|commit|delete|history|refs)$~D', $path, $m)) throw new GsFail(404, 'Not found.');
    $id = $m[1];
    $action = $m[2];
    $ttl = $c['ttl'];

    if ($action === 'lock' && $method === 'POST') {
        $b = gs_body();
        $device = $b['device'] ?? null;
        if (!is_string($device) || !preg_match('/^[a-zA-Z0-9_-]{1,128}$/D', $device)) throw new GsFail(400, 'Invalid device ID.');
        $restore = ($b['restore'] ?? false) === true;
        $ticket = bin2hex(random_bytes(32));
        $v = gs_tx($c, function (array &$idx) use ($c, $id, $device, $ticket, $ttl, $restore) {
            $r = $idx['profiles'][$id] ?? gs_blank($id);
            // A stale device must not silently bring a deleted profile back.
            if (gs_is_tomb($r) && !$restore) throw gs_tomb_fail($r);
            if ((int)$r['lock_until'] > time() && $r['lock_device'] !== $device) throw new GsFail(423, 'Profile is locked by another sync operation.');
            gs_clear_pending($c, $r);
            $r['lock_restore'] = gs_is_tomb($r) && $restore;
            $r['lock_device'] = $device;
            $r['lock_hash'] = hash('sha256', $ticket);
            $r['lock_until'] = time() + $ttl;
            $idx['profiles'][$id] = $r;
            return (int)$r['version'];
        });
        gs_reply(200, ['ok' => true, 'lockToken' => $ticket, 'version' => $v, 'expiresAt' => time() + $ttl]);
    }
    if ($action === 'unlock' && $method === 'POST') {
        gs_tx($c, function (array &$idx) use ($c, $id) {
            $r = gs_lease($idx, $id);
            gs_clear_pending($c, $r);
            $r['lock_device'] = $r['lock_hash'] = '';
            $r['lock_until'] = 0;
            if ((int)$r['version'] === 0) unset($idx['profiles'][$id]); else $idx['profiles'][$id] = $r;
        });
        gs_reply(200, ['ok' => true]);
    }
    if ($action === 'blob' && $method === 'PUT') {
        $version = $_GET['version'] ?? null;
        if (!is_string($version) || !preg_match('/^(0|[1-9][0-9]{0,15})$/D', $version)) throw new GsFail(400, 'A non-negative version is required.');
        $version = (int)$version;
        gs_tx($c, function (array &$idx) use ($id, $version) {
            $r = gs_lease($idx, $id);
            if ((int)$r['version'] !== $version) throw new GsFail(412, 'A newer version exists; pull first.');
        });
        if (isset($_SERVER['CONTENT_LENGTH']) && (int)$_SERVER['CONTENT_LENGTH'] > $c['max']) throw new GsFail(413, 'Blob is too large.');
        $name = bin2hex(random_bytes(24)) . '.part';
        $file = $c['dir'] . '/tmp/' . $name;
        $in = fopen('php://input', 'rb');
        $out = fopen($file, 'xb');
        if (!$in || !$out) throw new GsFail(500, 'Cannot open upload stream.');
        $ctx = hash_init('sha256');
        $size = 0;
        $head = '';
        try {
            while (!feof($in)) {
                $chunk = fread($in, 1048576);
                if ($chunk === false) throw new GsFail(500, 'Upload read failed.');
                if ($chunk === '') continue;
                $size += strlen($chunk);
                if ($size > $c['max']) throw new GsFail(413, 'Blob is too large.');
                if (strlen($head) < 4) $head .= substr($chunk, 0, 4 - strlen($head));
                hash_update($ctx, $chunk);
                if (fwrite($out, $chunk) !== strlen($chunk)) throw new GsFail(500, 'Upload write failed (disk full?).');
            }
            fclose($out); $out = null; fclose($in); $in = null;
            if ($size < 48 || $head !== 'GBE1') throw new GsFail(400, 'Expected an encrypted GBE1 blob.');
            $sha = hash_final($ctx);
            gs_tx($c, function (array &$idx) use ($c, $id, $version, $name, $sha, $size, $ttl) {
                $r = gs_lease($idx, $id);
                if ((int)$r['version'] !== $version) throw new GsFail(412, 'Version changed during upload.');
                gs_clear_pending($c, $r);
                $r['pending_path'] = $name;
                $r['pending_sha'] = $sha;
                $r['pending_size'] = $size;
                $r['pending_base'] = $version;
                $r['lock_until'] = time() + $ttl;
                $idx['profiles'][$id] = $r;
            });
        } catch (Throwable $e) {
            if (is_resource($out)) fclose($out);
            if (is_resource($in)) fclose($in);
            if (is_file($file)) @unlink($file);
            throw $e;
        }
        gs_reply(200, ['ok' => true, 'sha256' => $sha, 'bytes' => $size]);
    }
    if ($action === 'history' && $method === 'GET') {
        $r = gs_tx($c, function (array &$idx) use ($id) { return $idx['profiles'][$id] ?? null; });
        if (gs_is_tomb($r)) throw gs_tomb_fail($r);
        if (!$r || (int)$r['version'] === 0) throw new GsFail(404, 'Profile has no committed blob.');
        $out = [['version' => (int)$r['version'], 'size' => (int)$r['size'], 'updatedAt' => (int)$r['updated_at'], 'deviceName' => (string)($r['device_name'] ?? ''), 'current' => true]];
        foreach ($r['history'] ?? [] as $h) {
            $h = gs_hist_entry($h);
            if ((int)$h['version'] < 1 || !is_file($c['dir'] . '/blobs/' . $h['sha256'] . '.gbe')) continue;
            $out[] = ['version' => (int)$h['version'], 'size' => (int)$h['size'], 'updatedAt' => (int)$h['updated_at'], 'deviceName' => (string)$h['device_name'], 'current' => false];
        }
        gs_reply(200, $out);
    }
    if ($action === 'refs' && $method === 'PUT') {
        if (isset($_SERVER['CONTENT_LENGTH']) && (int)$_SERVER['CONTENT_LENGTH'] > 32 * 1048576) throw new GsFail(413, 'Chunk list is too large.');
        $raw = (string)file_get_contents('php://input', false, null, 0, 32 * 1048576 + 1);
        if (strlen($raw) > 32 * 1048576 || ($raw !== '' && !preg_match('/^([a-f0-9]{64}\n?)*$/D', $raw))) throw new GsFail(400, 'Invalid chunk list.');
        $name = bin2hex(random_bytes(24)) . '.refs';
        if (@file_put_contents($c['dir'] . '/tmp/' . $name, $raw) !== strlen($raw)) throw new GsFail(500, 'Cannot store chunk list.');
        gs_tx($c, function (array &$idx) use ($c, $id, $name) {
            $r = gs_lease($idx, $id);
            $old = $r['pending_refs'] ?? '';
            if ($old && preg_match('/^[a-f0-9]{48}\.refs$/D', $old)) @unlink($c['dir'] . '/tmp/' . $old);
            $r['pending_refs'] = $name;
            $idx['profiles'][$id] = $r;
        });
        gs_reply(200, ['ok' => true]);
    }
    if ($action === 'commit' && $method === 'POST') {
        $b = gs_body();
        if (($b['id'] ?? '') !== $id || !is_int($b['version'] ?? null) || !is_string($b['sha256'] ?? null) ||
            !preg_match('/^[a-f0-9]{64}$/D', $b['sha256']) || !is_string($b['name'] ?? null) || strlen($b['name']) > 1024 ||
            (($b['meta']['encrypted'] ?? null) !== true)) throw new GsFail(400, 'Invalid commit metadata.');
        $idx = gs_tx($c, function (array &$idx) use ($c, $id, $b) {
            $r = gs_lease($idx, $id);
            if (gs_is_tomb($r) && empty($r['lock_restore'])) throw gs_tomb_fail($r);
            if ((int)$r['version'] + 1 !== $b['version'] || (int)$r['pending_base'] !== (int)$r['version']) throw new GsFail(412, 'Version conflict.');
            if (!$r['pending_path'] || !hash_equals((string)$r['pending_sha'], $b['sha256'])) throw new GsFail(409, 'Upload SHA-256 does not match the commit.');
            $src = $c['dir'] . '/tmp/' . $r['pending_path'];
            if (!is_file($src) || filesize($src) !== (int)$r['pending_size'] || !hash_equals($b['sha256'], hash_file('sha256', $src))) {
                throw new GsFail(409, 'Pending blob failed integrity validation.');
            }
            $dst = $c['dir'] . '/blobs/' . $b['sha256'] . '.gbe';
            if (!is_file($dst)) {
                if (!rename($src, $dst)) throw new GsFail(500, 'Cannot store upload.');
            } else {
                @unlink($src);
            }
            $refs = $r['pending_refs'] ?? '';
            if ($refs && preg_match('/^[a-f0-9]{48}\.refs$/D', $refs) && is_file($c['dir'] . '/tmp/' . $refs)) {
                @rename($c['dir'] . '/tmp/' . $refs, $c['dir'] . '/refs/' . $b['sha256'] . '.txt');
            }
            $r['pending_refs'] = null;
            $hist = array_map('gs_hist_entry', $r['history'] ?? []);
            if ($r['sha256']) array_unshift($hist, ['sha256' => $r['sha256'], 'version' => (int)$r['version'], 'size' => (int)$r['size'],
                'updated_at' => (int)$r['updated_at'], 'device_name' => (string)($r['device_name'] ?? '')]);
            $seen = [];
            $hist = array_values(array_filter($hist, function ($h) use (&$seen, $b) {
                if ($h['sha256'] === $b['sha256'] || isset($seen[$h['sha256']])) return false;
                return $seen[$h['sha256']] = true;
            }));
            $r['history'] = array_slice($hist, 0, max(0, $c['keep'] - 1));
            $r['name'] = $b['name'];
            $r['version'] = $b['version'];
            $r['sha256'] = $b['sha256'];
            $r['size'] = (int)$r['pending_size'];
            $r['device'] = gs_header('X-Device-ID');
            $r['device_name'] = substr(rawurldecode(gs_header('X-Device-Name')), 0, 128);
            $r['meta'] = ['encrypted' => true, 'format' => 'GBE1', 'profileMetadata' => 'in-blob'];
            $r['updated_at'] = time();
            $r['lock_device'] = $r['lock_hash'] = '';
            $r['lock_until'] = 0;
            $r['pending_path'] = $r['pending_sha'] = null;
            $r['pending_size'] = $r['pending_base'] = null;
            unset($r['deleted'], $r['deleted_at'], $r['deleted_by'], $r['deleted_by_name'], $r['lock_restore']);
            $idx['profiles'][$id] = $r;
            return $idx;
        });
        gs_gc($c, $idx);
        gs_chunk_gc($c);
        gs_reply(200, ['ok' => true, 'version' => $b['version']]);
    }
    if ($action === 'blob' && $method === 'GET') {
        $r = gs_tx($c, function (array &$idx) use ($id) { return $idx['profiles'][$id] ?? null; });
        if (gs_is_tomb($r)) throw gs_tomb_fail($r);
        if (!$r || (int)$r['version'] === 0) throw new GsFail(404, 'Profile has no committed blob.');
        $want = $_GET['version'] ?? '';
        if (is_string($want) && $want !== '' && (int)$want !== (int)$r['version']) {
            $found = null;
            foreach ($r['history'] ?? [] as $h) { $h = gs_hist_entry($h); if ((int)$h['version'] === (int)$want) $found = $h; }
            if (!$found) throw new GsFail(404, 'That version is no longer kept on the server.');
            $r['sha256'] = $found['sha256'];
            $r['version'] = (int)$found['version'];
        }
        $p = $c['dir'] . '/blobs/' . $r['sha256'] . '.gbe';
        if (!is_file($p)) throw new GsFail(500, 'Committed blob is missing.');
        while (ob_get_level() > 0) ob_end_clean();
        header('Content-Type: application/octet-stream');
        header('Content-Length: ' . filesize($p));
        header('X-Profile-Version: ' . $r['version']);
        header('X-Blob-SHA256: ' . $r['sha256']);
        header('ETag: "' . $r['sha256'] . '"');
        header('Cache-Control: no-store');
        header('X-Content-Type-Options: nosniff');
        readfile($p);
        exit;
    }
    if ($action === 'delete' && $method === 'POST') {
        $b = gs_body();
        $force = ($b['force'] ?? false) === true;
        $me = gs_header('X-Device-ID');
        $res = gs_tx($c, function (array &$idx) use ($c, $id, $force, $me) {
            gs_purge_tombs($c, $idx);
            $r = $idx['profiles'][$id] ?? null;
            if (gs_is_tomb($r)) return ['idx' => $idx, 'state' => 'already', 'version' => (int)$r['version']];
            if (!$r || (int)$r['version'] === 0) {
                if ($r && (int)$r['lock_until'] > time() && $r['lock_device'] !== $me && !$force) throw new GsFail(423, '该环境正在被另一台设备上传，暂时不能删除；稍后重试或选择强制删除。');
                if ($r) { gs_clear_pending($c, $r); unset($idx['profiles'][$id]); }
                return ['idx' => $idx, 'state' => 'absent', 'version' => 0];
            }
            if ((int)$r['lock_until'] > time() && $r['lock_device'] !== $me && !$force) {
                throw new GsFail(423, '该环境正在被另一台设备（' . $r['lock_device'] . '）同步，暂时不能删除；稍后重试或选择强制删除。');
            }
            gs_clear_pending($c, $r);
            // Tombstone: the entry stays (without data) so other devices learn
            // about the deletion and stale uploads are refused.
            $r['version'] = (int)$r['version'] + 1;
            $r['sha256'] = '';
            $r['size'] = 0;
            $r['history'] = [];
            $r['meta'] = [];
            $r['deleted'] = true;
            $r['deleted_at'] = time();
            $r['deleted_by'] = $me;
            $r['deleted_by_name'] = substr(rawurldecode(gs_header('X-Device-Name')), 0, 128);
            $r['updated_at'] = time();
            $r['lock_device'] = $r['lock_hash'] = '';
            $r['lock_until'] = 0;
            $r['lock_restore'] = false;
            $idx['profiles'][$id] = $r;
            return ['idx' => $idx, 'state' => 'deleted', 'version' => $r['version']];
        });
        gs_gc($c, $res['idx']);
        gs_chunk_gc($c, true);
        gs_reply(200, ['ok' => true, 'state' => $res['state'], 'version' => $res['version']]);
    }
    header('Allow: ' . ($action === 'blob' ? 'GET, PUT' : ($action === 'history' ? 'GET' : ($action === 'refs' ? 'PUT' : 'POST'))));
    throw new GsFail(405, 'Method not allowed.');
}

// ======================================================================
// Accounts & security: devices, key envelope, public-key directory,
// profile sharing between users, metadata-only admin page.
// Everything stored here is metadata or client-encrypted; the server never
// sees a password, a key or decrypted profile data.
// ======================================================================

/** UTF-8 safe substring without mbstring. */
function gs_cut(string $s, int $from, int $n): string
{
    if (!preg_match('/^.{0,' . (int)$n . '}/us', $s, $m)) return substr($s, 0, $n);
    return $m[0];
}

function gs_sec_features(): array { return ['devices', 'keys', 'directory', 'shares']; }

/** Extra users: GHOST_USERS plus env GHOST_SYNC_USERS="token=name;token=name". */
function gs_users(): array
{
    $out = GHOST_USERS;
    $env = getenv('GHOST_SYNC_USERS');
    if (is_string($env) && $env !== '') {
        foreach (explode(';', $env) as $pair) {
            $kv = explode('=', trim($pair), 2);
            if (count($kv) === 2 && strlen($kv[0]) >= 32 && $kv[1] !== '') $out[$kv[0]] = $kv[1];
        }
    }
    return $out;
}

/** Data folder of an extra user (same rule as gs_serve). */
function gs_user_dir(array $c, string $tok): string
{
    return (getenv('GHOST_SYNC_DATA_DIR') || GHOST_DATA_DIR !== '')
        ? $c['base'] . '/u-' . substr(hash('sha256', 'ghostsync-user|' . $tok), 0, 20)
        : __DIR__ . '/.ghostsync-' . substr(hash('sha256', 'ghostsync|' . $tok), 0, 20);
}

/** Folder shared by all users (directory, shares, error log). */
function gs_shared_dir(string $base): string
{
    $d = $base . '/_shared';
    if (!is_dir($d . '/shares') && !@mkdir($d . '/shares', 0700, true) && !is_dir($d . '/shares')) throw new GsFail(500, 'Cannot create shared directory.');
    if (!is_file($d . '/.htaccess')) @file_put_contents($d . '/.htaccess', "Require all denied\nDeny from all\n");
    return $d;
}

/** Run $fn(&$data) on a JSON file under an exclusive lock; saves when changed. */
function gs_jtx(string $file, callable $fn)
{
    $lock = fopen($file . '.lock', 'c');
    if (!$lock || !flock($lock, LOCK_EX)) throw new GsFail(500, 'Cannot lock ' . basename($file) . '.');
    try {
        $data = is_file($file) ? json_decode((string)file_get_contents($file), true) : null;
        if (!is_array($data)) $data = [];
        $before = json_encode($data);
        $result = $fn($data);
        if (json_encode($data) !== $before) {
            $tmp = $file . '.' . bin2hex(random_bytes(6)) . '.tmp';
            if (file_put_contents($tmp, json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES)) === false || !rename($tmp, $file)) {
                @unlink($tmp);
                throw new GsFail(500, 'Cannot write ' . basename($file) . '.');
            }
        }
        return $result;
    } finally {
        flock($lock, LOCK_UN);
        fclose($lock);
    }
}

function gs_jread(string $file): array
{
    $d = is_file($file) ? json_decode((string)@file_get_contents($file), true) : null;
    return is_array($d) ? $d : [];
}

function gs_sec_log(int $status, string $msg): void
{
    $ctx = $GLOBALS['gs_sec_ctx'] ?? null;
    if (!is_array($ctx) || $status < 400 || $status === 404) return;
    try {
        $f = gs_shared_dir($ctx['base']) . '/errors.jsonl';
        @file_put_contents($f, json_encode(['t' => time(), 'user' => $ctx['user'], 'status' => $status,
            'route' => substr(preg_replace('~/api/profiles/[^/]+~', '/api/profiles/*', (string)$ctx['path']), 0, 80),
            'msg' => gs_cut($msg, 0, 160)], JSON_UNESCAPED_UNICODE) . "\n", FILE_APPEND | LOCK_EX);
        if (mt_rand(1, 20) === 1 && @filesize($f) > 262144) {
            $lines = file($f, FILE_IGNORE_NEW_LINES) ?: [];
            @file_put_contents($f, implode("\n", array_slice($lines, -300)) . "\n", LOCK_EX);
        }
    } catch (Throwable $e) {
    }
}

// ------------------------------------------------------------ devices

function gs_dev_view(array $d): array
{
    return ['id' => $d['id'], 'name' => (string)$d['name'], 'os' => (string)$d['os'], 'firstSeen' => (int)$d['first_seen'],
        'lastSeen' => (int)$d['last_seen'], 'approved' => !empty($d['approved']), 'revoked' => !empty($d['revoked']),
        'status' => !empty($d['revoked']) ? 'revoked' : (!empty($d['approved']) ? 'approved' : 'pending')];
}

/** Registers the calling device and refuses revoked / unapproved ones. */
function gs_sec_gate(array $c, string $user, string $path): void
{
    $id = gs_header('X-Device-ID');
    $open = in_array($path, ['/api/ping', '/api/devices/self'], true);
    $file = $c['dir'] . '/devices.json';
    if ($id === '') {
        $reg = gs_jread($file);
        if (!$open && !empty($reg['settings']['requireApproval'] ?? GHOST_REQUIRE_APPROVAL)) throw new GsFail(403, '该同步账号要求设备批准，请升级 GhostBrowser 客户端后再连接。');
        return;
    }
    if (!preg_match('/^[a-zA-Z0-9_-]{1,128}$/D', $id)) throw new GsFail(400, 'Invalid device ID.');
    $name = gs_cut(rawurldecode(gs_header('X-Device-Name')), 0, 64);
    $os = substr(preg_replace('/[^a-zA-Z0-9 ._-]/', '', gs_header('X-Device-OS')), 0, 32);
    $st = gs_jtx($file, function (array &$reg) use ($id, $name, $os) {
        if (!isset($reg['settings'])) $reg['settings'] = ['requireApproval' => (bool)GHOST_REQUIRE_APPROVAL];
        if (!isset($reg['devices'])) $reg['devices'] = [];
        $now = time();
        $d = $reg['devices'][$id] ?? null;
        if (!$d) {
            $anyApproved = false;
            foreach ($reg['devices'] as $o) if (!empty($o['approved']) && empty($o['revoked'])) $anyApproved = true;
            // The very first device of an account is always trusted (bootstrap).
            $d = ['id' => $id, 'name' => $name, 'os' => $os, 'first_seen' => $now, 'last_seen' => $now,
                'approved' => !$anyApproved || empty($reg['settings']['requireApproval']), 'revoked' => false];
            if (count($reg['devices']) >= 500) throw new GsFail(429, '设备数量过多，请先在设备管理中清理。');
        } else {
            if ($now - (int)$d['last_seen'] >= 60) $d['last_seen'] = $now;
            if ($name !== '' && $name !== $d['name']) $d['name'] = $name;
            if ($os !== '' && $os !== $d['os']) $d['os'] = $os;
        }
        $reg['devices'][$id] = $d;
        return gs_dev_view($d)['status'];
    });
    if ($st === 'revoked') throw new GsFail(403, '此设备已被吊销，不能再访问该同步账号。如需恢复，请在已批准的设备上「设置 → 云同步 → 设备管理」重新批准。');
    if ($st === 'pending' && !$open) throw new GsFail(403, '此设备正在等待批准：请在已批准的设备上打开「设置 → 云同步 → 设备管理」批准它。');
}

function gs_dev_routes(array $c, string $path, string $method): void
{
    $file = $c['dir'] . '/devices.json';
    $me = gs_header('X-Device-ID');
    if ($path === '/api/devices/self' && $method === 'GET') {
        $reg = gs_jread($file);
        $d = $reg['devices'][$me] ?? null;
        gs_reply(200, ['id' => $me, 'status' => $d ? gs_dev_view($d)['status'] : 'unknown',
            'requireApproval' => !empty($reg['settings']['requireApproval'] ?? GHOST_REQUIRE_APPROVAL)]);
    }
    if ($path === '/api/devices' && $method === 'GET') {
        $reg = gs_jread($file);
        $list = array_map('gs_dev_view', array_values($reg['devices'] ?? []));
        usort($list, function ($a, $b) { return $b['lastSeen'] <=> $a['lastSeen']; });
        gs_reply(200, ['self' => $me, 'requireApproval' => !empty($reg['settings']['requireApproval'] ?? GHOST_REQUIRE_APPROVAL), 'devices' => $list]);
    }
    if ($path === '/api/devices/settings' && $method === 'POST') {
        $b = gs_body();
        if (!is_bool($b['requireApproval'] ?? null)) throw new GsFail(400, 'requireApproval must be true or false.');
        $v = $b['requireApproval'];
        gs_jtx($file, function (array &$reg) use ($v) { $reg['settings']['requireApproval'] = $v; });
        gs_reply(200, ['ok' => true, 'requireApproval' => $v]);
    }
    if (preg_match('~^/api/devices/([a-zA-Z0-9_-]{1,128})/(approve|revoke|forget)$~D', $path, $m) && $method === 'POST') {
        [$all, $id, $act] = $m;
        if ($id === $me && $act !== 'approve') throw new GsFail(400, '不能吊销或移除当前正在使用的设备。');
        $view = gs_jtx($file, function (array &$reg) use ($id, $act) {
            $d = $reg['devices'][$id] ?? null;
            if (!$d) throw new GsFail(404, '找不到该设备。');
            if ($act === 'forget') { unset($reg['devices'][$id]); return null; }
            $d['approved'] = $act === 'approve';
            $d['revoked'] = $act === 'revoke';
            $reg['devices'][$id] = $d;
            return gs_dev_view($d);
        });
        gs_reply(200, ['ok' => true, 'device' => $view]);
    }
}

// ------------------------------------------------------------ key envelope

function gs_key_routes(array $c, string $path, string $method): void
{
    if ($path !== '/api/keys') return;
    $file = $c['dir'] . '/keys.json';
    if ($method === 'GET') {
        $k = gs_jread($file);
        gs_reply(200, ['epoch' => (int)($k['epoch'] ?? 0), 'envelope' => $k['envelope'] ?? null, 'updatedAt' => (int)($k['updated_at'] ?? 0)]);
    }
    if ($method === 'PUT') {
        $b = gs_body();
        $env = $b['envelope'] ?? null;
        if (!is_array($env) || !is_int($env['v'] ?? null) || strlen(json_encode($env)) > 65536) throw new GsFail(400, 'Invalid key envelope.');
        if (!is_int($b['epoch'] ?? null)) throw new GsFail(400, 'epoch (the epoch you read) is required.');
        $dev = gs_header('X-Device-ID');
        $epoch = gs_jtx($file, function (array &$k) use ($b, $env, $dev) {
            if ((int)($k['epoch'] ?? 0) !== $b['epoch']) throw new GsFail(409, '密钥信息已被另一台设备修改，请刷新后重试。');
            $k = ['epoch' => (int)($k['epoch'] ?? 0) + 1, 'envelope' => $env, 'updated_at' => time(), 'device' => $dev];
            return $k['epoch'];
        });
        gs_reply(200, ['ok' => true, 'epoch' => $epoch]);
    }
    throw new GsFail(405, 'Method not allowed.');
}

// ------------------------------------------------------------ directory + shares

function gs_dir_routes(array $c, string $user, string $path, string $method): void
{
    $file = gs_shared_dir($c['base']) . '/directory.json';
    if ($path === '/api/directory' && $method === 'GET') {
        $d = gs_jread($file);
        $out = [];
        foreach ($d as $h => $e) if ($h !== $user) $out[] = ['user' => (string)$h, 'updatedAt' => (int)$e['updated_at']];
        gs_reply(200, ['self' => $user, 'users' => $out]);
    }
    if ($path === '/api/directory' && $method === 'POST') {
        $b = gs_body();
        $pub = $b['pub'] ?? null;
        if (!is_string($pub) || strlen((string)base64_decode($pub, true)) !== 32) throw new GsFail(400, 'pub must be a base64 X25519 public key.');
        gs_jtx($file, function (array &$d) use ($user, $pub) {
            if (($d[$user]['pub'] ?? '') !== $pub) $d[$user] = ['pub' => $pub, 'updated_at' => time()];
        });
        gs_reply(200, ['ok' => true, 'user' => $user]);
    }
    if (preg_match('~^/api/directory/(.{1,64})$~Du', $path, $m) && $method === 'GET') {
        $h = rawurldecode($m[1]);
        $d = gs_jread($file);
        if (!isset($d[$h])) throw new GsFail(404, '找不到用户「' . $h . '」，或对方尚未启用分享（需要在客户端设置恢复密钥）。');
        gs_reply(200, ['user' => $h, 'pub' => $d[$h]['pub'], 'updatedAt' => (int)$d[$h]['updated_at']]);
    }
}

/** Status a recipient / sender sees, with lazy expiry. */
function gs_share_status(array $s): string
{
    if (in_array($s['status'], ['pending', 'accepted'], true) && (int)$s['expires_at'] <= time()) return 'expired';
    return (string)$s['status'];
}

function gs_share_view(array $s, bool $incoming): array
{
    $v = ['id' => $s['id'], 'from' => $s['from'], 'to' => $s['to'], 'name' => $s['name'], 'mode' => $s['mode'], 'note' => $s['note'],
        'os' => (string)($s['os'] ?? ''), 'createdAt' => (int)$s['created_at'], 'expiresAt' => (int)$s['expires_at'],
        'updatedAt' => (int)$s['updated_at'], 'size' => (int)($s['size'] ?? 0), 'status' => gs_share_status($s)];
    if ($incoming) $v['wrap'] = $s['wrap'];
    return $v;
}

/** Drops blobs of ended shares, and the records themselves 30 days later. */
function gs_share_purge(string $dir, array &$d): void
{
    $now = time();
    foreach ($d as $sid => $s) {
        $st = gs_share_status($s);
        if ($st === 'uploading' && (int)$s['created_at'] < $now - 86400) $st = 'failed';
        if (!in_array($st, ['pending', 'accepted', 'uploading'], true)) {
            if (is_file($dir . '/shares/' . $sid . '.gbe')) @unlink($dir . '/shares/' . $sid . '.gbe');
            if ($st !== $s['status']) { $d[$sid]['status'] = $st; $d[$sid]['updated_at'] = $now; }
            if ((int)$d[$sid]['updated_at'] < $now - 30 * 86400) unset($d[$sid]);
        }
    }
}

function gs_share_routes(array $c, string $user, string $path, string $method): void
{
    $dir = gs_shared_dir($c['base']);
    $file = $dir . '/shares.json';
    if ($path === '/api/shares' && $method === 'GET') {
        $out = gs_jtx($file, function (array &$d) use ($dir, $user) {
            gs_share_purge($dir, $d);
            $o = ['self' => $user, 'out' => [], 'in' => []];
            foreach ($d as $s) {
                if ($s['from'] === $user && $s['status'] !== 'failed') $o['out'][] = gs_share_view($s, false);
                if ($s['to'] === $user && !in_array($s['status'], ['uploading', 'failed'], true)) $o['in'][] = gs_share_view($s, true);
            }
            foreach (['out', 'in'] as $k) usort($o[$k], function ($a, $b) { return $b['createdAt'] <=> $a['createdAt']; });
            return $o;
        });
        gs_reply(200, $out);
    }
    if ($path === '/api/shares' && $method === 'POST') {
        $b = gs_body();
        $to = $b['to'] ?? null;
        $days = $b['days'] ?? null;
        $wrap = $b['wrap'] ?? null;
        if (!is_string($to) || $to === '' || $to === $user) throw new GsFail(400, '请选择另一位用户作为接收者。');
        $dirx = gs_jread($dir . '/directory.json');
        if (!isset($dirx[$to])) throw new GsFail(404, '找不到用户「' . $to . '」，或对方尚未启用分享。');
        if (!is_int($days) || $days < 1 || $days > 366) throw new GsFail(400, '有效期必须在 1–366 天之间。');
        if (!in_array($b['mode'] ?? '', ['session', 'config'], true)) throw new GsFail(400, 'mode must be session or config.');
        if (!is_array($wrap) || !is_string($wrap['eph'] ?? null) || !is_string($wrap['nonce'] ?? null) || !is_string($wrap['ct'] ?? null) || strlen(json_encode($wrap)) > 2048) throw new GsFail(400, 'Invalid wrapped share key.');
        $sid = bin2hex(random_bytes(12));
        $now = time();
        $s = ['id' => $sid, 'from' => $user, 'to' => $to, 'name' => gs_cut((string)($b['name'] ?? ''), 0, 120), 'mode' => $b['mode'],
            'note' => gs_cut((string)($b['note'] ?? ''), 0, 500), 'os' => substr((string)($b['os'] ?? ''), 0, 32),
            'wrap' => ['eph' => $wrap['eph'], 'nonce' => $wrap['nonce'], 'ct' => $wrap['ct'], 'pub' => $dirx[$to]['pub']],
            'created_at' => $now, 'updated_at' => $now, 'expires_at' => $now + $days * 86400, 'status' => 'uploading', 'size' => 0, 'sha256' => ''];
        gs_jtx($file, function (array &$d) use ($s, $user) {
            $n = 0;
            foreach ($d as $o) if ($o['from'] === $user && in_array($o['status'], ['pending', 'accepted', 'uploading'], true)) $n++;
            if ($n >= 500) throw new GsFail(429, '进行中的分享过多。');
            $d[$s['id']] = $s;
        });
        gs_reply(200, ['ok' => true, 'id' => $sid, 'expiresAt' => $s['expires_at']]);
    }
    if (!preg_match('~^/api/shares/([a-f0-9]{24})/(blob|accept|decline|leave|revoke|extend)$~D', $path, $m)) return;
    [$all, $sid, $act] = $m;
    $blob = $dir . '/shares/' . $sid . '.gbe';
    if ($act === 'blob' && $method === 'PUT') {
        $max = max(1, (int)(getenv('GHOST_SYNC_SHARE_MAX_MB') ?: GHOST_SHARE_MAX_MB)) * 1048576;
        $s = gs_jread($file)[$sid] ?? null;
        if (!$s || $s['from'] !== $user) throw new GsFail(404, '找不到该分享。');
        if ($s['status'] !== 'uploading') throw new GsFail(409, '该分享已上传。');
        if (isset($_SERVER['CONTENT_LENGTH']) && (int)$_SERVER['CONTENT_LENGTH'] > $max) throw new GsFail(413, 'Share is too large.');
        $tmp = $blob . '.' . bin2hex(random_bytes(6)) . '.part';
        $in = fopen('php://input', 'rb');
        $out = fopen($tmp, 'xb');
        $size = 0; $head = ''; $ctx = hash_init('sha256');
        while (!feof($in)) {
            $x = fread($in, 1048576);
            if ($x === false || $x === '') { if ($x === false) break; continue; }
            $size += strlen($x);
            if ($size > $max) { fclose($out); @unlink($tmp); throw new GsFail(413, 'Share is too large.'); }
            if (strlen($head) < 4) $head .= substr($x, 0, 4 - strlen($head));
            hash_update($ctx, $x);
            fwrite($out, $x);
        }
        fclose($out); fclose($in);
        if ($size < 48 || $head !== 'GBE1') { @unlink($tmp); throw new GsFail(400, 'Expected an encrypted GBE1 blob.'); }
        if (!@rename($tmp, $blob)) { @unlink($tmp); throw new GsFail(500, 'Cannot store share.'); }
        $sha = hash_final($ctx);
        gs_jtx($file, function (array &$d) use ($sid, $size, $sha) {
            $d[$sid]['status'] = 'pending'; $d[$sid]['size'] = $size; $d[$sid]['sha256'] = $sha; $d[$sid]['updated_at'] = time();
        });
        gs_reply(200, ['ok' => true, 'bytes' => $size, 'sha256' => $sha]);
    }
    if ($act === 'blob' && $method === 'GET') {
        $s = gs_jread($file)[$sid] ?? null;
        if (!$s || $s['to'] !== $user) throw new GsFail(404, '找不到该分享。');
        $st = gs_share_status($s);
        if (!in_array($st, ['pending', 'accepted'], true) || !is_file($blob)) throw new GsFail(410, $st === 'expired' ? '该分享已过期。' : '该分享已被撤销或已结束。');
        while (ob_get_level() > 0) ob_end_clean();
        header('Content-Type: application/octet-stream');
        header('Content-Length: ' . filesize($blob));
        header('X-Blob-SHA256: ' . $s['sha256']);
        header('Cache-Control: no-store');
        readfile($blob);
        exit;
    }
    if ($method !== 'POST') throw new GsFail(405, 'Method not allowed.');
    $b = $act === 'extend' ? gs_body() : [];
    $view = gs_jtx($file, function (array &$d) use ($dir, $sid, $act, $user, $b) {
        gs_share_purge($dir, $d);
        $s = $d[$sid] ?? null;
        $mine = $s && (in_array($act, ['accept', 'decline', 'leave'], true) ? $s['to'] === $user : $s['from'] === $user);
        if (!$mine) throw new GsFail(404, '找不到该分享。');
        $st = gs_share_status($s);
        $live = in_array($st, ['pending', 'accepted'], true);
        if ($act === 'accept') { if ($st !== 'pending' && $st !== 'accepted') throw new GsFail(410, $st === 'expired' ? '该分享已过期。' : '该分享已被撤销或已结束。'); $s['status'] = 'accepted'; }
        elseif ($act === 'decline') { if (!$live) throw new GsFail(410, '该分享已结束。'); $s['status'] = 'declined'; }
        elseif ($act === 'leave') { $s['status'] = 'left'; }
        elseif ($act === 'revoke') { $s['status'] = 'revoked'; }
        elseif ($act === 'extend') {
            $days = $b['days'] ?? null;
            if (!is_int($days) || $days < 1 || $days > 366) throw new GsFail(400, '续期天数必须在 1–366 天之间。');
            if (!$live) throw new GsFail(410, '该分享已结束，不能续期；请重新分享。');
            $s['expires_at'] = time() + $days * 86400;
        }
        $s['updated_at'] = time();
        $d[$sid] = $s;
        gs_share_purge($dir, $d);
        return gs_share_view($d[$sid] ?? $s, false);
    });
    gs_reply(200, ['ok' => true, 'share' => $view]);
}

function gs_sec_routes(array $c, string $user, string $path, string $method): void
{
    if (strpos($path, '/api/devices') === 0) { gs_dev_routes($c, $path, $method); throw new GsFail(404, 'Not found.'); }
    if ($path === '/api/keys') gs_key_routes($c, $path, $method);
    if (strpos($path, '/api/directory') === 0) { gs_dir_routes($c, $user, $path, $method); throw new GsFail(404, 'Not found.'); }
    if (strpos($path, '/api/shares') === 0) { gs_share_routes($c, $user, $path, $method); throw new GsFail(404, 'Not found.'); }
}

// ------------------------------------------------------------ admin page

function gs_admin_page(array $c): void
{
    $env = getenv('GHOST_SYNC_ADMIN_TOKEN');
    $tok = ($env !== false && $env !== '') ? $env : GHOST_ADMIN_TOKEN;
    header('Content-Type: text/html; charset=utf-8');
    header('Cache-Control: no-store');
    header('X-Frame-Options: DENY');
    header("Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; form-action 'self'");
    header('Referrer-Policy: no-referrer');
    $css = "body{margin:0;background:#0b0c10;color:#e7e8ee;font:14px/1.55 system-ui,'PingFang SC','Microsoft YaHei',sans-serif}"
        . ".wrap{max-width:1080px;margin:0 auto;padding:36px 28px}h1{font-size:20px;margin:0 0 4px;font-weight:650;letter-spacing:.2px}"
        . ".sub{color:#8b8fa3;font-size:12.5px;margin-bottom:24px}.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-bottom:22px}"
        . ".card{background:linear-gradient(180deg,#14161d,#111218);border:1px solid #23252f;border-radius:14px;padding:16px 18px}"
        . ".k{color:#8b8fa3;font-size:12px}.v{font-size:24px;font-weight:650;margin-top:4px;background:linear-gradient(90deg,#8ab4ff,#c59bff,#ff9bd2);-webkit-background-clip:text;background-clip:text;color:transparent}"
        . "h2{font-size:14px;margin:26px 0 10px;color:#c9cbd6;font-weight:600}table{width:100%;border-collapse:collapse;background:#111218;border:1px solid #23252f;border-radius:14px;overflow:hidden}"
        . "th,td{text-align:left;padding:10px 14px;border-bottom:1px solid #1d1f28;font-size:13px;white-space:nowrap}th{color:#8b8fa3;font-weight:500;font-size:12px;background:#14161d}"
        . "tr:last-child td{border-bottom:0}td.msg{white-space:normal;color:#c9cbd6;max-width:420px}.b{display:inline-block;padding:1px 8px;border-radius:99px;font-size:11.5px;border:1px solid}"
        . ".ok{color:#5fd4a0;border-color:#5fd4a033;background:#5fd4a012}.warn{color:#f3c262;border-color:#f3c26233;background:#f3c26212}.bad{color:#ff7a8a;border-color:#ff7a8a33;background:#ff7a8a12}"
        . ".mut{color:#6c7084}.login{max-width:360px;margin:16vh auto;padding:28px}.login input{width:100%;box-sizing:border-box;background:#0f1015;border:1px solid #2a2d39;color:#e7e8ee;border-radius:10px;padding:10px 12px;font:inherit;margin:14px 0}"
        . "button{background:linear-gradient(90deg,#6c8cff,#b07cff);border:0;color:#fff;border-radius:10px;padding:9px 16px;font:inherit;cursor:pointer;width:100%}"
        . ".note{color:#8b8fa3;font-size:12px;margin-top:18px}.empty{padding:22px;text-align:center;color:#6c7084}";
    $page = function (string $body) use ($css) {
        echo "<!doctype html><html lang=zh-CN><meta charset=utf-8><meta name=viewport content='width=device-width,initial-scale=1'><title>GhostSync 管理</title><style>$css</style><body>$body</body></html>";
        exit;
    };
    if (strlen((string)$tok) < 32) $page("<div class='card login'><h1>管理页未启用</h1><p class=sub>在 ghostsync.php 顶部把 <code>GHOST_ADMIN_TOKEN</code> 设为 32 位以上随机字符串（或环境变量 GHOST_SYNC_ADMIN_TOKEN）后刷新本页。</p></div>");
    $cookie = 'gsadm_' . substr(hash('sha256', __FILE__), 0, 8);
    $sess = hash_hmac('sha256', 'ghostsync-admin', (string)$tok);
    $given = gs_header('X-Admin-Token');
    $ok = ($given !== '' && hash_equals((string)$tok, $given)) || (isset($_COOKIE[$cookie]) && hash_equals($sess, (string)$_COOKIE[$cookie]));
    $err = '';
    if (!$ok && ($_SERVER['REQUEST_METHOD'] ?? '') === 'POST') {
        gs_guard_check();
        if (hash_equals((string)$tok, (string)($_POST['token'] ?? ''))) {
            setcookie($cookie, $sess, ['expires' => time() + 43200, 'httponly' => true, 'samesite' => 'Strict', 'secure' => !empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off']);
            header('Location: ?r=/admin', true, 303);
            exit;
        }
        gs_guard_fail();
        $err = "<p class='b bad'>令牌不正确</p>";
    }
    if (!$ok) {
        http_response_code(($_SERVER['REQUEST_METHOD'] ?? '') === 'POST' ? 401 : 200);
        $page("<form class='card login' method=post action='?r=/admin'><h1>GhostSync 管理</h1><div class=sub>只显示元数据（用户、设备、存储、分享数量），从不显示任何解密内容。</div>$err<input type=password name=token placeholder='管理令牌 GHOST_ADMIN_TOKEN' autofocus autocomplete=off><button>进入</button></form>");
    }
    $h = function ($s) { return htmlspecialchars((string)$s, ENT_QUOTES, 'UTF-8'); };
    $size = function (int $n) { $u = ['B', 'KB', 'MB', 'GB', 'TB']; $i = 0; $f = (float)$n; while ($f >= 1024 && $i < 4) { $f /= 1024; $i++; } return ($i ? number_format($f, 1) : (string)$n) . ' ' . $u[$i]; };
    $ago = function (int $t) { if (!$t) return '—'; $d = time() - $t; if ($d < 60) return '刚刚'; if ($d < 3600) return intdiv($d, 60) . ' 分钟前'; if ($d < 86400) return intdiv($d, 3600) . ' 小时前'; return intdiv($d, 86400) . ' 天前'; };
    $users = ['owner' => $c['base']];
    foreach (gs_users() as $t => $n) if (is_string($t) && strlen($t) >= 32) $users[(string)$n] = gs_user_dir($c, $t);
    $shdir = $c['base'] . '/_shared';
    $shares = gs_jread($shdir . '/shares.json');
    $directory = gs_jread($shdir . '/directory.json');
    $rows = '';
    $tot = ['bytes' => 0, 'dev' => 0, 'prof' => 0, 'share' => 0];
    foreach ($users as $name => $dir) {
        $bytes = 0;
        foreach (['blobs', 'chunks', 'refs', 'tmp'] as $sub) $bytes += gs_dir_size($dir . '/' . $sub);
        $idx = gs_jread($dir . '/index.json');
        $live = 0; $tomb = 0;
        foreach ($idx['profiles'] ?? [] as $r) { if (!empty($r['deleted'])) $tomb++; elseif ((int)($r['version'] ?? 0) > 0) $live++; }
        $reg = gs_jread($dir . '/devices.json');
        $da = $dp = $dr = 0; $seen = 0;
        foreach ($reg['devices'] ?? [] as $d) { if (!empty($d['revoked'])) $dr++; elseif (!empty($d['approved'])) $da++; else $dp++; $seen = max($seen, (int)$d['last_seen']); }
        $so = $si = 0;
        foreach ($shares as $s) { $st = gs_share_status($s); if (!in_array($st, ['pending', 'accepted'], true)) continue; if ($s['from'] === $name) $so++; if ($s['to'] === $name) $si++; }
        $keys = is_file($dir . '/keys.json');
        $tot['bytes'] += $bytes; $tot['dev'] += $da + $dp; $tot['prof'] += $live; $tot['share'] += $so;
        $rows .= '<tr><td><b>' . $h($name) . '</b></td><td>' . $size($bytes) . "</td><td>$live" . ($tomb ? " <span class=mut>(+$tomb 已删除)</span>" : '') . '</td>'
            . "<td><span class='b ok'>$da</span> " . ($dp ? "<span class='b warn'>$dp 待批准</span> " : '') . ($dr ? "<span class='b bad'>$dr 已吊销</span>" : '') . '</td>'
            . '<td>' . ($keys ? "<span class='b ok'>已设置</span>" : "<span class=mut>未设置</span>") . (isset($directory[$name]) ? " <span class='b ok'>可分享</span>" : '') . '</td>'
            . "<td>$so / $si</td><td class=mut>" . $ago($seen) . (!empty($reg['settings']['requireApproval']) ? " · 需批准" : '') . '</td></tr>';
    }
    $errs = '';
    $lines = is_file($shdir . '/errors.jsonl') ? (file($shdir . '/errors.jsonl', FILE_IGNORE_NEW_LINES) ?: []) : [];
    foreach (array_reverse(array_slice($lines, -40)) as $l) {
        $e = json_decode($l, true);
        if (!is_array($e)) continue;
        $cls = (int)$e['status'] >= 500 ? 'bad' : 'warn';
        $errs .= '<tr><td class=mut>' . $h(date('m-d H:i:s', (int)$e['t'])) . '</td><td>' . $h($e['user']) . "</td><td><span class='b $cls'>" . (int)$e['status'] . '</span></td><td>' . $h($e['route']) . '</td><td class=msg>' . $h($e['msg']) . '</td></tr>';
    }
    if ($errs === '') $errs = "<tr><td colspan=5 class=empty>最近没有错误</td></tr>";
    $page("<div class=wrap><h1>GhostSync 管理</h1><div class=sub>仅元数据 · 数据均为客户端加密，服务器无法解密 · " . $h(date('Y-m-d H:i')) . "</div>"
        . "<div class=grid><div class=card><div class=k>用户</div><div class=v>" . count($users) . "</div></div><div class=card><div class=k>存储占用</div><div class=v>" . $size($tot['bytes']) . "</div></div>"
        . "<div class=card><div class=k>云端环境</div><div class=v>{$tot['prof']}</div></div><div class=card><div class=k>有效设备 / 进行中分享</div><div class=v>{$tot['dev']} / {$tot['share']}</div></div></div>"
        . "<h2>用户</h2><table><tr><th>用户</th><th>存储</th><th>环境</th><th>设备</th><th>密钥</th><th>分享 发出/收到</th><th>最近活动</th></tr>$rows</table>"
        . "<h2>最近错误</h2><table><tr><th>时间</th><th>用户</th><th>状态</th><th>接口</th><th>信息</th></tr>$errs</table>"
        . "<div class=note>令牌、密码、密钥与环境内容从不显示，也不会写入本页。</div></div>");
}


try {
    gs_serve();
} catch (GsFail $e) {
    gs_sec_log($e->status, $e->getMessage());
    gs_reply($e->status, ['error' => $e->getMessage()]);
} catch (Throwable $e) {
    error_log('ghostsync: ' . $e);
    gs_sec_log(500, 'Server error');
    gs_reply(500, ['error' => 'Server error: ' . $e->getMessage()]);
}
