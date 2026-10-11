<?php

require dirname(__DIR__) . '/service/ClientIpResolver.php';

use app\common\service\ClientIpResolver;

$cases = [
    ['172.18.0.7', '203.0.113.25', '203.0.113.25', 'trusted Docker gateway'],
    ['198.51.100.7', '1.1.1.1', '198.51.100.7', 'forged header from public client'],
    ['127.0.0.1', '2001:db8::25', '2001:db8::25', 'trusted local gateway with IPv6 client'],
    ['172.18.0.7', 'not-an-ip', '172.18.0.7', 'invalid internal header'],
    ['', '1.1.1.1', '0.0.0.0', 'missing remote address'],
];

$failed = 0;
foreach ($cases as $case) {
    [$remote, $internal, $expected, $name] = $case;
    $actual = ClientIpResolver::resolve([
        'REMOTE_ADDR' => $remote,
        'HTTP_X_INTERNAL_CLIENT_IP' => $internal,
    ]);
    if ($actual !== $expected) {
        $failed++;
        fwrite(STDERR, "FAIL {$name}: expected {$expected}, got {$actual}\n");
    } else {
        fwrite(STDOUT, "PASS {$name}\n");
    }
}

exit($failed > 0 ? 1 : 0);
