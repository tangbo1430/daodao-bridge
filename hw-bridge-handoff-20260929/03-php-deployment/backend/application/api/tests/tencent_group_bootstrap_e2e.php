<?php

$root = dirname(__DIR__, 3);
$appId = getenv('IM_SDK_APP_ID');
$secret = getenv('IM_SDK_SECRET_KEY');
$base = rtrim(getenv('IM_REST_BASE_URL') ?: 'https://adminapisgp.im.qcloud.com', '/') . '/v4';
$admin = getenv('IM_ADMIN_ACCOUNT') ?: 'administrator';
if (!$appId || !$secret) {
    fwrite(STDERR, "IM_SDK_APP_ID and IM_SDK_SECRET_KEY are required.\n");
    exit(2);
}

require $root . '/vendor/tencent/tls-sig-api-v2/src/TLSSigAPIv2.php';
$sigApi = new \Tencent\TLSSigAPIv2((int) $appId, $secret);
$userSig = $sigApi->genUserSig($admin);
$suffix = bin2hex(random_bytes(4));
$owner = 'bootstrap_owner_' . $suffix;
$member = 'bootstrap_member_' . $suffix;
$groupId = 'bootstrap_group_' . $suffix;

function bootstrapApi($base, $appId, $admin, $userSig, $service, $command, array $body)
{
    $query = http_build_query([
        'sdkappid' => $appId,
        'identifier' => $admin,
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
    $raw = curl_exec($curl);
    if ($raw === false) {
        throw new \RuntimeException(curl_error($curl));
    }
    $http = curl_getinfo($curl, CURLINFO_HTTP_CODE);
    curl_close($curl);
    $result = json_decode($raw, true);
    if ($http !== 200 || !is_array($result) || (int) ($result['ErrorCode'] ?? -1) !== 0) {
        throw new \RuntimeException($service . '/' . $command . ' failed: ' . $raw);
    }
    return $result;
}

try {
    foreach ([$owner, $member] as $account) {
        bootstrapApi($base, $appId, $admin, $userSig, 'im_open_login_svc', 'account_import', [
            'Identifier' => $account,
            'Nick' => $account,
        ]);
    }
    bootstrapApi($base, $appId, $admin, $userSig, 'group_open_http_svc', 'create_group', [
        'Owner_Account' => $owner,
        'Type' => 'Public',
        'GroupId' => $groupId,
        'Name' => 'Bootstrap E2E',
        'MemberList' => [['Member_Account' => $member]],
    ]);
    $message = bootstrapApi($base, $appId, $admin, $userSig, 'group_open_http_svc', 'send_group_msg', [
        'GroupId' => $groupId,
        'From_Account' => $owner,
        'Random' => random_int(100000, 999999999),
        'MsgBody' => [[
            'MsgType' => 'TIMTextElem',
            'MsgContent' => ['Text' => '群聊已创建'],
        ]],
    ]);
    if ((int) ($message['MsgSeq'] ?? 0) <= 0) {
        throw new \RuntimeException('Tencent did not return a persisted group message sequence.');
    }
    fwrite(STDOUT, 'PASS Tencent persisted bootstrap group message, MsgSeq=' . $message['MsgSeq'] . PHP_EOL);
} finally {
    try {
        bootstrapApi($base, $appId, $admin, $userSig, 'group_open_http_svc', 'destroy_group', ['GroupId' => $groupId]);
    } catch (\Throwable $ignored) {
    }
    try {
        bootstrapApi($base, $appId, $admin, $userSig, 'im_open_login_svc', 'account_delete', [
            'DeleteItem' => [['UserID' => $owner], ['UserID' => $member]],
        ]);
    } catch (\Throwable $ignored) {
    }
}
