<?php
require (getenv('MEDIA_SOURCE_FILE') ?: '/var/www/app/application/imaccess/service/MediaSourceService.php');
$service = new \app\imaccess\service\MediaSourceService();
$id = str_repeat('a', 64);
$secret = 'fixture-only';
$head = $service->resolve(['url' => 'https://fixture.invalid', 'secret' => $secret], $id, 'fixture-signature', true);
if (strpos($head['url'], '/api/bridge/media?') === false || !$head['headers']) throw new RuntimeException('HEAD must use metadata, not a GET-only COS signature');
function ticket($data, $key) {
    $payload = json_encode($data);
    return json_encode(['payload' => $payload, 'signature' => hash_hmac('sha256', "hw-bridge-media-source-v1\n" . $payload, $key)]);
}
$valid = ['id' => $id, 'mode' => 'public', 'expires' => time() + 60,
    'url' => 'https://cdn.example.invalid/file.mp4', 'addresses' => ['8.8.8.8']];
$out = $service->verify(ticket($valid, $secret), $secret, $id);
if (!$out || $out['headers'] !== [] || $out['resolve'] !== ['cdn.example.invalid:443:8.8.8.8']) throw new RuntimeException('Valid ticket not pinned or credential forwarded');
$bad = [
    array_merge($valid, ['expires' => time() - 1]),
    array_merge($valid, ['expires' => time() + 1000]),
    array_merge($valid, ['addresses' => ['127.0.0.1']]),
    array_merge($valid, ['addresses' => ['8.8.8.8', '10.0.0.1']]),
    array_merge($valid, ['addresses' => ['169.254.169.254']]),
    array_merge($valid, ['url' => 'http://cdn.example.invalid/file']),
    array_merge($valid, ['url' => 'https://user:pass@cdn.example.invalid/file']),
    array_merge($valid, ['url' => 'https://cdn.example.invalid:8443/file']),
    array_merge($valid, ['id' => str_repeat('b', 64)]),
];
foreach ($bad as $v) if ($service->verify(ticket($v, $secret), $secret, $id) !== null) throw new RuntimeException('Unsafe ticket accepted');
if ($service->verify(ticket($valid, 'tampered'), $secret, $id) !== null) throw new RuntimeException('Forged ticket accepted');
echo "MEDIA_SOURCE_CONTRACT_OK\n";
