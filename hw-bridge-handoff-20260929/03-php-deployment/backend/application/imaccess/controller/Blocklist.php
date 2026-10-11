<?php

namespace app\imaccess\controller;

use app\imaccess\service\BlocklistService;
use app\imaccess\service\BlocklistException;
use app\imaccess\service\SessionService;
use think\Controller;
use think\facade\Log;

class Blocklist extends Controller
{
    public function status()
    {
        return $this->execute('status');
    }

    public function change()
    {
        return $this->execute('change');
    }

    public function listing()
    {
        return $this->execute('listing');
    }

    private function execute($action)
    {
        if (!request()->isPost()) {
            return json(['code' => 405, 'msg' => 'Method Not Allowed', 'data' => null], 405);
        }

        try {
            $service = new BlocklistService();
            $token = (string) input('login_token', '');
            if ($action === 'status') {
                $result = $service->status($token, (string) input('peer_account', ''));
            } elseif ($action === 'change') {
                $result = $service->change(
                    $token,
                    (string) input('peer_account', ''),
                    (string) input('blocked', '') === '1'
                );
            } else {
                $result = $service->listing($token);
            }
            return json($result['body'], $result['http_status']);
        } catch (\Throwable $e) {
            if ($e instanceof BlocklistException) {
                return json(['code' => $e->httpStatus(), 'msg' => $e->getMessage(), 'data' => null], $e->httpStatus());
            }
            $traceId = SessionService::traceId();
            Log::error('[IM blocklist] trace_id=' . $traceId . ' error=' . $e->getMessage());
            return json(['code' => 500, 'msg' => '屏蔽服务暂不可用', 'trace_id' => $traceId, 'data' => null], 500);
        }
    }
}
