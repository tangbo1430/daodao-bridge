<?php

namespace app\imaccess\controller;

use app\imaccess\service\BridgeService;
use think\Controller;
use think\facade\Log;

class Bridge extends Controller
{
    public function groupCreate() { return $this->groupMutation('create'); }
    public function groupInvite() { return $this->groupMutation('invite'); }
    public function groupRemove() { return $this->groupMutation('remove'); }
    public function groupExit() { return $this->groupMutation('exit'); }
    public function groupDisband() { return $this->groupMutation('disband'); }
    public function groupUpdate() { return $this->groupMutation('update'); }
    public function groupSync() { return $this->groupMutation('sync'); }

    private function groupMutation($action)
    {
        if (!request()->isPost()) return json(['code' => 405, 'msg' => 'Method Not Allowed'], 405);
        $body = json_decode((string) file_get_contents('php://input'), true);
        if (!is_array($body)) $body = request()->post();
        try {
            $result = (new BridgeService())->groupAction((string) ($body['login_token'] ?? ''), $action, $body);
            return json($result['body'], $result['http_status']);
        } catch (\Throwable $e) {
            Log::error('[IM bridge group] operation=' . $action . ' failed');
            return json(['code' => 502, 'msg' => '群操作失败，请稍后重试'], 502);
        }
    }

    public function friendAccept() { return $this->friendMutation('accept'); }
    public function friendRefuse() { return $this->friendMutation('refuse'); }
    public function friendDelete() { return $this->friendMutation('delete'); }
    public function friendRequests() { return $this->run('friendRequests', request()->isPost() ? 'POST' : 'GET'); }

    private function friendMutation($action)
    {
        if (!request()->isPost()) return json(['code' => 405, 'msg' => 'Method Not Allowed'], 405);
        $body = json_decode((string) file_get_contents('php://input'), true);
        if (!is_array($body)) $body = [];
        try {
            $result = (new BridgeService())->friendAction(
                (string) ($body['login_token'] ?? input('login_token', '')),
                (string) ($body['target_tsdd_uid'] ?? input('target_tsdd_uid', '')), $action);
            return json($result['body'], $result['http_status']);
        } catch (\Throwable $e) {
            Log::error('[IM bridge relation] operation=' . $action . ' failed');
            return json(['code' => 502, 'msg' => '好友操作失败，请稍后重试'], 502);
        }
    }

    public function config()
    {
        return $this->run('configForLoginToken', 'GET');
    }

    public function provision()
    {
        return $this->run('configForLoginToken', 'POST');
    }

    public function disable()
    {
        if (!request()->isPost()) return json(['code' => 405, 'msg' => 'Method Not Allowed', 'data' => null], 405);
        try {
            $result = (new BridgeService())->disable((string) input('login_token', ''));
            return json($result['body'], $result['http_status']);
        } catch (\Throwable $e) {
            Log::error('[IM bridge account] ' . $e->getMessage());
            return json(['code' => 502, 'msg' => '服务暂不可用，请稍后重试', 'data' => null], 502);
        }
    }

    public function searchUser()
    {
        if (!request()->isGet() && !request()->isPost()) return json(['code' => 405, 'msg' => 'Method Not Allowed', 'data' => null], 405);
        try {
            $body = json_decode((string) file_get_contents('php://input'), true);
            if (!is_array($body)) $body = [];
            $result = (new BridgeService())->searchUser((string) ($body['login_token'] ?? input('login_token', '')), (string) ($body['keyword'] ?? input('keyword', '')));
            return json($result['body'], $result['http_status']);
        } catch (\Throwable $e) {
            Log::error('[IM bridge relation] search error=' . $e->getMessage());
            return json(['code' => 502, 'msg' => '服务暂不可用，请稍后重试', 'data' => null], 502);
        }
    }

    public function friendApply()
    {
        if (!request()->isPost()) return json(['code' => 405, 'msg' => 'Method Not Allowed', 'data' => null], 405);
        try {
            // ThinkPHP's input() does not consistently decode application/json
            // bodies in this deployment. Merge JSON fields explicitly so the
            // Android client can use its normal JSON request.
            $json = json_decode((string) file_get_contents('php://input'), true);
            if (!is_array($json)) $json = [];
            $value = function ($key, $default = '') use ($json) {
                $fromJson = array_key_exists($key, $json) ? $json[$key] : null;
                return (string) ($fromJson !== null ? $fromJson : input($key, $default));
            };
            $result = (new BridgeService())->friendApply($value('login_token'), $value('target_tsdd_uid'), $value('remark'), $value('vercode'));
            return json($result['body'], $result['http_status']);
        } catch (\Throwable $e) {
            Log::error('[IM bridge relation] apply error=' . $e->getMessage());
            return json(['code' => 502, 'msg' => '好友申请失败', 'data' => null], 502);
        }
    }

    private function run($method, $verb)
    {
        if (strtoupper($verb) === 'GET' ? !request()->isGet() : !request()->isPost()) return json(['code' => 405, 'msg' => 'Method Not Allowed', 'data' => null], 405);
        try {
            $body = json_decode((string) file_get_contents('php://input'), true);
            if (!is_array($body)) $body = [];
            $result = (new BridgeService())->{$method}((string) ($body['login_token'] ?? input('login_token', '')));
            return json($result['body'], $result['http_status']);
        } catch (\Throwable $e) {
            Log::error('[IM bridge account] ' . $e->getMessage());
            return json(['code' => 502, 'msg' => '服务暂不可用，请稍后重试', 'data' => null], 502);
        }
    }
}
