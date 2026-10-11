<?php

namespace app\imaccess\service;

use think\Db;
use think\facade\Log;

class SessionService
{
    private $config;
    private $accountClient;

    public function __construct(array $config = null, $accountClient = null)
    {
        $this->config = $config ?: config('im_access.');
        $this->accountClient = $accountClient ?: new TencentAccountClient($this->config);
    }

    public function issue($loginToken, $requestedUid = null)
    {
        $traceId = self::traceId();
        $uid = 0;
        $imUserId = '';
        $expiresAt = 0;
        try {
            $this->validateConfig();
            $user = Db::name('user')->where('login_token', trim((string) $loginToken))->find();
            if (!$user) return $this->failure($traceId, 401, '登录失效');
            $uid = (int) $user['uid'];
            if ((int) $user['login_status'] !== 1) return $this->failure($traceId, 403, '用户已被封禁', $uid);
            if ($requestedUid !== null && (string) $requestedUid !== '' && (int) $requestedUid !== $uid) {
                return $this->failure($traceId, 403, '不能为其他用户签发 UserSig', $uid);
            }

            $mappingService = new AccountMappingService();
            $mapping = $mappingService->resolve($uid, (string) $this->config['sdk_app_id']);
            $imUserId = (string) $mapping['im_user_id'];
            if ($mapping['import_status'] !== 'imported') {
                try {
                    $nick = isset($user['base64_nick_name']) ? base64_decode($user['base64_nick_name'], true) : '';
                    $nick = is_string($nick) ? $nick : '';
                    $this->accountClient->importAccount($imUserId, $nick, '');
                    Db::name('im_account_mapping')->where('id', $mapping['id'])->update([
                        'import_status' => 'imported', 'import_error' => '',
                        'imported_at' => time(), 'updated_at' => time(),
                    ]);
                } catch (\Throwable $e) {
                    Db::name('im_account_mapping')->where('id', $mapping['id'])->update([
                        'import_status' => 'failed',
                        'import_error' => mb_substr($e->getMessage(), 0, 1000),
                        'updated_at' => time(),
                    ]);
                    throw $e;
                }
            }

            $ttl = max(300, (int) $this->config['sig_ttl']);
            $sig = UserSigService::generate(
                $this->config['sdk_app_id'], $this->config['secret_key'], $imUserId, $ttl
            );
            $expiresAt = time() + $ttl;
            $this->audit($traceId, $uid, $imUserId, 'success', '', $expiresAt);
            return ['http_status' => 200, 'body' => [
                'code' => 200, 'msg' => '签发成功', 'trace_id' => $traceId,
                'data' => [
                    'sdk_app_id' => (string) $this->config['sdk_app_id'],
                    'im_user_id' => $imUserId,
                    'user_sig' => $sig,
                    'expires_in' => $ttl,
                    'expires_at' => $expiresAt,
                ],
            ]];
        } catch (\Throwable $e) {
            Log::error('[IM access] trace_id=' . $traceId . ' error=' . $e->getMessage());
            $message = strpos($e->getMessage(), 'Tencent IM') === 0 ? $e->getMessage() : 'IM session service unavailable';
            return $this->failure($traceId, 502, $message, $uid, $imUserId);
        }
    }

    public static function traceId()
    {
        return 'ims-' . date('YmdHis') . '-' . bin2hex(random_bytes(8));
    }

    private function validateConfig()
    {
        if (empty($this->config['sdk_app_id']) || empty($this->config['secret_key'])) {
            throw new \RuntimeException('IM access configuration is incomplete');
        }
    }

    private function failure($traceId, $status, $message, $uid = 0, $imUserId = '')
    {
        $this->audit($traceId, $uid, $imUserId, 'failed', $message, 0);
        return ['http_status' => $status, 'body' => [
            'code' => $status, 'msg' => $message, 'trace_id' => $traceId, 'data' => null,
        ]];
    }

    private function audit($traceId, $uid, $imUserId, $result, $error, $expiresAt)
    {
        try {
            Db::name('im_sig_audit')->insert([
                'trace_id' => $traceId, 'uid' => (int) $uid,
                'im_user_id' => (string) $imUserId,
                'sdk_app_id' => (string) ($this->config['sdk_app_id'] ?? ''),
                'result' => $result, 'error_message' => mb_substr((string) $error, 0, 500),
                'issued_at' => time(), 'expires_at' => (int) $expiresAt,
            ]);
        } catch (\Throwable $ignored) {
        }
    }
}
