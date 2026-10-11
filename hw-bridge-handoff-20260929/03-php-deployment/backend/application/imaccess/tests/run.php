<?php

namespace think;

$root = dirname(__DIR__, 3);
require $root . '/thinkphp/base.php';
Container::get('app')->path($root . '/application/')->initialize();

use app\imaccess\service\SessionService;
use think\Db;

class FakeTencentAccountClient
{
    public $imports = 0;
    public function importAccount($imUserId, $nickName = '', $faceUrl = '')
    {
        $this->imports++;
        return ['ActionStatus' => 'OK', 'ErrorCode' => 0];
    }
}

$tests = 0;
$failures = 0;
function checkAccess($condition, $message)
{
    global $tests, $failures;
    $tests++;
    if ($condition) fwrite(STDOUT, "PASS {$message}\n");
    else { $failures++; fwrite(STDOUT, "FAIL {$message}\n"); }
}

$user = Db::name('user')->where('login_status', 1)->where('login_token', '<>', '')->find();
if (!$user) {
    fwrite(STDERR, "No active local test user with login_token.\n");
    exit(2);
}

$appId = (string) random_int(900000000, 999999999);
$config = [
    'sdk_app_id' => $appId,
    'secret_key' => str_repeat('a', 64),
    'sig_ttl' => 86400,
    'admin_account' => 'administrator',
    'rest_base_url' => 'https://console.tim.qq.com',
];
$fake = new FakeTencentAccountClient();
$service = new SessionService($config, $fake);
$uid = (int) $user['uid'];
$oldSig = $user['user_sig'];

try {
    $invalid = $service->issue('invalid-unit-token');
    checkAccess($invalid['http_status'] === 401, 'invalid token rejected');

    $forbidden = $service->issue($user['login_token'], $uid + 1);
    checkAccess($forbidden['http_status'] === 403, 'cross-user signing rejected');

    $first = $service->issue($user['login_token']);
    checkAccess($first['http_status'] === 200, 'session issued');
    $firstData = isset($first['body']['data']) && is_array($first['body']['data']) ? $first['body']['data'] : [];
    checkAccess(isset($firstData['im_user_id']) && $firstData['im_user_id'] === (string) $uid, 'uid used as stable IM user id');
    checkAccess(isset($firstData['expires_in']) && $firstData['expires_in'] === 86400, '24 hour expiry returned');

    $second = $service->issue($user['login_token']);
    checkAccess($second['http_status'] === 200, 'repeat session issued');
    checkAccess($fake->imports === 1, 'Tencent account import is idempotent');
    checkAccess(Db::name('im_account_mapping')->where(['uid' => $uid, 'sdk_app_id' => $appId])->count() === 1, 'mapping remains unique');
    checkAccess(Db::name('user')->where('uid', $uid)->value('user_sig') === $oldSig, 'legacy user_sig is unchanged');

    $missingConfig = new SessionService(array_merge($config, ['secret_key' => '']), $fake);
    $missing = $missingConfig->issue($user['login_token']);
    checkAccess($missing['http_status'] === 502, 'missing configuration rejected');
} finally {
    Db::name('im_account_mapping')->where('sdk_app_id', $appId)->delete();
    Db::name('im_sig_audit')->where('sdk_app_id', $appId)->delete();
}

fwrite(STDOUT, "Tests: {$tests}, Failures: {$failures}\n");
exit($failures > 0 ? 1 : 0);
