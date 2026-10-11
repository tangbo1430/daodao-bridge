<?php

namespace app\imcallback\service;

use think\Db;

class AdminAuditService
{
    public function record($adminId, $eventId, $action, $before, $after, $result, $error = '', $traceId = '')
    {
        $traceId = $traceId ?: CallbackService::createTraceId();
        Db::name('im_callback_admin_action')->insert([
            'admin_id' => (int) $adminId, 'event_id' => (int) $eventId,
            'action' => (string) $action, 'before_status' => (string) $before,
            'after_status' => (string) $after, 'result' => (string) $result,
            'error_message' => mb_substr((string) $error, 0, 500),
            'trace_id' => $traceId, 'ip_address' => request_client_ip(), 'created_at' => time(),
        ]);
        return $traceId;
    }
}
