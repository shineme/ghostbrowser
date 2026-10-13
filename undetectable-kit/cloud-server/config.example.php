<?php
// Copy to config.php, or use the GHOST_SYNC_* environment variables instead.
// config.php and data/ must remain outside the public document root.
return [
    'token' => '', // Generate: php -r "echo bin2hex(random_bytes(32)), PHP_EOL;"
    'data_dir' => __DIR__ . '/data',
    'max_blob_bytes' => 268435456,
    'lease_seconds' => 600,
    'base_path' => '', // e.g. /ghostsync if hosted beneath a URL prefix
];
