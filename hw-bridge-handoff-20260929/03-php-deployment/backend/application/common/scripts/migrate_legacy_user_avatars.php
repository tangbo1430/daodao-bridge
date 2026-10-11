<?php

namespace think;

use app\common\service\UploadStorageConfig;
use Qcloud\Cos\Client;
use think\Db;

$root = dirname(__DIR__, 3);
require $root . '/thinkphp/base.php';
Container::get('app')->path($root . '/application/')->initialize();

$execute = in_array('--execute', $argv, true);
$limit = 500;
foreach ($argv as $argument) {
    if (strpos($argument, '--limit=') === 0) {
        $limit = max(1, (int) substr($argument, 8));
    }
}

$storage = new UploadStorageConfig(get_system_config());
$cos = $storage->cos();
if ($execute && $storage->missingCosSettings()) {
    fwrite(STDERR, "COS configuration is incomplete\n");
    exit(2);
}

$client = $execute ? new Client([
    'region' => $cos['region'],
    'schema' => 'https',
    'allow_accelerate' => $cos['accelerate'],
    'credentials' => ['secretId' => $cos['secret_id'], 'secretKey' => $cos['secret_key']],
]) : null;

$stats = ['scanned' => 0, 'migratable' => 0, 'migrated' => 0, 'missing' => 0, 'absolute' => 0, 'skipped_default' => 0];
$rows = Db::name('user')->field('uid,head_pic')->limit($limit)->select();
foreach ($rows as $row) {
    $stats['scanned']++;
    $path = str_replace('\\', '/', trim((string) $row['head_pic']));
    if ($path === '' || $path === 'head_pic/head_pic.png' || $path === 'head_pic/head_pic1.png') {
        $stats['skipped_default']++;
        continue;
    }
    if (preg_match('#^https?://#i', $path)) {
        $stats['absolute']++;
        fwrite(STDOUT, json_encode(['uid' => (int) $row['uid'], 'status' => 'absolute_preserved', 'url' => $path], JSON_UNESCAPED_SLASHES) . PHP_EOL);
        continue;
    }

    $relative = ltrim($path, '/');
    if (strpos($relative, 'uploads/') === 0) $relative = substr($relative, 8);
    $localFile = $root . '/public/uploads/' . $relative;
    if (!is_file($localFile)) {
        $stats['missing']++;
        fwrite(STDOUT, json_encode(['uid' => (int) $row['uid'], 'status' => 'missing', 'path' => $relative], JSON_UNESCAPED_SLASHES) . PHP_EOL);
        continue;
    }

    $stats['migratable']++;
    if (!$execute) {
        fwrite(STDOUT, json_encode(['uid' => (int) $row['uid'], 'status' => 'dry_run', 'path' => $relative], JSON_UNESCAPED_SLASHES) . PHP_EOL);
        continue;
    }

    $extension = strtolower(pathinfo($localFile, PATHINFO_EXTENSION));
    $key = 'legacy/user/' . (int) $row['uid'] . '/avatar' . ($extension === '' ? '' : '.' . $extension);
    $stream = fopen($localFile, 'rb');
    try {
        $client->putObject(['Bucket' => $cos['bucket'], 'Key' => $key, 'Body' => $stream]);
    } finally {
        if (is_resource($stream)) fclose($stream);
    }
    $url = $storage->publicUrl($key);
    Db::name('user')->where('uid', (int) $row['uid'])->update(['head_pic' => $url, 'update_time' => time()]);
    $stats['migrated']++;
    fwrite(STDOUT, json_encode(['uid' => (int) $row['uid'], 'status' => 'migrated', 'url' => $url], JSON_UNESCAPED_SLASHES) . PHP_EOL);
}

fwrite(STDOUT, json_encode(['mode' => $execute ? 'execute' : 'dry_run', 'stats' => $stats], JSON_UNESCAPED_SLASHES) . PHP_EOL);
