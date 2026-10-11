<?php

namespace app\imcallback\service;

use app\imcallback\handler\HandlerRegistry;
use think\Db;
use think\facade\Log;

class CallbackService
{
    public function receive($rawBody, $requestToken, $traceId, array $query = [])
    {
        $startedAt = microtime(true);
        $config = config('im_callback.');
        $expectedToken = isset($config['token']) ? (string) $config['token'] : '';
        $expectedAppId = isset($config['sdk_app_id']) ? (string) $config['sdk_app_id'] : '';

        if ($expectedToken === '' || $expectedAppId === '') {
            Log::error('[IM callback] Missing IM_CALLBACK_TOKEN or IM_SDK_APP_ID');
            return $this->failure(500, 50001, 'Callback service is not configured');
        }

        if ($requestToken === '' || !hash_equals($expectedToken, $requestToken)) {
            return $this->failure(403, 40301, 'Invalid callback token');
        }

        if (!empty($config['verify_signature'])) {
            $requestTime = isset($query['RequestTime']) ? (string) $query['RequestTime'] : '';
            $signature = isset($query['Sign']) ? (string) $query['Sign'] : '';
            if ($requestTime === '' || $signature === '' || !ctype_digit($requestTime) || abs(time() - (int) $requestTime) > 60) {
                return $this->failure(403, 40303, 'Invalid callback signature timestamp');
            }
            $expectedSignature = hash('sha256', $expectedToken . $requestTime);
            if (!hash_equals($expectedSignature, $signature)) {
                return $this->failure(403, 40304, 'Invalid callback signature');
            }
        }

        if (!is_string($rawBody) || trim($rawBody) === '') {
            return $this->failure(400, 40001, 'Empty callback body');
        }

        $payload = json_decode($rawBody, true);
        if (!is_array($payload) || json_last_error() !== JSON_ERROR_NONE) {
            return $this->failure(400, 40002, 'Invalid callback JSON');
        }

        // Tencent sends callback metadata in the query string for real events.
        foreach (['CallbackCommand', 'SdkAppid', 'ClientIP', 'OptPlatform', 'RequestId'] as $key) {
            if (isset($query[$key]) && $query[$key] !== '') {
                $payload[$key] = $query[$key];
            }
        }

        $command = isset($payload['CallbackCommand']) ? trim((string) $payload['CallbackCommand']) : '';
        $sdkAppId = isset($payload['SdkAppid']) ? (string) $payload['SdkAppid'] : '';

        if ($command === '') {
            return $this->failure(400, 40003, 'CallbackCommand is required');
        }

        if ($sdkAppId === '' || !hash_equals($expectedAppId, $sdkAppId)) {
            return $this->failure(403, 40302, 'Invalid SdkAppid');
        }

        $eventKey = $this->createEventKey($payload, $rawBody);
        $existing = Db::name('im_callback_event')->where('event_key', $eventKey)->find();
        if ($existing) {
            Db::name('im_callback_event')->where('id', $existing['id'])->update([
                'duplicate_count' => Db::raw('duplicate_count + 1'),
                'last_received_at' => time(),
            ]);
            if ($this->requiresImmediateProjection($command)) {
                (new WorkerService())->processOne($existing['id']);
            }
            if (!empty($existing['response_json'])) {
                $storedResponse = json_decode($existing['response_json'], true);
                if (is_array($storedResponse)) {
                    return ['http_status' => 200, 'body' => $storedResponse];
                }
            }
            return $this->success();
        }

        $handler = HandlerRegistry::resolve($command);
        $now = time();
        $row = $this->buildEventRow($payload, $rawBody, $eventKey, $traceId, $now);
        $row['handler_status'] = $handler ? 'received' : 'ignored';
        $row['event_category'] = $handler ? $handler->category() : 'unknown';
        $row['queue_status'] = $handler && $handler->isAsync() ? 'pending' : 'none';

        try {
            $eventId = Db::name('im_callback_event')->insertGetId($row);
        } catch (\Throwable $e) {
            $duplicate = Db::name('im_callback_event')->where('event_key', $eventKey)->find();
            if ($duplicate) {
                Db::name('im_callback_event')->where('id', $duplicate['id'])->update([
                    'duplicate_count' => Db::raw('duplicate_count + 1'),
                    'last_received_at' => time(),
                ]);
                return $this->success();
            }
            throw $e;
        }

        if (!$handler) {
            return $this->success();
        }

        if ($handler->isAsync()) {
            // Blacklist state is read by the synchronous before-send policy. Project it
            // before acknowledging Tencent so a message cannot slip through the worker gap.
            if ($this->requiresImmediateProjection($command)) {
                $processed = (new WorkerService())->processOne($eventId);
                if ($processed) {
                    Db::name('im_callback_event')->where('id', $eventId)->update([
                        'duration_ms' => $this->durationMs($startedAt),
                    ]);
                    return $this->success();
                }
            }

            try {
                (new QueueService())->enqueue($eventId);
            } catch (\Throwable $e) {
                Log::warning(sprintf(
                    '[IM callback] queue unavailable event_id=%s trace_id=%s error=%s',
                    $eventId,
                    $traceId,
                    $e->getMessage()
                ));
            }

            Db::name('im_callback_event')->where('id', $eventId)->update([
                'duration_ms' => $this->durationMs($startedAt),
            ]);
            return $this->success();
        }

        try {
            $handlerResult = $handler->handle($payload);
            $response = isset($handlerResult['response'])
                ? $handlerResult['response']
                : $this->success()['body'];
            Db::name('im_callback_event')->where('id', $eventId)->update([
                'handler_status' => 'processed',
                'processed_at' => time(),
                'response_json' => json_encode($response, JSON_UNESCAPED_UNICODE),
                'duration_ms' => $this->durationMs($startedAt),
            ]);

            return ['http_status' => 200, 'body' => $response];
        } catch (\Throwable $e) {
            Db::name('im_callback_event')->where('id', $eventId)->update([
                'handler_status' => 'failed',
                'error_message' => substr($e->getMessage(), 0, 1000),
                'processed_at' => time(),
            ]);
            throw $e;
        }
    }

    public static function createTraceId()
    {
        try {
            return bin2hex(random_bytes(16));
        } catch (\Throwable $e) {
            return str_replace('.', '', uniqid('', true));
        }
    }

    private function buildEventRow(array $payload, $rawBody, $eventKey, $traceId, $now)
    {
        return [
            'event_key' => $eventKey,
            'trace_id' => $traceId,
            'request_id' => $this->value($payload, ['RequestId']),
            'callback_command' => (string) $payload['CallbackCommand'],
            'event_category' => 'unknown',
            'sdk_app_id' => (string) $payload['SdkAppid'],
            'event_time' => $this->value($payload, ['EventTime', 'MsgTime']),
            'from_account' => $this->value($payload, ['From_Account', 'Operator_Account']),
            'to_account' => $this->value($payload, ['To_Account']),
            'group_id' => $this->value($payload, ['GroupId']),
            'msg_seq' => $this->value($payload, ['MsgSeq']),
            'msg_random' => $this->value($payload, ['MsgRandom']),
            'payload_json' => $rawBody,
            'summary' => Payload::messageSummary($payload),
            'response_json' => null,
            'handler_status' => 'received',
            'queue_status' => 'none',
            'retry_count' => 0,
            'next_retry_at' => 0,
            'error_message' => '',
            'duplicate_count' => 0,
            'received_at' => $now,
            'last_received_at' => $now,
            'processed_at' => 0,
            'duration_ms' => 0,
        ];
    }

    private function createEventKey(array $payload, $rawBody)
    {
        $parts = [
            isset($payload['SdkAppid']) ? $payload['SdkAppid'] : '',
            isset($payload['CallbackCommand']) ? $payload['CallbackCommand'] : '',
            $this->value($payload, ['From_Account', 'Operator_Account']),
            $this->value($payload, ['To_Account']),
            $this->value($payload, ['GroupId']),
            $this->value($payload, ['MsgSeq']),
            $this->value($payload, ['MsgRandom']),
            $this->value($payload, ['EventTime', 'MsgTime']),
            hash('sha256', $rawBody),
        ];

        return hash('sha256', implode('|', $parts));
    }

    private function value(array $payload, array $keys)
    {
        foreach ($keys as $key) {
            if (isset($payload[$key]) && !is_array($payload[$key])) {
                return (string) $payload[$key];
            }
        }
        return '';
    }

    private function success()
    {
        return [
            'http_status' => 200,
            'body' => [
                'ActionStatus' => 'OK',
                'ErrorInfo' => '',
                'ErrorCode' => 0,
            ],
        ];
    }

    private function failure($httpStatus, $errorCode, $errorInfo)
    {
        return [
            'http_status' => $httpStatus,
            'body' => [
                'ActionStatus' => 'FAIL',
                'ErrorInfo' => $errorInfo,
                'ErrorCode' => $errorCode,
            ],
        ];
    }

    private function durationMs($startedAt)
    {
        return max(0, (int) round((microtime(true) - $startedAt) * 1000));
    }

    private function requiresImmediateProjection($command)
    {
        return in_array($command, [
            'Sns.CallbackBlackListAdd',
            'Sns.CallbackBlackListDelete',
        ], true);
    }
}
