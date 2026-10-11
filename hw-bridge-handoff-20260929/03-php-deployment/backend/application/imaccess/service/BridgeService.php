<?php

namespace app\imaccess\service;

use think\Db;
use think\facade\Log;

/** PHP facade for the Bridge account lifecycle. Secrets stay server-side. */
class BridgeService
{
    private $config;

    public function __construct(array $config = null)
    {
        $this->config = $config ?: config('im_bridge.');
    }

    public function configForLoginToken($loginToken)
    {
        $user = $this->authenticate($loginToken);
        if (!$user) return ['http_status' => 401, 'body' => ['code' => 401, 'msg' => '登录失效', 'data' => null]];
        $uid = (string) $user['uid'];
        if (!(new RegistrationService())->accountReady($uid)) return ['http_status'=>409,'body'=>['code'=>409,'msg'=>'账号已创建，聊天服务开通中，请稍后重试','data'=>['bridge_enabled'=>false,'bridge_status'=>'pending']]];
        if (!$this->allowed($uid)) {
            return ['http_status' => 200, 'body' => ['code' => 200, 'msg' => 'bridge disabled', 'data' => ['bridge_enabled' => false, 'tencent_user_id' => $uid]]];
        }

        $session = (new SessionService())->issue($loginToken, $uid);
        if ($session['http_status'] !== 200) return $session;
        $data = $session['body']['data'];
        $name = '';
        if (!empty($user['base64_nick_name'])) {
            $decoded = base64_decode($user['base64_nick_name'], true);
            $name = is_string($decoded) ? $decoded : '';
        }
        $provisionBody = [
            'tencent_user_id' => (string) $data['im_user_id'],
            'display_name' => $name,
            'avatar_url' => (string) ($user['head_pic'] ?? ''),
        ];
        $reserved=$this->callBridge('/api/v1/account/reserve', $provisionBody);
        if (isset($reserved['code']) && !in_array((int)$reserved['code'],[0,200],true)) {
            return ['http_status'=>409,'body'=>['code'=>409,'msg'=>'聊天服务暂不可用，请确认账号状态后重试','data'=>['bridge_enabled'=>false]]];
        }
        $confirmed=$this->callBridge('/api/v1/account/confirm?tencent_user_id=' . rawurlencode((string) $data['im_user_id']), []);
        if (isset($confirmed['code']) && !in_array((int)$confirmed['code'],[0,200],true)) {
            return ['http_status'=>409,'body'=>['code'=>409,'msg'=>'聊天服务开通中，请稍后重试','data'=>['bridge_enabled'=>false]]];
        }
        $data['bridge_enabled'] = true;
        $data['bridge_peer_id'] = (string) ($this->config['peer_id'] ?? '');
        $data['target'] = 'tangseng';
        return ['http_status' => 200, 'body' => ['code' => 200, 'msg' => 'success', 'data' => $data]];
    }

    // Called only after the authenticated GroupV operation succeeds. The actual
    // operator cannot be recovered from Tencent's administrator REST callback.
    public function notifyGroupRecall($uid, $groupId, $messageSeq)
    {
        if (strpos((string) $groupId, 'dd_') !== 0) return ['code' => 200];
        if ((int) $messageSeq <= 0) return ['code' => 400, 'msg' => '消息序号无效'];
        $mapping = (new AccountMappingService())->resolve((int) $uid, (string) config('im_access.sdk_app_id'));
        return $this->callBridge('/api/bridge/message_recall_notify', [
            'tencent_user_id' => (string) $mapping['im_user_id'],
            'message_key' => (string) $groupId . ':' . (string) $messageSeq,
        ]);
    }

    public function groupRecall($uid, $groupId, $messageSeq)
    {
        if (strpos((string) $groupId, 'dd_') !== 0 || (int) $messageSeq <= 0) return ['code' => 400, 'msg' => '群消息参数无效'];
        $mapping = (new AccountMappingService())->resolve((int) $uid, (string) config('im_access.sdk_app_id'));
        return $this->callBridge('/api/bridge/message_recall', [
            'tencent_user_id' => (string) $mapping['im_user_id'],
            'message_key' => (string) $groupId . ':' . (string) $messageSeq,
        ]);
    }

    public function disable($loginToken)
    {
        $user = $this->authenticate($loginToken);
        if (!$user) return ['http_status' => 401, 'body' => ['code' => 401, 'msg' => '登录失效', 'data' => null]];
        $uid = (string) $user['uid'];
        $this->callBridge('/api/v1/account/cancel?tencent_user_id=' . rawurlencode($uid), []);
        return ['http_status' => 200, 'body' => ['code' => 200, 'msg' => 'success', 'data' => ['bridge_enabled' => false]]];
    }

    /** Enqueue inside the caller's local user transaction; no network handshake. */
    public function queueAccountDeleted($uid)
    {
        if (empty($this->config['enabled']) || !$this->allowed((string) $uid)) return;
        $mapping = (new AccountMappingService())->resolve((int) $uid, (string) config('im_access.sdk_app_id'));
        $callback = config('im_callback.');
        $now = (string) time();
        $payload = [
            'CallbackCommand' => 'Bridge.AccountDeleted',
            'SdkAppid' => (string) ($callback['sdk_app_id'] ?? ''),
            'From_Account' => (string) $mapping['im_user_id'],
            'EventTime' => (int) $now,
            'RequestId' => 'bridge-delete-' . (string) $mapping['im_user_id'],
        ];
        $token = (string) ($callback['token'] ?? '');
        $query = ['RequestTime' => $now, 'Sign' => hash('sha256', $token . $now)];
        $result = (new \app\imcallback\service\CallbackService())->receive(
            json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES),
            $token, \app\imcallback\service\CallbackService::createTraceId(), $query
        );
        if ($result['http_status'] !== 200 || (int) ($result['body']['ErrorCode'] ?? -1) !== 0) {
            throw new \RuntimeException('Bridge account deletion could not be queued');
        }
    }

    public function searchUser($loginToken, $keyword)
    {
        $user = $this->authenticate($loginToken);
        if (!$user) return ['http_status' => 401, 'body' => ['code' => 401, 'msg' => '登录失效', 'data' => null]];
        $mapping = (new AccountMappingService())->resolve((int) $user['uid'], (string) config('im_access.sdk_app_id'));
        $query = http_build_query(['tencent_user_id' => (string) $mapping['im_user_id'], 'keyword' => trim((string) $keyword)]);
        return ['http_status' => 200, 'body' => $this->callBridgeGet('/api/bridge/search_user?' . $query)];
    }

    public function friendApply($loginToken, $targetUID, $remark, $vercode)
    {
        return $this->friendAction($loginToken, $targetUID, 'apply', ['remark' => (string) $remark, 'vercode' => (string) $vercode]);
    }

    public function friendAction($loginToken, $targetUID, $action, array $fields = [])
    {
        if (!in_array($action, ['apply', 'accept', 'refuse', 'delete', 'block', 'unblock'], true)) {
            return ['http_status' => 400, 'body' => ['code' => 400, 'msg' => '未知好友操作']];
        }
        $user = $this->authenticate($loginToken);
        if (!$user) return ['http_status' => 401, 'body' => ['code' => 401, 'msg' => '登录失效', 'data' => null]];
        $mapping = (new AccountMappingService())->resolve((int) $user['uid'], (string) config('im_access.sdk_app_id'));
        $body = $this->callBridge('/api/bridge/friend_' . $action, [
            'tencent_user_id' => (string) $mapping['im_user_id'],
            'target_tsdd_uid' => trim((string) $targetUID),
            'remark' => (string) ($fields['remark'] ?? ''),
            'vercode' => (string) ($fields['vercode'] ?? ''),
        ]);
        return ['http_status' => 200, 'body' => $body];
    }

    public function friendRequests($loginToken)
    {
        $user = $this->authenticate($loginToken);
        if (!$user) return ['http_status' => 401, 'body' => ['code' => 401, 'msg' => '登录失效']];
        $mapping = (new AccountMappingService())->resolve((int) $user['uid'], (string) config('im_access.sdk_app_id'));
        return ['http_status' => 200, 'body' => $this->callBridgeGet('/api/bridge/friend_requests?tencent_user_id=' . rawurlencode((string) $mapping['im_user_id']))];
    }

    private function authenticate($loginToken)
    {
        $token = trim((string) $loginToken);
        if ($token === '') return null;
        $user = Db::name('user')->where('login_token', $token)->find();
        return $user && (int) $user['login_status'] === 1 ? $user : null;
    }

    public function groupAction($loginToken, $action, array $fields)
    {
        if (!in_array($action, ['create', 'invite', 'remove', 'exit', 'disband', 'update', 'sync'], true)) {
            return ['http_status' => 400, 'body' => ['code' => 400, 'msg' => '未知群操作']];
        }
        $user = $this->authenticate($loginToken);
        if (!$user) return ['http_status' => 401, 'body' => ['code' => 401, 'msg' => '登录失效']];
        return $this->groupForUser($user, $action, $fields);
    }

    /** Internal model entry; uid comes from Common's authenticated session. */
    public function createGroupForUser($uid, $name, array $members, $requestId)
    {
        $user = Db::name('user')->where('uid', (int) $uid)->find();
        if (!$user || (int) $user['login_status'] !== 1) {
            return ['http_status' => 401, 'body' => ['code' => 401, 'msg' => '登录失效']];
        }
        return $this->groupForUser($user, 'create', [
            'name' => $name, 'members' => $members, 'request_id' => $requestId,
        ]);
    }

    private function groupForUser(array $user, $action, array $fields)
    {
        $mapping = (new AccountMappingService())->resolve((int) $user['uid'], (string) config('im_access.sdk_app_id'));
        $members = $fields['members'] ?? [];
        if (!is_array($members) || count($members) > 1000) {
            return ['http_status' => 400, 'body' => ['code' => 400, 'msg' => '群成员参数无效']];
        }
        foreach ($members as $member) {
            if (!is_string($member) || strlen($member) > 128 || $member === '') {
                return ['http_status' => 400, 'body' => ['code' => 400, 'msg' => '群成员参数无效']];
            }
        }
        $payload = ['tencent_user_id' => (string) $mapping['im_user_id'], 'members' => array_values(array_unique($members))];
        foreach (['group_id', 'request_id', 'name', 'notice', 'field', 'value'] as $key) {
            if (isset($fields[$key]) && !is_scalar($fields[$key])) return ['http_status' => 400, 'body' => ['code' => 400, 'msg' => '群参数无效']];
            $payload[$key] = (string) ($fields[$key] ?? '');
        }
        return ['http_status' => 200, 'body' => $this->callBridge('/api/bridge/group_' . $action, $payload)];
    }

    /** Provision the TangSeng shadow account immediately after local registration. */
    public function profileUser($uid)
    {
        if (empty($this->config['enabled'])) return ['skipped' => true];
        $user = Db::name('user')->where('uid', (int) $uid)->find();
        if (!$user) throw new \RuntimeException('user not found');
        $mapping = (new AccountMappingService())->resolve((int) $uid, (string) config('im_access.sdk_app_id'));
        $result = $this->callBridge('/api/bridge/profile_sync', [
            'tencent_user_id' => (string) $mapping['im_user_id'],
            'display_name' => (string) base64_decode((string) $user['base64_nick_name']),
            'avatar_url' => localpath_to_netpath((string) $user['head_pic']),
        ]);
        if (isset($result['code']) && (int) $result['code'] === 403) return ['skipped' => true];
        if (isset($result['code']) && !in_array((int) $result['code'], [0, 200], true)) throw new \RuntimeException('Bridge profile sync failed');
        return $result;
    }

    /** Provision the TangSeng shadow account immediately after local registration. */
    public function provisionUser($uid)
    {
        if (empty($this->config['enabled'])) return ['bridge_enabled' => false, 'skipped' => true];
        if (!(new RegistrationService())->accountReady($uid)) return ['bridge_enabled'=>false,'bridge_status'=>'pending'];
        $user = Db::name('user')->where('uid', (int) $uid)->find();
        if (!$user) throw new \RuntimeException('user not found');
        $mapping = (new AccountMappingService())->resolve((int) $uid, (string) config('im_access.sdk_app_id'));
        $name = '';
        if (!empty($user['base64_nick_name'])) {
            $decoded = base64_decode($user['base64_nick_name'], true);
            $name = is_string($decoded) ? $decoded : '';
        }
        $this->callBridge('/api/v1/account/provision', [
            'tencent_user_id' => (string) $mapping['im_user_id'],
            'display_name' => $name,
            'avatar_url' => (string) ($user['head_pic'] ?? ''),
        ]);
        return ['bridge_enabled' => true, 'tencent_user_id' => (string) $mapping['im_user_id']];
    }

    private function allowed($uid)
    {
        $users = $this->config['test_users'] ?? [];
        return empty($users) || in_array((string) $uid, $users, true);
    }

    public function registrationCall($action, array $payload)
    {
        if (!in_array($action, ['prepare','commit','abort'], true)) throw new \InvalidArgumentException('Unknown registration action');
        return $this->callBridge('/api/v1/account/' . $action, $payload);
    }

    private function callBridge($path, array $payload)
    {
        if (empty($this->config['enabled']) || empty($this->config['url']) || empty($this->config['secret'])) {
            throw new \RuntimeException('Bridge is not configured');
        }
        $body = json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $ts = (string) time();
        $nonce = bin2hex(random_bytes(16));
        $sign = hash_hmac('sha256', $ts . "\n" . $nonce . "\n" . $body, $this->config['secret']);
        $url = rtrim($this->config['url'], '/') . $path;
        $ch = curl_init($url);
        $headers = ['Content-Type: application/json'];
        if (strpos($path, '/api/bridge/') === 0) {
            $headers[] = 'X-Bridge-Signature: ' . hash_hmac('sha256', $body, $this->config['secret']);
        } else {
            $headers[] = 'X-HWB-Timestamp: ' . $ts; $headers[] = 'X-HWB-Nonce: ' . $nonce; $headers[] = 'X-HWB-Sign: ' . $sign;
        }
        curl_setopt_array($ch, [CURLOPT_RETURNTRANSFER => true, CURLOPT_POST => true, CURLOPT_POSTFIELDS => $body,
            CURLOPT_HTTPHEADER => $headers,
            CURLOPT_CONNECTTIMEOUT => $this->config['timeout'], CURLOPT_TIMEOUT => strpos($path, '/api/v1/account/prepare') === 0 ? 8 : (strpos($path, '/api/bridge/group_') === 0 ? 60 : $this->config['timeout'])]);
        $raw = curl_exec($ch); $error = curl_error($ch); $status = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE); curl_close($ch);
        if ($raw === false || $error !== '' || $status < 200 || $status >= 300) {
            Log::error('[IM bridge account] request failed status=' . $status . ' error=' . $error);
            $response = json_decode((string) $raw, true);
            if ($error === '' && $status >= 400 && $status < 500 && is_array($response) && isset($response['msg'])) {
                return ['code' => $status, 'msg' => (string) $response['msg'], 'data' => null];
            }
            throw new \RuntimeException('Bridge account request failed');
        }
        return $this->decodeResponse($raw);
    }

    private function callBridgeGet($path)
    {
        if (empty($this->config['enabled']) || empty($this->config['url']) || empty($this->config['secret'])) throw new \RuntimeException('Bridge is not configured');
        $url = rtrim($this->config['url'], '/') . $path;
        $sign = hash_hmac('sha256', '', $this->config['secret']);
        $ch = curl_init($url);
        curl_setopt_array($ch, [CURLOPT_RETURNTRANSFER => true, CURLOPT_HTTPGET => true, CURLOPT_HTTPHEADER => ['X-Bridge-Signature: ' . $sign], CURLOPT_CONNECTTIMEOUT => $this->config['timeout'], CURLOPT_TIMEOUT => $this->config['timeout']]);
        $raw = curl_exec($ch); $error = curl_error($ch); $status = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE); curl_close($ch);
        if ($raw === false || $error !== '' || $status < 200 || $status >= 300) throw new \RuntimeException('Bridge relation request failed');
        return $this->decodeResponse($raw);
    }

    private function decodeResponse($raw)
    {
        // HTTP 200 alone is not success: an empty/malformed response must not
        // advance account confirmation or report a completed business action.
        $object = json_decode($raw);
        if (!($object instanceof \stdClass) || count(get_object_vars($object)) === 0) {
            throw new \RuntimeException('Bridge response invalid');
        }
        return json_decode($raw, true);
    }
}
