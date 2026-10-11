<?php

namespace think;

$root = dirname(__DIR__, 3);
require $root . '/thinkphp/base.php';
Container::get('app')->path($root . '/application/')->initialize();

use app\imaccess\service\SessionService;
use app\imaccess\service\TencentAccountClient;
use think\Db;

$config = config('im_access.');
if (empty($config['sdk_app_id']) || empty($config['secret_key'])) {
    fwrite(STDERR, "IM_SDK_APP_ID and IM_SDK_SECRET_KEY are required.\n");
    exit(2);
}
$user = Db::name('user')->where('login_status', 1)->where('login_token', '<>', '')->find();
if (!$user) {
    fwrite(STDERR, "No active local user is available.\n");
    exit(2);
}

$oldSig = $user['user_sig'];
$result = (new SessionService())->issue($user['login_token']);
if ($result['http_status'] !== 200) {
    fwrite(STDERR, 'SESSION FAIL ' . $result['body']['msg'] . ' trace=' . $result['body']['trace_id'] . PHP_EOL);
    exit(1);
}
$data = $result['body']['data'];
$check = (new TencentAccountClient($config))->accountCheck($data['im_user_id']);
$status = $check['ResultItem'][0]['AccountStatus'] ?? '';
$mappingCount = Db::name('im_account_mapping')->where([
    'uid' => $user['uid'], 'sdk_app_id' => (string) $config['sdk_app_id'],
])->count();
$legacyUnchanged = Db::name('user')->where('uid', $user['uid'])->value('user_sig') === $oldSig;

fwrite(STDOUT, 'SESSION PASS uid=' . $user['uid'] . ' im_user_id=' . $data['im_user_id'] . PHP_EOL);
fwrite(STDOUT, 'ACCOUNT CHECK ' . $status . PHP_EOL);
fwrite(STDOUT, 'MAPPING COUNT ' . $mappingCount . PHP_EOL);
fwrite(STDOUT, 'LEGACY SIG UNCHANGED ' . ($legacyUnchanged ? 'yes' : 'no') . PHP_EOL);
exit($status === 'Imported' && $mappingCount === 1 && $legacyUnchanged ? 0 : 1);
