<?php

namespace app\imcallback\service;

use think\Db;
use think\facade\Log;

class BeforePolicyService
{
    public function evaluate(array $payload)
    {
        try {
            $command = isset($payload['CallbackCommand']) ? $payload['CallbackCommand'] : '';
            $from = Payload::first($payload, ['From_Account', 'Operator_Account']);
            $to = Payload::first($payload, ['To_Account']);

            if (in_array($command, ['C2C.CallbackBeforeSendMsg', 'Group.CallbackBeforeSendMsg'], true)) {
                if ($from !== '') {
                    $loginStatus = Db::name('user')->where('uid', $from)->value('login_status');
                    if ($loginStatus !== null && (int) $loginStatus !== 1) {
                        return $this->reject('您已被平台封禁，不能发送消息');
                    }
                }
            }

            if ($command === 'C2C.CallbackBeforeSendMsg' && $from !== '' && $to !== '') {
                if ($this->hasBlocked($from, $to)) {
                    return $this->reject('您已将对方拉黑，不能发送消息');
                }
                if ($this->hasBlocked($to, $from)) {
                    return $this->reject('您已被对方拉黑，不能发送消息');
                }
            }
        } catch (\Throwable $e) {
            Log::warning('[IM callback] before policy fail-open: ' . $e->getMessage());
        }

        return $this->allow();
    }

    private function hasBlocked($owner, $peer)
    {
        $legacy = Db::name('user_black')
            ->where(['uid' => $owner, 'receive_uid' => $peer])
            ->value('id');
        if ($legacy) {
            return true;
        }

        return (bool) Db::name('im_relation_snapshot')->where([
            'owner_account' => $owner,
            'peer_account' => $peer,
            'status' => 'active',
        ])->where('relation_type', 'in', ['blacklist', 'app_blocklist'])->value('id');
    }

    private function allow()
    {
        return ['ActionStatus' => 'OK', 'ErrorInfo' => '', 'ErrorCode' => 0];
    }

    private function reject($message)
    {
        return ['ActionStatus' => 'OK', 'ErrorInfo' => $message, 'ErrorCode' => 120001];
    }
}
