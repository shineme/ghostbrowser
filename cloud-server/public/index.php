<?php
declare(strict_types=1);

// GhostBrowser private sync service. The server stores encrypted GBE1 blobs;
// profile details, proxy credentials and cookies remain inside that ciphertext.
ini_set('display_errors', '0');
require dirname(__DIR__) . '/server.php';

try {
    serve();
} catch (HttpFailure $e) {
    reply($e->status, ['error' => $e->getMessage()]);
} catch (Throwable $e) {
    error_log('GhostBrowser sync: ' . $e->getMessage());
    reply(500, ['error' => 'Internal server error; inspect the PHP server log.']);
}
