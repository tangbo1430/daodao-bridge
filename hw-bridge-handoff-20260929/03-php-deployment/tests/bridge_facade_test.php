<?php
// Standalone contract test: every database and HTTP operation is a local fixture.
namespace think {
    class Db {
        public static function name($name) { return new self(); }
        public function where($field, $value) { $this->token = $value; return $this; }
        public function find() { return in_array($this->token, ['fixture-login', 700001], true) ? ['uid' => 700001, 'login_status' => 1, 'base64_nick_name' => base64_encode('Fixture Profile'), 'head_pic' => 'fixture.png'] : null; }
    }
}
namespace think\facade { class Log { public static function error($message) {} } }
namespace app\imaccess\service {
    class RegistrationService { public function accountReady($uid) { return true; } }
    class SessionService { public function issue($token,$uid) { return ['http_status'=>200,'body'=>['data'=>['im_user_id'=>'fixture-owner']]]; } }
    class AccountMappingService { public function resolve($uid, $appId) { return ['im_user_id' => 'fixture-owner']; } }
    function config($name) { return 'fixture-sdk'; }
    function localpath_to_netpath($name) { return 'https://fixture.invalid/' . $name; }
    function curl_init($url) { $GLOBALS['request_url'] = $url; return 1; }
    function curl_setopt_array($handle, $options) { $GLOBALS['request_options'] = $options; return true; }
    function curl_exec($handle) { if (!empty($GLOBALS['fixture_responses'])) return array_shift($GLOBALS['fixture_responses']); return '{"code":200,"data":{"group_id":"dd_fixture-group"}}'; }
    function curl_error($handle) { return ''; }
    function curl_getinfo($handle, $key) { return 200; }
    function curl_close($handle) {}
}
namespace {
    require (getenv('BRIDGE_SERVICE_FILE') ?: '/var/www/app/application/imaccess/service/BridgeService.php');
    function check($condition, $message) { if (!$condition) throw new \RuntimeException($message); }
    $service = new \app\imaccess\service\BridgeService(['enabled' => true, 'url' => 'https://fixture.invalid', 'secret' => 'fixture-only', 'timeout' => 5]);
    $result = $service->createGroupForUser(700001, 'Fixture mixed group', ['fixture-peer'], 'fixture-request');
    $created = json_decode($GLOBALS['request_options'][CURLOPT_POSTFIELDS], true);
    check($created['tencent_user_id'] === 'fixture-owner' && $created['request_id'] === 'fixture-request', 'legacy group actor/request mapping lost');
    check($service->createGroupForUser(700002, 'Invalid owner', [], 'fixture-request')['http_status'] === 401, 'unavailable group owner accepted');
    $service->notifyGroupRecall(700001, 'dd_fixture-group', 42);
    $recall = json_decode($GLOBALS['request_options'][CURLOPT_POSTFIELDS], true);
    check($recall['tencent_user_id'] === 'fixture-owner' && $recall['message_key'] === 'dd_fixture-group:42', 'recall did not preserve authenticated operator/sequence');
    $result = $service->groupAction('', 'create', []);
    check($result['http_status'] === 401, 'empty login accepted');
    $result = $service->groupAction('fixture-login', 'create', ['tencent_user_id' => 'spoofed-owner', 'members' => ['fixture-peer'], 'request_id' => 'fixture-request']);
    check($result['http_status'] === 200, 'group request failed');
    $body = json_decode($GLOBALS['request_options'][CURLOPT_POSTFIELDS], true);
    check($body['tencent_user_id'] === 'fixture-owner', 'APK supplied owner was trusted');
    check($body['members'] === ['fixture-peer'], 'members changed');
    check($GLOBALS['request_options'][CURLOPT_TIMEOUT] >= 60, 'group timeout too short');
    check(!array_key_exists('login_token', $body), 'login token forwarded to Bridge');
    $result = $service->groupAction('fixture-login', 'invite', ['members' => [['unexpected' => 'object']]]);
    check($result['http_status'] === 400, 'invalid members accepted');
    $result = $service->friendAction('fixture-login', 'fixture-peer', 'refuse');
    $body = json_decode($GLOBALS['request_options'][CURLOPT_POSTFIELDS], true);
    check($body['tencent_user_id'] === 'fixture-owner', 'friend actor mapping missing');
    check(substr($GLOBALS['request_url'], -13) === 'friend_refuse', 'refuse route incorrect');
    $service->profileUser(700001);
    $body = json_decode($GLOBALS['request_options'][CURLOPT_POSTFIELDS], true);
    check($body['tencent_user_id'] === 'fixture-owner' && $body['display_name'] === 'Fixture Profile', 'profile source mapping incorrect');
    check($body['avatar_url'] === 'https://fixture.invalid/fixture.png', 'profile URL not normalized');
    $GLOBALS['fixture_responses']=['{"code":409,"msg":"native-private-error"}'];
    $result=$service->configForLoginToken('fixture-login');
    check($result['http_status']===409 && $result['body']['data']['bridge_enabled']===false, 'reserve refusal reported enabled');
    check(strpos(json_encode($result),'native-private-error')===false,'native error leaked to APK');
    $GLOBALS['fixture_responses']=['{"status":"reserved"}','{"code":409}'];
    $result=$service->configForLoginToken('fixture-login');
    check($result['http_status']===409 && $result['body']['data']['bridge_enabled']===false, 'failed confirmation reported enabled');
    echo "BRIDGE_FACADE_CONTRACT_OK\n";
    foreach (['', '<html>gateway error</html>', 'null', '[]', '{}', '"ok"'] as $bad) {
        $GLOBALS['fixture_responses']=[$bad];
        $rejected=false;
        try { $service->configForLoginToken('fixture-login'); } catch (\RuntimeException $e) { $rejected=true; }
        check($rejected, 'invalid upstream 200 accepted during account setup');
        $GLOBALS['fixture_responses']=[$bad];
        $rejected=false;
        try { $service->searchUser('fixture-login','fixture-peer'); } catch (\RuntimeException $e) { $rejected=true; }
        check($rejected, 'invalid upstream GET response accepted');
    }
    $GLOBALS['fixture_responses']=['{"users":[]}'];
    check($service->searchUser('fixture-login','missing')['body']['users']===[], 'valid empty search rejected');
    echo "BRIDGE_INVALID_200_FAILS_CLOSED_OK\n";
}
