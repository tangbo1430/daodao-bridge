<?php

namespace think;

$root = dirname(__DIR__, 3);
require $root . '/thinkphp/base.php';
Container::get('app')->path($root . '/application/')->initialize();

use app\imcallback\service\WorkerService;
use think\Db;

$token = getenv('IM_CALLBACK_TEST_ADMIN_TOKEN');
$base = getenv('IM_CALLBACK_TEST_BASE_URL') ?: 'http://127.0.0.1:8001/index.php?s=/imcallback/admin/';
if (!$token) { fwrite(STDERR, "IM_CALLBACK_TEST_ADMIN_TOKEN is required.\n"); exit(2); }
$tests = 0; $failures = 0; $eventId = 0; $groupId = 'admin_retry_' . bin2hex(random_bytes(4));
function adminCheck($condition, $message) { global $tests, $failures; $tests++; if ($condition) fwrite(STDOUT, "PASS {$message}\n"); else { $failures++; fwrite(STDOUT, "FAIL {$message}\n"); } }
function adminPost($url, array $data) {
    $ch = curl_init($url); curl_setopt_array($ch, [CURLOPT_RETURNTRANSFER=>true,CURLOPT_POST=>true,CURLOPT_POSTFIELDS=>http_build_query($data),CURLOPT_TIMEOUT=>10]);
    $raw = curl_exec($ch); $status = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE); curl_close($ch);
    return [$status, json_decode((string) $raw, true)];
}
try {
    $now = time();
    $payload = ['CallbackCommand'=>'Group.CallbackAfterCreateGroup','SdkAppid'=>config('im_callback.sdk_app_id'),'GroupId'=>$groupId,'Name'=>'Admin Retry Group','EventTime'=>$now * 1000];
    $eventId = Db::name('im_callback_event')->insertGetId([
        'event_key'=>hash('sha256',$groupId),'trace_id'=>'admin-test-'.bin2hex(random_bytes(4)),'request_id'=>'admin-test-'.$groupId,
        'callback_command'=>$payload['CallbackCommand'],'event_category'=>'group','sdk_app_id'=>config('im_callback.sdk_app_id'),'event_time'=>(string)$payload['EventTime'],
        'from_account'=>'','to_account'=>'','group_id'=>$groupId,'msg_seq'=>'','msg_random'=>'','payload_json'=>json_encode($payload),'summary'=>'','response_json'=>null,
        'handler_status'=>'failed','queue_status'=>'dead','retry_count'=>5,'manual_retry_count'=>0,'next_retry_at'=>0,'error_message'=>'Unit dead event',
        'duplicate_count'=>0,'received_at'=>$now,'last_received_at'=>$now,'processed_at'=>$now,'duration_ms'=>1,
    ]);
    list($status, $retry) = adminPost($base.'retry', ['login_token'=>$token,'id'=>$eventId]);
    adminCheck($status===200 && $retry['code']===0, 'dead event accepted for manual retry');
    $queued = Db::name('im_callback_event')->where('id',$eventId)->find();
    adminCheck($queued['queue_status']==='pending' && (int)$queued['manual_retry_count']===1 && (int)$queued['retry_count']===0, 'retry resets automatic count and increments manual count');
    (new WorkerService())->processOne($eventId);
    adminCheck(Db::name('im_callback_event')->where('id',$eventId)->value('queue_status')==='done', 'retried event processed once');
    adminCheck(Db::name('im_group_snapshot')->where('group_id',$groupId)->count()===1, 'retried event projected');
    adminCheck(Db::name('im_callback_admin_action')->where(['event_id'=>$eventId,'action'=>'manual_retry'])->count()===1, 'manual retry audited');
    list($status, $detail) = adminPost($base.'detail', ['login_token'=>$token,'id'=>$eventId,'include_raw'=>1]);
    adminCheck($status===200 && isset($detail['data']['payload']['GroupId']), 'raw event can be viewed by super admin');
    adminCheck(Db::name('im_callback_admin_action')->where(['event_id'=>$eventId,'action'=>'view_raw'])->count()===1, 'raw view audited');
} finally {
    if ($eventId) { Db::name('im_callback_admin_action')->where('event_id',$eventId)->delete(); Db::name('im_callback_event')->where('id',$eventId)->delete(); }
    Db::name('im_group_snapshot')->where('group_id',$groupId)->delete();
}
fwrite(STDOUT, "Tests: {$tests}, Failures: {$failures}\n"); exit($failures?1:0);
