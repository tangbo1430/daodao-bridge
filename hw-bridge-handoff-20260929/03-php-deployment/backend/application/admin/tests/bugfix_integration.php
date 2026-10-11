<?php

namespace think;

use think\Db;

if (getenv('BUGFIX_TEST_ALLOW') !== '1') {
    fwrite(STDERR, "Set BUGFIX_TEST_ALLOW=1 to run this isolated database test\n");
    exit(2);
}

$_SERVER['REMOTE_ADDR'] = '203.0.113.25';
$root = dirname(__DIR__, 3);
require $root . '/thinkphp/base.php';
Container::get('app')->path($root . '/application/')->initialize();

$failures = 0;
function bugfix_check($condition, $message)
{
    global $failures;
    if (!$condition) {
        $failures++;
        fwrite(STDERR, "FAIL: {$message}\n");
    }
}

$phone = '139' . substr((string) time(), -8);
$uid = 0;
$groupIds = ['bug_group_same', 'bug_group_snapshot'];
try {
    $userModel = model('admin/User');
    $created = $userModel->add_user_info($phone, 'Pass1234');
    bugfix_check($created['code'] === 200, 'admin creates a user');
    $uid = (int) ($created['data']['uid'] ?? 0);
    $user = Db::name('user')->where('uid', $uid)->find();
    bugfix_check((int) $user['system_type'] === (int) config('app.system_type'), 'new user uses current system type');
    bugfix_check((int) $user['is_show'] === 1, 'new user is visible');
    bugfix_check($user['head_pic'] === 'head_pic/head_pic.png', 'new user uses canonical avatar');
    $businessView = $userModel->get_user_info($uid);
    bugfix_check(strpos($businessView['data']['head_pic'], '/uploads/head_pic/head_pic.png') !== false, 'legacy default avatar stays on local uploads');
    bugfix_check($userModel->get_user_info(999999999)['code'] === 201, 'missing user detail is rejected');
    $invalidProfile = $userModel->edit_user_info($uid, '', 1, 1);
    bugfix_check($invalidProfile['code'] === 201 && strpos($invalidProfile['msg'], '昵称') !== false, 'invalid profile is rejected before Tencent sync');
    bugfix_check($userModel->add_user_info($phone, 'Pass1234')['code'] === 201, 'duplicate account is rejected');

    Db::name('user')->where('uid', $uid)->update(['login_token' => 'bugfix-token']);
    $changed = $userModel->edit_user_password(['uid' => $uid, 'password' => 'Next1234']);
    $user = Db::name('user')->where('uid', $uid)->find();
    bugfix_check($changed['code'] === 200, 'password update succeeds');
    bugfix_check($user['password'] === md5('Next1234'), 'password uses legacy-compatible hash');
    bugfix_check($user['login_token'] === '', 'password update invalidates login token');
    bugfix_check($userModel->edit_user_password(['uid' => 0, 'password' => 'Next1234'])['code'] === 201, 'missing user is rejected');

    Db::name('im_account_mapping')->insert([
        'uid' => $uid, 'im_user_id' => (string) $uid, 'sdk_app_id' => (string) (getenv('IM_SDK_APP_ID') ?: '20044723'),
        'import_status' => 'imported', 'import_error' => '', 'imported_at' => time(), 'created_at' => time(), 'updated_at' => time(),
    ]);
    Db::name('im_user_profile_snapshot')->insert([
        'account' => (string) $uid, 'nick_name' => 'Tencent Projection',
        'avatar_url' => 'https://cos.example.test/avatar.png', 'status' => 'active',
        'profile_json' => '{}', 'updated_at' => time(), 'source_event_time' => time() * 1000, 'source_event_id' => 1,
    ]);
    $view = $userModel->get_user_info($uid);
    bugfix_check($view['data']['nick_name'] === 'Tencent Projection', 'Tencent profile nickname has display priority');
    bugfix_check($view['data']['profile_source'] === 'tencent', 'profile source is exposed');

    $configModel = model('admin/Config');
    $config = Db::name('config')->find();
    $sameConfig = $configModel->edit_config([
        'cid' => $config['cid'], 'key_title' => $config['key_title'], 'key_name' => $config['key_name'],
        'key_value' => $config['key_value'], 'key_desc' => $config['key_desc'],
    ]);
    bugfix_check($sameConfig['code'] === 200, 'idempotent config edit succeeds');
    bugfix_check($configModel->config_info(99999999)['code'] === 201, 'missing config is rejected');

    Db::name('group_v')->insert([
        'id' => 91001, 'uid' => $uid, 'group_name' => 'Legacy duplicate',
        'add_time' => time() - 10, 'update_time' => time() - 10, 'im_g_id' => $groupIds[0],
        'is_delete' => 1, 'system_type' => (int) config('app.system_type'),
    ]);
    foreach ($groupIds as $index => $groupId) {
        Db::name('im_group_snapshot')->insert([
            'group_id' => $groupId, 'group_name' => $index === 0 ? 'Tencent wins' : 'Snapshot only',
            'owner_account' => (string) $uid, 'group_type' => 'Public', 'member_count' => 2 + $index,
            'status' => 'active', 'raw_json' => '{}', 'created_at' => time(), 'updated_at' => time(),
            'source_event_time' => time() * 1000, 'source_event_id' => $index + 1,
        ]);
    }
    $groups = model('admin/Group')->get_group_list('', $uid, '', 'update_time', 'desc', 1, 20);
    bugfix_check($groups['data']['count'] === 2, 'legacy and Tencent groups merge without duplicates');
    $byId = [];
    foreach ($groups['data']['list'] as $group) $byId[$group['group_id']] = $group;
    bugfix_check($byId[$groupIds[0]]['source'] === 'tencent', 'Tencent group snapshot has priority');
    bugfix_check($byId[$groupIds[1]]['group_name'] === 'Snapshot only', 'callback-only group is visible');
    $groupSearch = model('admin/Group')->get_group_list($groupIds[1], 0, '', 'update_time', 'desc', 1, 20);
    bugfix_check($groupSearch['data']['count'] === 1 && $groupSearch['data']['list'][0]['group_id'] === $groupIds[1], 'group id search finds callback projection');
} finally {
    foreach ($groupIds as $groupId) {
        Db::name('im_group_snapshot')->where('group_id', $groupId)->delete();
        Db::name('group_v')->where('im_g_id', $groupId)->delete();
    }
    if ($uid > 0) {
        Db::name('im_user_profile_snapshot')->where('account', (string) $uid)->delete();
        Db::name('im_account_mapping')->where('uid', $uid)->delete();
        Db::name('user')->where('uid', $uid)->delete();
    }
}

fwrite(STDOUT, $failures === 0 ? "Admin bugfix integration tests passed\n" : "Failures: {$failures}\n");
exit($failures === 0 ? 0 : 1);
