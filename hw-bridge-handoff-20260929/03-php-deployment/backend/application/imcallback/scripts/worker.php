<?php

namespace think;

$root = dirname(__DIR__, 3);
require $root . '/thinkphp/base.php';

Container::get('app')->path($root . '/application/')->initialize();

$once = in_array('--once', $argv, true);
$limit = 100;
foreach ($argv as $argument) {
    if (strpos($argument, '--limit=') === 0) {
        $limit = max(1, (int) substr($argument, 8));
    }
}

$worker = new \app\imcallback\service\WorkerService();
do {
    $processed = $worker->processBatch($limit);
    fwrite(STDOUT, 'Processed events: ' . $processed . PHP_EOL);
    if (!$once && $processed === 0) sleep(1);
} while (!$once);
