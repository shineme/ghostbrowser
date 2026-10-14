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
 *   GET  /api/update/latest           -> update/latest.json next to this file (in-app updater; 404 when none)
 *   GET  /api/update/file/{name}      -> update/{name} (the signed EXE named in latest.json)
 *   GET  /api/catalog                 -> catalog.json next to this file (site shortcuts + announcements)
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
        foreach (GHOST_USERS as $tok => $name) {
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
    gs_app_routes($path, $method);

    if ($path === '/api/ping' && $method === 'GET') {
        $limits = array_filter([$c['max'], gs_ini_bytes('post_max_size')]);
        $free = @disk_free_space($c['dir']);
        gs_reply(200, ['ok' => true, 'service' => 'ghostbrowser-sync', 'protocol' => 2, 'single' => true, 'time' => time(),
            'user' => $user, 'features' => ['chunks', 'history', 'multiuser', 'tombstones'], 'keepVersions' => $c['keep'],
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

// ---- app shell: in-app updater + site shortcuts catalog / announcements ----
// Static files next to this script: update/latest.json, update/<exe>, catalog.json.
// The client verifies the EXE's sha256 and Ed25519 signature; this server only serves files.
function gs_app_routes(string $path, string $method): void
{
    if ($method !== 'GET') return;
    if ($path === '/api/update/latest') {
        $f = __DIR__ . '/update/latest.json';
        if (!is_file($f)) throw new GsFail(404, '服务器上还没有发布更新（缺少 update/latest.json）。');
        $doc = json_decode((string)file_get_contents($f), true);
        if (!is_array($doc) || !isset($doc['version'])) throw new GsFail(500, 'update/latest.json 格式不对。');
        gs_reply(200, $doc);
    }
    if (strpos($path, '/api/update/file/') === 0) {
        $name = substr($path, strlen('/api/update/file/'));
        if (!preg_match('/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/', $name) || $name === 'latest.json') throw new GsFail(400, 'Bad file name.');
        $f = __DIR__ . '/update/' . $name;
        if (!is_file($f)) throw new GsFail(404, 'File not found.');
        header('Content-Type: application/octet-stream');
        header('Content-Length: ' . filesize($f));
        header('Content-Disposition: attachment; filename="' . $name . '"');
        header('Cache-Control: no-store');
        readfile($f);
        exit;
    }
    if ($path === '/api/catalog') {
        $f = __DIR__ . '/catalog.json';
        $doc = is_file($f) ? json_decode((string)file_get_contents($f), true) : null;
        if (is_file($f) && !is_array($doc)) throw new GsFail(500, 'catalog.json 格式不对。');
        if (!is_array($doc)) $doc = ['revision' => 0, 'groups' => [], 'announcements' => []];
        gs_reply(200, $doc);
    }
}

try {
    gs_serve();
} catch (GsFail $e) {
    gs_reply($e->status, ['error' => $e->getMessage()]);
} catch (Throwable $e) {
    error_log('ghostsync: ' . $e);
    gs_reply(500, ['error' => 'Server error: ' . $e->getMessage()]);
}
