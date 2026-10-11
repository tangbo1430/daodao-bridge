<?php

namespace think;

$root = dirname(__DIR__, 3);
require $root . '/thinkphp/base.php';
Container::get('app')->path($root . '/application/')->initialize();

use think\Db;

$appId = getenv('IM_SDK_APP_ID');
$secret = getenv('IM_SDK_SECRET_KEY');
$region = getenv('IM_REST_REGION') ?: 'sgp';
if (!$appId || !$secret) {
    fwrite(STDERR, "IM_SDK_APP_ID and IM_SDK_SECRET_KEY are required.\n");
    exit(2);
}

$domains = [
    'sgp' => 'https://adminapisgp.im.qcloud.com/v4',
    'china' => 'https://console.tim.qq.com/v4',
];
if (!isset($domains[$region])) {
    fwrite(STDERR, "Unsupported IM_REST_REGION.\n");
    exit(2);
}

require $root . '/vendor/tencent/tls-sig-api-v2/src/TLSSigAPIv2.php';
$sigApi = new \Tencent\TLSSigAPIv2((int) $appId, $secret);
$userSig = $sigApi->genUserSig('administrator');
$base = $domains[$region];
$run = 'e2e_' . date('YmdHis') . '_' . bin2hex(random_bytes(3));
$accountA = 'callback_e2e_a';
$accountB = 'callback_e2e_b';
$groupId = $run . '_group';
$startedAt = time();
$tests = [];
$failures = 0;

function api($base, $appId, $userSig, $service, $command, array $body)
{
    $caFile = getenv('SSL_CERT_FILE');
    $query = http_build_query([
        'sdkappid' => $appId,
        'identifier' => 'administrator',
        'usersig' => $userSig,
        'random' => random_int(10000000, 99999999),
        'contenttype' => 'json',
    ]);
    $curl = curl_init($base . '/' . $service . '/' . $command . '?' . $query);
    curl_setopt_array($curl, [
        CURLOPT_POST => true,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER => ['Content-Type: application/json'],
        CURLOPT_POSTFIELDS => json_encode($body, JSON_UNESCAPED_UNICODE),
        CURLOPT_CONNECTTIMEOUT => 10,
        CURLOPT_TIMEOUT => 30,
    ]);
    if ($caFile) {
        curl_setopt($curl, CURLOPT_CAINFO, $caFile);
    }
    $raw = curl_exec($curl);
    if ($raw === false) throw new \RuntimeException(curl_error($curl));
    $http = curl_getinfo($curl, CURLINFO_HTTP_CODE);
    curl_close($curl);
    $result = json_decode($raw, true);
    if ($http !== 200 || !is_array($result) || ($result['ErrorCode'] ?? -1) !== 0) {
        throw new \RuntimeException($service . '/' . $command . ' failed: ' . $raw);
    }
    return $result;
}

function action($name, callable $callback)
{
    global $failures;
    try {
        $result = $callback();
        fwrite(STDOUT, "ACTION OK   {$name}\n");
        return $result;
    } catch (\Throwable $e) {
        $failures++;
        fwrite(STDOUT, "ACTION FAIL {$name}: {$e->getMessage()}\n");
        return null;
    }
}

function ensureAccount($base, $appId, $userSig, $account)
{
    $check = api($base, $appId, $userSig, 'im_open_login_svc', 'account_check', [
        'CheckItem' => [['UserID' => $account]],
    ]);
    $status = $check['ResultItem'][0]['AccountStatus'] ?? '';
    if ($status === 'Imported') return $check;
    return api($base, $appId, $userSig, 'im_open_login_svc', 'account_import', [
        'Identifier' => $account, 'Nick' => $account,
    ]);
}

function expectCallback($command, $label)
{
    global $tests;
    $tests[$command] = $label;
}

function waitForCallbacks($startedAt, $tests, $accountA, $accountB, $groupId)
{
    $deadline = time() + 35;
    $found = [];
    while (time() < $deadline) {
        $rows = Db::name('im_callback_event')
            ->where('received_at', '>=', $startedAt)
            ->where(function ($q) use ($accountA, $accountB, $groupId) {
                $q->where('from_account', 'in', [$accountA, $accountB])
                    ->whereOr('to_account', 'in', [$accountA, $accountB])
                    ->whereOr('group_id', $groupId)
                    ->whereOr('payload_json', 'like', '%' . $accountA . '%')
                    ->whereOr('payload_json', 'like', '%' . $accountB . '%')
                    ->whereOr('payload_json', 'like', '%' . $groupId . '%');
            })->select();
        foreach ($rows as $row) $found[$row['callback_command']] = $row;
        if (count(array_intersect_key($tests, $found)) === count($tests)) break;
        usleep(500000);
    }
    return $found;
}

try {
    action('import account A', function () use ($base, $appId, $userSig, $accountA) {
        return ensureAccount($base, $appId, $userSig, $accountA);
    });
    action('import account B', function () use ($base, $appId, $userSig, $accountB) {
        return ensureAccount($base, $appId, $userSig, $accountB);
    });

    $send = action('send direct message', function () use ($base, $appId, $userSig, $accountA, $accountB) {
        return api($base, $appId, $userSig, 'openim', 'sendmsg', [
            'From_Account' => $accountA, 'To_Account' => $accountB, 'MsgRandom' => random_int(100000, 999999),
            'MsgBody' => [['MsgType' => 'TIMTextElem', 'MsgContent' => ['Text' => 'Tencent callback E2E']]],
        ]);
    });
    expectCallback('C2C.CallbackBeforeSendMsg', 'direct message before');
    expectCallback('C2C.CallbackAfterSendMsg', 'direct message after');

    if ($send && !empty($send['MsgKey'])) {
        action('withdraw direct message', function () use ($base, $appId, $userSig, $accountA, $accountB, $send) {
            return api($base, $appId, $userSig, 'openim', 'admin_msgwithdraw', [
                'From_Account' => $accountA, 'To_Account' => $accountB, 'MsgKey' => $send['MsgKey'],
            ]);
        });
        expectCallback('C2C.CallbackAfterMsgWithDraw', 'direct message withdraw');
    }

    action('create group', function () use ($base, $appId, $userSig, $groupId, $accountA) {
        return api($base, $appId, $userSig, 'group_open_http_svc', 'create_group', [
            'Owner_Account' => $accountA, 'Type' => 'Public', 'GroupId' => $groupId, 'Name' => 'E2E Group',
        ]);
    });
    expectCallback('Group.CallbackBeforeCreateGroup', 'group before create');
    expectCallback('Group.CallbackAfterCreateGroup', 'group after create');

    action('add group member', function () use ($base, $appId, $userSig, $groupId, $accountB) {
        return api($base, $appId, $userSig, 'group_open_http_svc', 'add_group_member', [
            'GroupId' => $groupId, 'MemberList' => [['Member_Account' => $accountB]],
        ]);
    });
    expectCallback('Group.CallbackAfterNewMemberJoin', 'group member join');

    action('send group message', function () use ($base, $appId, $userSig, $groupId, $accountA) {
        return api($base, $appId, $userSig, 'group_open_http_svc', 'send_group_msg', [
            'GroupId' => $groupId, 'From_Account' => $accountA, 'Random' => random_int(100000, 999999),
            'MsgBody' => [['MsgType' => 'TIMTextElem', 'MsgContent' => ['Text' => 'Group callback E2E']]],
        ]);
    });
    expectCallback('Group.CallbackBeforeSendMsg', 'group message before');
    expectCallback('Group.CallbackAfterSendMsg', 'group message after');

    action('modify group name', function () use ($base, $appId, $userSig, $groupId) {
        return api($base, $appId, $userSig, 'group_open_http_svc', 'modify_group_base_info', ['GroupId' => $groupId, 'Name' => 'E2E Group Updated']);
    });
    expectCallback('Group.CallbackAfterGroupInfoChanged', 'group info changed');

    action('add friend', function () use ($base, $appId, $userSig, $accountA, $accountB) {
        return api($base, $appId, $userSig, 'sns', 'friend_add', [
            'From_Account' => $accountA, 'AddFriendItem' => [['To_Account' => $accountB, 'AddSource' => 'AddSource_Type_Web']],
            'AddType' => 'Add_Type_Both', 'ForceAddFlags' => 1,
        ]);
    });
    expectCallback('Sns.CallbackPrevFriendAdd', 'friend before add');
    expectCallback('Sns.CallbackFriendAdd', 'friend after add');

    action('add blacklist', function () use ($base, $appId, $userSig, $accountA, $accountB) {
        return api($base, $appId, $userSig, 'sns', 'black_list_add', ['From_Account' => $accountA, 'To_Account' => [$accountB]]);
    });
    expectCallback('Sns.CallbackBlackListAdd', 'blacklist add');

    action('set profile', function () use ($base, $appId, $userSig, $accountA) {
        return api($base, $appId, $userSig, 'profile', 'portrait_set', [
            'From_Account' => $accountA,
            'ProfileItem' => [['Tag' => 'Tag_Profile_IM_Nick', 'Value' => 'E2E User A']],
        ]);
    });
    expectCallback('Profile.CallbackPortraitSet', 'profile changed');

    $found = waitForCallbacks($startedAt, $tests, $accountA, $accountB, $groupId);
    foreach ($tests as $command => $label) {
        if (isset($found[$command])) {
            fwrite(STDOUT, "CALLBACK PASS {$label} ({$command})\n");
        } else {
            $failures++;
            fwrite(STDOUT, "CALLBACK FAIL {$label} ({$command})\n");
        }
    }

    fwrite(STDOUT, "CALLBACK SKIP online state requires a real SDK login/logout client\n");
} finally {
    action('destroy test group', function () use ($base, $appId, $userSig, $groupId) {
        return api($base, $appId, $userSig, 'group_open_http_svc', 'destroy_group', ['GroupId' => $groupId]);
    });
}

fwrite(STDOUT, 'Failures: ' . $failures . PHP_EOL);
exit($failures > 0 ? 1 : 0);
