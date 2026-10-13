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
 *   GET  /api/profiles
 *   POST /api/profiles/{id}/lock      {"device": "..."}
 *   POST /api/profiles/{id}/unlock
 *   PUT  /api/profiles/{id}/blob?version=N   (body = encrypted blob)
 *   POST /api/profiles/{id}/commit    {"id","name","version","sha256","meta"}
 *   GET  /api/profiles/{id}/blob
 *   POST /api/profiles/{id}/delete
 */

// ------------------------------------------------------------- settings
const GHOST_TOKEN = '';                 // 32+ random characters (or env GHOST_SYNC_TOKEN)
const GHOST_DATA_DIR = '';              // '' = a hidden folder next to this file (or env GHOST_SYNC_DATA_DIR)
const GHOST_MAX_BLOB_MB = 512;          // largest profile upload
const GHOST_LEASE_SECONDS = 600;        // how long an upload lock lives
const GHOST_KEEP_VERSIONS = 2;          // blobs kept per profile (current + previous)
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
        'keep' => max(1, (int)GHOST_KEEP_VERSIONS),
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
    foreach ([$c['dir'], $c['dir'] . '/blobs', $c['dir'] . '/tmp'] as $d) {
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
    $r['pending_path'] = $r['pending_sha'] = null;
    $r['pending_size'] = $r['pending_base'] = null;
}

function gs_view(array $r): array
{
    return ['id' => $r['id'], 'name' => $r['name'], 'version' => (int)$r['version'], 'sha256' => $r['sha256'],
        'meta' => $r['meta'] ?: new stdClass(), 'lockedBy' => (int)$r['lock_until'] > time() ? $r['lock_device'] : '',
        'updatedAt' => (int)$r['updated_at'], 'size' => (int)($r['size'] ?? 0), 'device' => (string)($r['device'] ?? ''), 'deviceName' => (string)($r['device_name'] ?? '')];
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
        foreach ($r['history'] ?? [] as $h) $live[$h] = true;
    }
    foreach (glob($c['dir'] . '/blobs/*.gbe') ?: [] as $f) {
        if (!isset($live[basename($f, '.gbe')])) @unlink($f);
    }
    foreach (glob($c['dir'] . '/tmp/*.part') ?: [] as $f) {
        if (filemtime($f) < time() - 86400) @unlink($f);
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
    $given = '';
    if (preg_match('/^Bearer\s+(.+)$/i', gs_header('Authorization'), $m)) $given = trim($m[1]);
    if ($given === '') $given = gs_header('X-Ghost-Token');
    if ($given === '' || !hash_equals($c['token'], $given)) {
        header('WWW-Authenticate: Bearer');
        throw new GsFail(401, 'Invalid token.');
    }
    gs_init_dir($c);

    if ($path === '/api/ping' && $method === 'GET') {
        gs_reply(200, ['ok' => true, 'service' => 'ghostbrowser-sync', 'protocol' => 1, 'single' => true, 'time' => time()]);
    }
    if ($path === '/api/profiles' && $method === 'GET') {
        $list = gs_tx($c, function (array &$idx) {
            $out = [];
            foreach ($idx['profiles'] as $r) if ((int)$r['version'] > 0) $out[] = gs_view($r);
            usort($out, function ($a, $b) { return [$b['updatedAt'], $a['id']] <=> [$a['updatedAt'], $b['id']]; });
            return $out;
        });
        gs_reply(200, $list);
    }
    if (!preg_match('~^/api/profiles/([a-zA-Z0-9_-]{1,64})/(lock|unlock|blob|commit|delete)$~D', $path, $m)) throw new GsFail(404, 'Not found.');
    $id = $m[1];
    $action = $m[2];
    $ttl = $c['ttl'];

    if ($action === 'lock' && $method === 'POST') {
        $b = gs_body();
        $device = $b['device'] ?? null;
        if (!is_string($device) || !preg_match('/^[a-zA-Z0-9_-]{1,128}$/D', $device)) throw new GsFail(400, 'Invalid device ID.');
        $ticket = bin2hex(random_bytes(32));
        $v = gs_tx($c, function (array &$idx) use ($c, $id, $device, $ticket, $ttl) {
            $r = $idx['profiles'][$id] ?? gs_blank($id);
            if ((int)$r['lock_until'] > time() && $r['lock_device'] !== $device) throw new GsFail(423, 'Profile is locked by another sync operation.');
            gs_clear_pending($c, $r);
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
    if ($action === 'commit' && $method === 'POST') {
        $b = gs_body();
        if (($b['id'] ?? '') !== $id || !is_int($b['version'] ?? null) || !is_string($b['sha256'] ?? null) ||
            !preg_match('/^[a-f0-9]{64}$/D', $b['sha256']) || !is_string($b['name'] ?? null) || strlen($b['name']) > 1024 ||
            (($b['meta']['encrypted'] ?? null) !== true)) throw new GsFail(400, 'Invalid commit metadata.');
        $idx = gs_tx($c, function (array &$idx) use ($c, $id, $b) {
            $r = gs_lease($idx, $id);
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
            $hist = $r['history'] ?? [];
            if ($r['sha256']) array_unshift($hist, $r['sha256']);
            $r['history'] = array_slice(array_values(array_unique($hist)), 0, max(0, $c['keep'] - 1));
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
            $idx['profiles'][$id] = $r;
            return $idx;
        });
        gs_gc($c, $idx);
        gs_reply(200, ['ok' => true, 'version' => $b['version']]);
    }
    if ($action === 'blob' && $method === 'GET') {
        $r = gs_tx($c, function (array &$idx) use ($id) { return $idx['profiles'][$id] ?? null; });
        if (!$r || (int)$r['version'] === 0) throw new GsFail(404, 'Profile has no committed blob.');
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
        $idx = gs_tx($c, function (array &$idx) use ($c, $id) {
            $r = $idx['profiles'][$id] ?? null;
            if ($r && (int)$r['lock_until'] > time()) throw new GsFail(423, 'Profile is being synced.');
            if ($r) { gs_clear_pending($c, $r); unset($idx['profiles'][$id]); }
            return $idx;
        });
        gs_gc($c, $idx);
        gs_reply(200, ['ok' => true]);
    }
    header('Allow: ' . ($action === 'blob' ? 'GET, PUT' : 'POST'));
    throw new GsFail(405, 'Method not allowed.');
}

try {
    gs_serve();
} catch (GsFail $e) {
    gs_reply($e->status, ['error' => $e->getMessage()]);
} catch (Throwable $e) {
    error_log('ghostsync: ' . $e);
    gs_reply(500, ['error' => 'Server error: ' . $e->getMessage()]);
}
