<?php

namespace app\imaccess\controller;

use app\imaccess\service\SessionService;
use think\Controller;
use think\facade\Log;

class Session extends Controller
{
    public function index()
    {
        if (!request()->isPost()) {
            return json(['code' => 405, 'msg' => 'Method Not Allowed', 'data' => null], 405);
        }
        try {
            $result = (new SessionService())->issue(
                (string) input('login_token', ''),
                input('?uid') ? input('uid') : null
            );
            return json($result['body'], $result['http_status']);
        } catch (\Throwable $e) {
            $traceId = SessionService::traceId();
            Log::error('[IM access] trace_id=' . $traceId . ' error=' . $e->getMessage());
            return json(['code' => 500, 'msg' => 'IM session service unavailable', 'trace_id' => $traceId, 'data' => null], 500);
        }
    }
}
