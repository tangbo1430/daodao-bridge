<?php

namespace app\imcallback\controller;

use app\imcallback\service\CallbackService;
use think\Controller;
use think\facade\Log;

class Tencent extends Controller
{
    public function receive()
    {
        $traceId = CallbackService::createTraceId();

        try {
            $service = new CallbackService();
            $result = $service->receive(
                request()->getContent(),
                (string) input('get.token', ''),
                $traceId,
                request()->get()
            );

            return json($result['body'], $result['http_status']);
        } catch (\Throwable $e) {
            Log::error(sprintf(
                '[IM callback] trace_id=%s error=%s',
                $traceId,
                $e->getMessage()
            ));

            return json([
                'ActionStatus' => 'FAIL',
                'ErrorInfo' => 'Internal callback error. TraceID: ' . $traceId,
                'ErrorCode' => 50000,
            ], 500);
        }
    }
}
