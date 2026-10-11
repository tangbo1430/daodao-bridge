<?php

namespace think;

$root = dirname(__DIR__, 3);
$runId = 'unit-' . date('YmdHis') . '-' . bin2hex(random_bytes(4));
$prefix = 'unit_' . bin2hex(random_bytes(4));
$queueKey = 'im:callback:test:' . $runId;
putenv('IM_CALLBACK_QUEUE_KEY=' . $queueKey);

require $root . '/thinkphp/base.php';
Container::get('app')->path($root . '/application/')->initialize();

use app\imcallback\handler\HandlerRegistry;
use app\imcallback\service\CallbackService;
use app\imcallback\service\Payload;
use app\imcallback\service\WorkerService;
use think\Db;

$token = getenv('IM_CALLBACK_TOKEN');
$appId = getenv('IM_SDK_APP_ID');
if (!$token || !$appId) {
    fwrite(STDERR, "IM_CALLBACK_TOKEN and IM_SDK_APP_ID are required.\n");
    exit(2);
}

$tests = 0;
$failures = 0;
$eventIds = [];

function check($condition, $message)
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

function callback(CallbackService $service, $token, $appId, $runId, $command, array $body, $suffix)
{
    $query = [
        'CallbackCommand' => $command,
        'SdkAppid' => $appId,
        'RequestId' => $runId . '-' . $suffix,
        'OptPlatform' => 'UnitTest',
    ];
    return $service->receive(
        json_encode($body, JSON_UNESCAPED_UNICODE),
        $token,
        CallbackService::createTraceId(),
        $query
    );
}

try {
    $service = new CallbackService();
    $worker = new WorkerService();

    check(Payload::messageSummary([
        'MsgBody' => [['MsgType' => 'TIMTextElem', 'MsgContent' => ['Text' => 'hello']]],
    ]) === 'hello', 'text message summary');
    check(HandlerRegistry::resolve('C2C.CallbackBeforeSendMsg')->isAsync() === false, 'before event is synchronous');
    check(HandlerRegistry::resolve('Group.CallbackAfterCreateGroup')->category() === 'group', 'group event category');
    check(HandlerRegistry::resolve('Robot.CallbackAfterRobotEvent') === null, 'unsupported event remains unregistered');

    $invalidJson = $service->receive('{bad', $token, CallbackService::createTraceId(), [
        'CallbackCommand' => 'C2C.CallbackAfterSendMsg', 'SdkAppid' => $appId,
    ]);
    check($invalidJson['http_status'] === 400, 'invalid JSON is rejected');

    $badToken = $service->receive('{}', 'wrong-token', CallbackService::createTraceId(), []);
    check($badToken['http_status'] === 403, 'invalid token is rejected');

    $messageBody = [
        'From_Account' => $prefix . '_a',
        'To_Account' => $prefix . '_b',
        'MsgSeq' => 1001,
        'MsgRandom' => 2001,
        'MsgTime' => time(),
        'MsgBody' => [['MsgType' => 'TIMTextElem', 'MsgContent' => ['Text' => 'unit message']]],
    ];
    $first = callback($service, $token, $appId, $runId, 'C2C.CallbackAfterSendMsg', $messageBody, 'message');
    $second = callback($service, $token, $appId, $runId, 'C2C.CallbackAfterSendMsg', $messageBody, 'message');
    check($first['body']['ErrorCode'] === 0 && $second['body']['ErrorCode'] === 0, 'message callback accepts duplicates');
    $messageEvent = Db::name('im_callback_event')->where('request_id', $runId . '-message')->find();
    $eventIds[] = $messageEvent['id'];
    check((int) $messageEvent['duplicate_count'] === 1, 'duplicate increments without a second event');
    check($messageEvent['queue_status'] === 'pending', 'async event is pending before worker');
    check($worker->processOne($messageEvent['id']) === true, 'worker processes message event');
    check(Db::name('im_callback_event')->where('id', $messageEvent['id'])->value('queue_status') === 'done', 'message event becomes done');

    $before = callback($service, $token, $appId, $runId, 'C2C.CallbackBeforeSendMsg', $messageBody, 'before');
    check($before['body']['ErrorCode'] === 0, 'unknown test accounts fail open in before policy');

    callback($service, $token, $appId, $runId, 'Group.CallbackAfterCreateGroup', [
        'GroupId' => $prefix . '_group', 'Name' => 'Unit Group', 'Owner_Account' => $prefix . '_a', 'Type' => 'Public',
    ], 'group-create');
    $groupEvent = Db::name('im_callback_event')->where('request_id', $runId . '-group-create')->find();
    $eventIds[] = $groupEvent['id'];
    check($worker->processOne($groupEvent['id']) === true, 'group projector executes');
    check(Db::name('im_group_snapshot')->where('group_id', $prefix . '_group')->value('group_name') === 'Unit Group', 'group projection persisted');

    callback($service, $token, $appId, $runId, 'Group.CallbackAfterNewMemberJoin', [
        'GroupId' => $prefix . '_group',
        'NewMemberList' => [['Member_Account' => $prefix . '_b', 'Role' => 'Member']],
    ], 'group-member');
    $memberEvent = Db::name('im_callback_event')->where('request_id', $runId . '-group-member')->find();
    $eventIds[] = $memberEvent['id'];
    $worker->processOne($memberEvent['id']);
    check(Db::name('im_group_member_snapshot')->where(['group_id' => $prefix . '_group', 'account' => $prefix . '_b'])->count() === 1, 'member projection persisted');

    callback($service, $token, $appId, $runId, 'Sns.CallbackFriendAdd', [
        'PairList' => [['From_Account' => $prefix . '_a', 'To_Account' => $prefix . '_b']],
    ], 'friend');
    $friendEvent = Db::name('im_callback_event')->where('request_id', $runId . '-friend')->find();
    $eventIds[] = $friendEvent['id'];
    $worker->processOne($friendEvent['id']);
    check(Db::name('im_relation_snapshot')->where(['owner_account' => $prefix . '_a', 'relation_type' => 'friend'])->count() === 1, 'friend projection persisted');

    callback($service, $token, $appId, $runId, 'Sns.CallbackBlackListAdd', [
        'PairList' => [['From_Account' => $prefix . '_a', 'To_Account' => $prefix . '_b']],
    ], 'blacklist');
    $blacklistEvent = Db::name('im_callback_event')->where('request_id', $runId . '-blacklist')->find();
    $eventIds[] = $blacklistEvent['id'];
    check($blacklistEvent['queue_status'] === 'done', 'blacklist projection does not wait for worker');
    check(Db::name('im_relation_snapshot')->where([
        'owner_account' => $prefix . '_a',
        'peer_account' => $prefix . '_b',
        'relation_type' => 'blacklist',
        'status' => 'active',
    ])->count() === 1, 'blacklist projection persisted');
    $blockedByPeer = callback($service, $token, $appId, $runId, 'C2C.CallbackBeforeSendMsg', [
        'From_Account' => $prefix . '_b',
        'To_Account' => $prefix . '_a',
        'MsgSeq' => 1002,
        'MsgRandom' => 2002,
        'MsgTime' => time(),
        'MsgBody' => [['MsgType' => 'TIMTextElem', 'MsgContent' => ['Text' => 'blocked message']]],
    ], 'blacklist-before-peer');
    $blockedByPeerEvent = Db::name('im_callback_event')->where('request_id', $runId . '-blacklist-before-peer')->find();
    $eventIds[] = $blockedByPeerEvent['id'];
    check((int) $blockedByPeer['body']['ErrorCode'] === 120001, 'blocked user cannot message blacklist owner');

    $blockedBySelf = callback($service, $token, $appId, $runId, 'C2C.CallbackBeforeSendMsg', [
        'From_Account' => $prefix . '_a',
        'To_Account' => $prefix . '_b',
        'MsgSeq' => 1003,
        'MsgRandom' => 2003,
        'MsgTime' => time(),
        'MsgBody' => [['MsgType' => 'TIMTextElem', 'MsgContent' => ['Text' => 'blocked reverse message']]],
    ], 'blacklist-before-self');
    $blockedBySelfEvent = Db::name('im_callback_event')->where('request_id', $runId . '-blacklist-before-self')->find();
    $eventIds[] = $blockedBySelfEvent['id'];
    check((int) $blockedBySelf['body']['ErrorCode'] === 120001, 'blacklist owner cannot message blocked user');

    callback($service, $token, $appId, $runId, 'Sns.CallbackBlackListDelete', [
        'PairList' => [['From_Account' => $prefix . '_a', 'To_Account' => $prefix . '_b']],
    ], 'blacklist-delete');
    $blacklistDeleteEvent = Db::name('im_callback_event')->where('request_id', $runId . '-blacklist-delete')->find();
    $eventIds[] = $blacklistDeleteEvent['id'];
    check($blacklistDeleteEvent['queue_status'] === 'done', 'blacklist removal projection does not wait for worker');
    $allowedAfterDelete = callback($service, $token, $appId, $runId, 'C2C.CallbackBeforeSendMsg', [
        'From_Account' => $prefix . '_a',
        'To_Account' => $prefix . '_b',
        'MsgSeq' => 1004,
        'MsgRandom' => 2004,
        'MsgTime' => time(),
        'MsgBody' => [['MsgType' => 'TIMTextElem', 'MsgContent' => ['Text' => 'allowed after unblock']]],
    ], 'blacklist-after-delete');
    $allowedAfterDeleteEvent = Db::name('im_callback_event')->where('request_id', $runId . '-blacklist-after-delete')->find();
    $eventIds[] = $allowedAfterDeleteEvent['id'];
    check((int) $allowedAfterDelete['body']['ErrorCode'] === 0, 'messages resume after blacklist removal');

    Db::name('im_relation_snapshot')->insert([
        'owner_account' => $prefix . '_a',
        'peer_account' => $prefix . '_b',
        'relation_type' => 'app_blocklist',
        'status' => 'active',
        'raw_json' => '{}',
        'updated_at' => time(),
    ]);
    $appBlocked = callback($service, $token, $appId, $runId, 'C2C.CallbackBeforeSendMsg', [
        'From_Account' => $prefix . '_a',
        'To_Account' => $prefix . '_b',
        'MsgSeq' => 1005,
        'MsgRandom' => 2005,
        'MsgTime' => time(),
        'MsgBody' => [['MsgType' => 'TIMTextElem', 'MsgContent' => ['Text' => 'app blocked message']]],
    ], 'app-blocklist-before');
    $appBlockedEvent = Db::name('im_callback_event')->where('request_id', $runId . '-app-blocklist-before')->find();
    $eventIds[] = $appBlockedEvent['id'];
    check((int) $appBlocked['body']['ErrorCode'] === 120001, 'app blocklist rejects owner messages without deleting friendship');
    Db::name('im_relation_snapshot')->where([
        'owner_account' => $prefix . '_a',
        'peer_account' => $prefix . '_b',
        'relation_type' => 'app_blocklist',
    ])->update(['status' => 'deleted', 'updated_at' => time()]);
    $appUnblocked = callback($service, $token, $appId, $runId, 'C2C.CallbackBeforeSendMsg', [
        'From_Account' => $prefix . '_a',
        'To_Account' => $prefix . '_b',
        'MsgSeq' => 1006,
        'MsgRandom' => 2006,
        'MsgTime' => time(),
        'MsgBody' => [['MsgType' => 'TIMTextElem', 'MsgContent' => ['Text' => 'app unblocked message']]],
    ], 'app-blocklist-after-delete');
    $appUnblockedEvent = Db::name('im_callback_event')->where('request_id', $runId . '-app-blocklist-after-delete')->find();
    $eventIds[] = $appUnblockedEvent['id'];
    check((int) $appUnblocked['body']['ErrorCode'] === 0, 'app blocklist removal restores messaging');

    callback($service, $token, $appId, $runId, 'Profile.CallbackPortraitSet', [
        'From_Account' => $prefix . '_a',
        'ProfileItem' => [['Tag' => 'Tag_Profile_IM_Nick', 'Value' => 'Unit A']],
    ], 'profile');
    $profileEvent = Db::name('im_callback_event')->where('request_id', $runId . '-profile')->find();
    $eventIds[] = $profileEvent['id'];
    $worker->processOne($profileEvent['id']);
    check(Db::name('im_user_profile_snapshot')->where('account', $prefix . '_a')->value('nick_name') === 'Unit A', 'profile projection persisted');

    callback($service, $token, $appId, $runId, 'State.StateChange', [
        'Info' => [['To_Account' => $prefix . '_a', 'Action' => 'Login', 'Reason' => 'UnitTest']],
        'EventTime' => 1000,
    ], 'state');
    $stateEvent = Db::name('im_callback_event')->where('request_id', $runId . '-state')->find();
    $eventIds[] = $stateEvent['id'];
    $worker->processOne($stateEvent['id']);
    check(Db::name('im_user_state_snapshot')->where('account', $prefix . '_a')->value('state') === 'Login', 'state projection persisted');

    (new \app\imcallback\projector\StateProjector())->project([], [
        'Info' => ['To_Account' => $prefix . '_a', 'Action' => 'Logout', 'Reason' => 'RealFormat'],
        'EventTime' => 2000,
    ]);
    check(Db::name('im_user_state_snapshot')->where('account', $prefix . '_a')->value('state') === 'Logout', 'real state callback object is projected');
    (new \app\imcallback\projector\StateProjector())->project([], [
        'Info' => ['To_Account' => $prefix . '_a', 'Action' => 'Login', 'Reason' => 'OlderEvent'],
        'EventTime' => 1500,
    ]);
    check(Db::name('im_user_state_snapshot')->where('account', $prefix . '_a')->value('state') === 'Logout', 'older state event does not overwrite current state');

    callback($service, $token, $appId, $runId, 'Group.CallbackAfterCreateGroup', ['Name' => 'Invalid'], 'retry');
    $retryEvent = Db::name('im_callback_event')->where('request_id', $runId . '-retry')->find();
    $eventIds[] = $retryEvent['id'];
    check($worker->processOne($retryEvent['id']) === false, 'projection failure is captured');
    $retryState = Db::name('im_callback_event')->where('id', $retryEvent['id'])->find();
    check($retryState['queue_status'] === 'failed' && (int) $retryState['retry_count'] === 1, 'failed event schedules retry');

    callback($service, $token, $appId, $runId, 'Robot.CallbackAfterRobotEvent', ['Value' => 1], 'unknown');
    $unknown = Db::name('im_callback_event')->where('request_id', $runId . '-unknown')->find();
    $eventIds[] = $unknown['id'];
    check($unknown['handler_status'] === 'ignored', 'unknown callback is recorded as ignored');
} catch (\Throwable $e) {
    $failures++;
    fwrite(STDOUT, 'ERROR ' . $e->getMessage() . PHP_EOL . $e->getTraceAsString() . PHP_EOL);
} finally {
    Db::name('im_callback_event')->where('request_id', 'like', $runId . '%')->delete();
    Db::name('im_group_member_snapshot')->where('group_id', 'like', $prefix . '%')->delete();
    Db::name('im_group_snapshot')->where('group_id', 'like', $prefix . '%')->delete();
    Db::name('im_relation_snapshot')->where('owner_account', 'like', $prefix . '%')->delete();
    Db::name('im_user_profile_snapshot')->where('account', 'like', $prefix . '%')->delete();
    Db::name('im_user_state_snapshot')->where('account', 'like', $prefix . '%')->delete();
    try {
        $redis = new \Redis();
        $redis->connect('127.0.0.1', 6379, 1.0);
        $redis->del($queueKey);
    } catch (\Throwable $e) {
    }
}

fwrite(STDOUT, sprintf("Tests: %d, Failures: %d\n", $tests, $failures));
exit($failures > 0 ? 1 : 0);
