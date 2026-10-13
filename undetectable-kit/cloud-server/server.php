<?php
declare(strict_types=1);

final class HttpFailure extends RuntimeException
{
    public function __construct(public int $status, string $message) { parent::__construct($message); }
}

function reply(int $status, mixed $value): never
{
    http_response_code($status);
    header('Content-Type: application/json; charset=utf-8');
    header('Cache-Control: no-store');
    header('X-Content-Type-Options: nosniff');
    echo json_encode($value, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_THROW_ON_ERROR);
    exit;
}

function bodyJson(): array
{
    $f = fopen('php://input', 'rb');
    $b = stream_get_contents($f, 131073);
    fclose($f);
    if ($b === false || strlen($b) > 131072) throw new HttpFailure(413, 'JSON body exceeds 128 KiB.');
    try { $value = json_decode($b, true, 64, JSON_THROW_ON_ERROR); }
    catch (JsonException) { throw new HttpFailure(400, 'Invalid JSON body.'); }
    if (!is_array($value) || array_is_list($value)) throw new HttpFailure(400, 'JSON body must be an object.');
    return $value;
}

function dbRow(PDO $db, string $id): ?array
{
    $q = $db->prepare('SELECT * FROM profiles WHERE id=?');
    $q->execute([$id]);
    $r = $q->fetch(PDO::FETCH_ASSOC);
    return $r === false ? null : $r;
}

function transaction(PDO $db, callable $fn): mixed
{
    $db->exec('BEGIN IMMEDIATE');
    try { $v = $fn(); $db->exec('COMMIT'); return $v; }
    catch (Throwable $e) { $db->exec('ROLLBACK'); throw $e; }
}

function lockRow(PDO $db, string $id): array
{
    $r = dbRow($db, $id);
    $ticket = $_SERVER['HTTP_X_LOCK_TOKEN'] ?? '';
    $device = $_SERVER['HTTP_X_DEVICE_ID'] ?? '';
    if (!$r || (int)$r['lock_until'] <= time() || $r['lock_device'] !== $device ||
        $ticket === '' || !hash_equals((string)$r['lock_hash'], hash('sha256', $ticket))) {
        throw new HttpFailure(423, 'A valid profile lease is required.');
    }
    return $r;
}

function clearPending(string $data, ?string $pending): void
{
    if ($pending && preg_match('/^[a-f0-9]{48}\.part$/D', $pending)) {
        $p = $data . '/tmp/' . $pending;
        if (is_file($p)) unlink($p);
    }
}

function metaView(array $r): array
{
    return ['id' => $r['id'], 'name' => $r['name'], 'version' => (int)$r['version'],
        'sha256' => $r['sha256'], 'meta' => json_decode($r['meta'], true, 32, JSON_THROW_ON_ERROR),
        'lockedBy' => (int)$r['lock_until'] > time() ? $r['lock_device'] : '',
        'updatedAt' => (int)$r['updated_at']];
}

function serve(): never
{
    $cfg = ['token' => '', 'data_dir' => __DIR__ . '/data', 'max_blob_bytes' => 268435456, 'lease_seconds' => 600, 'base_path' => ''];
    if (is_file(__DIR__ . '/config.php')) $cfg = array_replace($cfg, require __DIR__ . '/config.php');
    foreach (['GHOST_SYNC_TOKEN' => 'token', 'GHOST_SYNC_DATA_DIR' => 'data_dir',
        'GHOST_SYNC_MAX_BLOB_BYTES' => 'max_blob_bytes', 'GHOST_SYNC_LEASE_SECONDS' => 'lease_seconds',
        'GHOST_SYNC_BASE_PATH' => 'base_path'] as $env => $key) {
        $v = getenv($env);
        if ($v !== false && $v !== '') $cfg[$key] = $v;
    }
    $token = (string)$cfg['token'];
    if (strlen($token) < 32) throw new HttpFailure(503, 'Configure a random GHOST_SYNC_TOKEN (at least 32 characters).');
    $auth = $_SERVER['HTTP_AUTHORIZATION'] ?? $_SERVER['REDIRECT_HTTP_AUTHORIZATION'] ?? '';
    if (!preg_match('/^Bearer\s+(.+)$/iD', $auth, $m) || !hash_equals($token, $m[1])) {
        header('WWW-Authenticate: Bearer');
        throw new HttpFailure(401, 'Invalid bearer token.');
    }
    if (!extension_loaded('pdo_sqlite')) throw new HttpFailure(503, 'Enable PDO SQLite in PHP.');
    $data = (string)$cfg['data_dir'];
    foreach ([$data, $data . '/tmp', $data . '/blobs'] as $dir) {
        if (!is_dir($dir) && !mkdir($dir, 0700, true) && !is_dir($dir)) throw new RuntimeException('Cannot create storage directory.');
    }
    $data = realpath($data);
    $public = realpath(__DIR__ . '/public');
    if ($data === $public || str_starts_with(strtolower($data), strtolower($public . DIRECTORY_SEPARATOR))) {
        throw new HttpFailure(503, 'The storage directory must be outside the public web root.');
    }
    $max = max(48, (int)$cfg['max_blob_bytes']);
    $ttl = max(60, min(3600, (int)$cfg['lease_seconds']));
    $db = new PDO('sqlite:' . $data . '/sync.sqlite', null, null,
        [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_TIMEOUT => 10]);
    $db->exec('PRAGMA busy_timeout=10000');
    $db->exec('PRAGMA journal_mode=WAL');
    $db->exec('CREATE TABLE IF NOT EXISTS profiles (
        id TEXT PRIMARY KEY, name TEXT NOT NULL DEFAULT "", version INTEGER NOT NULL DEFAULT 0,
        sha256 TEXT NOT NULL DEFAULT "", meta TEXT NOT NULL DEFAULT "{}", updated_at INTEGER NOT NULL DEFAULT 0,
        lock_device TEXT NOT NULL DEFAULT "", lock_hash TEXT NOT NULL DEFAULT "", lock_until INTEGER NOT NULL DEFAULT 0,
        pending_path TEXT, pending_sha TEXT, pending_size INTEGER, pending_base INTEGER
    )');
    $method = $_SERVER['REQUEST_METHOD'];
    $path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
    $base = rtrim((string)$cfg['base_path'], '/');
    if ($base !== '') {
        if (!str_starts_with($path, $base . '/')) throw new HttpFailure(404, 'Not found.');
        $path = substr($path, strlen($base));
    }
    if ($path === '/api/ping' && $method === 'GET') reply(200, ['ok' => true, 'service' => 'ghostbrowser-sync', 'protocol' => 1]);
    if ($path === '/api/profiles' && $method === 'GET') {
        $out = [];
        foreach ($db->query('SELECT * FROM profiles WHERE version>0 ORDER BY updated_at DESC,id') as $r) $out[] = metaView($r);
        reply(200, $out);
    }
    if (!preg_match('~^/api/profiles/([a-zA-Z0-9_-]{1,64})/(lock|unlock|blob|commit)$~D', $path, $m)) {
        throw new HttpFailure(404, 'Not found.');
    }
    [$unused, $id, $action] = $m;
    if ($action === 'lock' && $method === 'POST') {
        $b = bodyJson();
        $device = $b['device'] ?? null;
        if (!is_string($device) || !preg_match('/^[a-zA-Z0-9_-]{1,128}$/D', $device)) throw new HttpFailure(400, 'Invalid device ID.');
        $ticket = bin2hex(random_bytes(32));
        $v = transaction($db, function () use ($db, $id, $device, $ticket, $ttl, $data) {
            $r = dbRow($db, $id);
            if ($r && (int)$r['lock_until'] > time()) throw new HttpFailure(423, 'Profile is locked by another sync operation.');
            if ($r) clearPending($data, $r['pending_path']);
            $q = $db->prepare('INSERT INTO profiles(id,lock_device,lock_hash,lock_until) VALUES(?,?,?,?)
                ON CONFLICT(id) DO UPDATE SET lock_device=excluded.lock_device,lock_hash=excluded.lock_hash,
                lock_until=excluded.lock_until,pending_path=NULL,pending_sha=NULL,pending_size=NULL,pending_base=NULL');
            $q->execute([$id, $device, hash('sha256', $ticket), time() + $ttl]);
            return (int)($r['version'] ?? 0);
        });
        reply(200, ['ok' => true, 'lockToken' => $ticket, 'version' => $v, 'expiresAt' => time() + $ttl]);
    }
    if ($action === 'unlock' && $method === 'POST') {
        transaction($db, function () use ($db, $id, $data) {
            $r = lockRow($db, $id);
            clearPending($data, $r['pending_path']);
            $db->prepare('UPDATE profiles SET lock_device="",lock_hash="",lock_until=0,
                pending_path=NULL,pending_sha=NULL,pending_size=NULL,pending_base=NULL WHERE id=?')->execute([$id]);
        });
        reply(200, ['ok' => true]);
    }
    if ($action === 'blob' && $method === 'PUT') {
        $version = $_GET['version'] ?? null;
        if (!is_string($version) || !preg_match('/^(0|[1-9][0-9]{0,15})$/D', $version)) throw new HttpFailure(400, 'A non-negative version is required.');
        $version = (int)$version;
        transaction($db, function () use ($db, $id, $version) {
            $r = lockRow($db, $id);
            if ((int)$r['version'] !== $version) throw new HttpFailure(412, 'A newer version exists; pull first.');
        });
        if (isset($_SERVER['CONTENT_LENGTH']) && (int)$_SERVER['CONTENT_LENGTH'] > $max) throw new HttpFailure(413, 'Blob is too large.');
        $name = bin2hex(random_bytes(24)) . '.part';
        $file = $data . '/tmp/' . $name;
        $in = fopen('php://input', 'rb');
        $out = fopen($file, 'xb');
        if ($in === false || $out === false) throw new RuntimeException('Cannot open upload stream.');
        $ctx = hash_init('sha256'); $size = 0; $header = '';
        try {
            while (!feof($in)) {
                $chunk = fread($in, 65536);
                if ($chunk === false) throw new RuntimeException('Upload read failed.');
                if ($chunk === '') continue;
                $size += strlen($chunk);
                if ($size > $max) throw new HttpFailure(413, 'Blob is too large.');
                if (strlen($header) < 4) $header .= substr($chunk, 0, 4 - strlen($header));
                hash_update($ctx, $chunk);
                if (fwrite($out, $chunk) !== strlen($chunk)) throw new RuntimeException('Upload write failed.');
            }
            fflush($out); fclose($out); $out = null; fclose($in); $in = null;
            if ($size < 48 || $header !== 'GBE1') throw new HttpFailure(400, 'Expected an encrypted GBE1 blob.');
            $sha = hash_final($ctx);
            transaction($db, function () use ($db, $id, $version, $name, $sha, $size, $ttl, $data) {
                $r = lockRow($db, $id);
                if ((int)$r['version'] !== $version) throw new HttpFailure(412, 'Version changed during upload.');
                clearPending($data, $r['pending_path']);
                $db->prepare('UPDATE profiles SET pending_path=?,pending_sha=?,pending_size=?,pending_base=?,lock_until=? WHERE id=?')
                    ->execute([$name, $sha, $size, $version, time() + $ttl, $id]);
            });
        } catch (Throwable $e) {
            if (is_resource($out)) fclose($out);
            if (is_resource($in)) fclose($in);
            if (is_file($file)) unlink($file);
            throw $e;
        }
        reply(200, ['ok' => true, 'sha256' => $sha, 'bytes' => $size]);
    }
    if ($action === 'commit' && $method === 'POST') {
        $b = bodyJson();
        if (($b['id'] ?? '') !== $id || !is_int($b['version'] ?? null) ||
            !is_string($b['sha256'] ?? null) || !preg_match('/^[a-f0-9]{64}$/D', $b['sha256']) ||
            !is_string($b['name'] ?? null) || strlen($b['name']) > 1024 ||
            ($b['meta']['encrypted'] ?? null) !== true) throw new HttpFailure(400, 'Invalid commit metadata.');
        transaction($db, function () use ($db, $id, $b, $data) {
            $r = lockRow($db, $id);
            if ((int)$r['version'] + 1 !== $b['version'] || (int)$r['pending_base'] !== (int)$r['version']) throw new HttpFailure(412, 'Version conflict.');
            if (!$r['pending_path'] || !hash_equals((string)$r['pending_sha'], $b['sha256'])) throw new HttpFailure(409, 'Upload SHA-256 does not match the commit.');
            $source = $data . '/tmp/' . $r['pending_path'];
            if (!is_file($source) || filesize($source) !== (int)$r['pending_size'] ||
                !hash_equals($b['sha256'], hash_file('sha256', $source))) throw new HttpFailure(409, 'Pending blob failed integrity validation.');
            $dest = $data . '/blobs/' . $b['sha256'] . '.gbe';
            if (!is_file($dest)) {
                if (!rename($source, $dest)) throw new RuntimeException('Cannot promote upload.');
            } else {
                if (!hash_equals($b['sha256'], hash_file('sha256', $dest))) throw new RuntimeException('Stored content-addressed blob is corrupt.');
                unlink($source);
            }
            $meta = json_encode(['encrypted' => true, 'format' => 'GBE1', 'profileMetadata' => 'in-blob'], JSON_THROW_ON_ERROR);
            $db->prepare('UPDATE profiles SET name=?,version=?,sha256=?,meta=?,updated_at=?,
                lock_device="",lock_hash="",lock_until=0,pending_path=NULL,pending_sha=NULL,pending_size=NULL,pending_base=NULL WHERE id=?')
                ->execute([$b['name'], $b['version'], $b['sha256'], $meta, time(), $id]);
        });
        reply(200, ['ok' => true, 'version' => $b['version']]);
    }
    if ($action === 'blob' && $method === 'GET') {
        $r = dbRow($db, $id);
        if (!$r || (int)$r['version'] === 0) throw new HttpFailure(404, 'Profile has no committed blob.');
        $p = $data . '/blobs/' . $r['sha256'] . '.gbe';
        if (!is_file($p)) throw new RuntimeException('Committed blob is missing.');
        $f = fopen($p, 'rb');
        header('Content-Type: application/octet-stream');
        header('Content-Length: ' . filesize($p));
        header('X-Profile-Version: ' . $r['version']);
        header('X-Blob-SHA256: ' . $r['sha256']);
        header('ETag: "' . $r['sha256'] . '"');
        header('Cache-Control: no-store');
        header('X-Content-Type-Options: nosniff');
        fpassthru($f); fclose($f); exit;
    }
    header('Allow: ' . ($action === 'blob' ? 'GET, PUT' : 'POST'));
    throw new HttpFailure(405, 'Method not allowed.');
}
