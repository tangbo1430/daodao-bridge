<?php

namespace think;

$root = dirname(__DIR__, 3);
require $root . '/thinkphp/base.php';
Container::get('app')->path($root . '/application/')->initialize();

use app\api\model\User;
use app\imaccess\service\BlocklistService;
use think\Db;

$tests = 0;
$failures = 0;
function checkRecent($condition, $message)
{
    global $tests, $failures;
    $tests++;
    if ($condition) {
        fwrite(STDOUT, "PASS {$message}\n");
    } else {
        $failures++;
        fwrite(STDOUT, "FAIL {$message}\n");
    }
}

$suffix = (string) random_int(100000, 999999);
$ownerUid = random_int(700000, 799999);
$peerUid = random_int(800000, 899999);
$ownerToken = md5('owner-' . $suffix);
$now = time();

try {
    foreach ([
        [$ownerUid, 'owner_' . $suffix, $ownerToken],
        [$peerUid, 'peer_' . $suffix, md5('peer-' . $suffix)],
    ] as $row) {
        Db::name('user')->insert([
            'uid' => $row[0],
            'user_name' => $row[1],
            'password' => md5('oldpass1'),
            'nick_name' => $row[1],
            'base64_nick_name' => base64_encode($row[1]),
            'hobby' => '',
            'login_status' => 1,
            'login_token' => $row[2],
            'system_type' => 3,
            'add_time' => $now,
            'update_time' => $now,
        ]);
        Db::name('im_account_mapping')->insert([
            'uid' => $row[0],
            'im_user_id' => (string) $row[0],
            'sdk_app_id' => 'regression-' . $suffix,
            'import_status' => 'imported',
            'created_at' => $now,
            'updated_at' => $now,
        ]);
    }

    $blocklist = new BlocklistService();
    $blocked = $blocklist->change($ownerToken, (string) $peerUid, true);
    checkRecent($blocked['body']['data']['blocked'] === true, 'app block enabled');
    $blocklist->change($ownerToken, (string) $peerUid, true);
    checkRecent(Db::name('im_relation_snapshot')->where([
        'owner_account' => (string) $ownerUid,
        'peer_account' => (string) $peerUid,
        'relation_type' => BlocklistService::RELATION_TYPE,
    ])->count() === 1, 'duplicate app block is idempotent');
    checkRecent($blocklist->status($ownerToken, (string) $peerUid)['body']['data']['blocked'] === true, 'app block status visible');

    $blocklist->change($ownerToken, (string) $peerUid, false);
    checkRecent($blocklist->status($ownerToken, (string) $peerUid)['body']['data']['blocked'] === false, 'app block removed');

    Db::name('user_black')->insert(['uid' => $ownerUid, 'receive_uid' => $peerUid, 'add_time' => $now]);
    checkRecent($blocklist->status($ownerToken, (string) $peerUid)['body']['data']['blocked'] === true, 'legacy block status visible');
    checkRecent(count($blocklist->listing($ownerToken)['body']['data']) === 1, 'legacy block appears in listing');
    $blocklist->change($ownerToken, (string) $peerUid, false);
    checkRecent(Db::name('user_black')->where(['uid' => $ownerUid, 'receive_uid' => $peerUid])->count() === 0, 'legacy block removed');

    $userModel = new User();
    $empty = $userModel->modify_password_account($ownerUid, '', 'oldpass1');
    checkRecent((int) $empty['code'] !== 200, 'empty new password rejected');
    $wrong = $userModel->modify_password_account($ownerUid, 'newpass1', 'wrongpass');
    checkRecent((int) $wrong['code'] !== 200, 'wrong old password rejected');
    $wrongSame = $userModel->modify_password_account($ownerUid, 'wrongpass', 'wrongpass');
    checkRecent($wrongSame['msg'] === '旧密码不正确', 'wrong old password is checked before same-password validation');
    checkRecent(Db::name('user')->where('uid', $ownerUid)->value('login_token') === $ownerToken, 'failed password change keeps token');
    $changed = $userModel->modify_password_account($ownerUid, 'newpass1', 'oldpass1');
    checkRecent((int) $changed['code'] === 200, 'valid password change succeeds');
    checkRecent(Db::name('user')->where('uid', $ownerUid)->value('login_token') === '', 'successful password change clears token');

    $groupModel = model('api/GroupV');
    $method = new \ReflectionMethod($groupModel, 'normalizeInitialMembers');
    $method->setAccessible(true);
    $memberValidation = $method->invoke($groupModel, $ownerUid, ['testbo321']);
    checkRecent(
        isset($memberValidation['members']) && $memberValidation['members'] === ['testbo321'],
        'alphanumeric Tencent member account accepted'
    );

    $rawMemberIds = '["20426"]';
    request()->withPost(['member_ids' => $rawMemberIds]);
    checkRecent(
        request()->post('member_ids', '', null) === $rawMemberIds,
        'member JSON bypasses global HTML filtering'
    );
    $groupControllerClass = new \ReflectionClass(\app\api\controller\GroupV::class);
    $groupController = $groupControllerClass->newInstanceWithoutConstructor();
    $parseMemberIds = $groupControllerClass->getMethod('parseMemberIds');
    $parseMemberIds->setAccessible(true);
    checkRecent(
        $parseMemberIds->invoke($groupController, $rawMemberIds) === ['20426'],
        'Android member JSON is parsed into Tencent account IDs'
    );
    checkRecent(
        $parseMemberIds->invoke($groupController, [$rawMemberIds]) === ['20426'],
        'wrapped member JSON remains compatible'
    );

    $tencentModel = model('api/Tencent');
    $bootstrapPayloadMethod = new \ReflectionMethod($tencentModel, 'buildGroupCreatedMessagePayload');
    $bootstrapPayloadMethod->setAccessible(true);
    $bootstrapPayload = $bootstrapPayloadMethod->invoke($tencentModel, 'group-100', '20426');
    checkRecent(
        $bootstrapPayload['GroupId'] === 'group-100'
        && $bootstrapPayload['From_Account'] === '20426'
        && ($bootstrapPayload['MsgBody'][0]['MsgType'] ?? '') === 'TIMTextElem'
        && ($bootstrapPayload['MsgBody'][0]['MsgContent']['Text'] ?? '') === '群聊已创建',
        'group bootstrap message creates a persisted Tencent conversation'
    );

    $deletedGroupId = random_int(900000, 999999);
    Db::name('group_v')->insert([
        'id' => $deletedGroupId,
        'uid' => $ownerUid,
        'group_name' => 'deleted regression group',
        'add_time' => $now,
        'update_time' => $now,
        'is_delete' => 2,
        'delete_time' => $now,
        'im_g_id' => (string) $deletedGroupId,
        'open_protect' => 2,
        'system_type' => 3,
        'client_request_id' => 'deleted_' . $suffix,
    ]);
    $activeGroupCount = new \ReflectionMethod($groupModel, 'countActiveGroupsByOwner');
    $activeGroupCount->setAccessible(true);
    checkRecent(
        $activeGroupCount->invoke($groupModel, $ownerUid) === 0,
        'soft-deleted group does not consume creation quota'
    );
} finally {
    Db::name('group_v')->where('uid', $ownerUid)->delete();
    Db::name('im_relation_snapshot')->where('owner_account', (string) $ownerUid)->delete();
    Db::name('user_black')->where('uid', $ownerUid)->delete();
    Db::name('im_account_mapping')->where('sdk_app_id', 'regression-' . $suffix)->delete();
    Db::name('user')->where('uid', 'in', [$ownerUid, $peerUid])->delete();
}

fwrite(STDOUT, "Tests: {$tests}, Failures: {$failures}\n");
exit($failures > 0 ? 1 : 0);
