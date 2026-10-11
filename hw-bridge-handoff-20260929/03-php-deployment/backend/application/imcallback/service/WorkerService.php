<?php

namespace app\imcallback\service;

use app\imcallback\projector\ProjectorRegistry;
use think\Db;

class WorkerService
{
    public function processBatch($limit = 100)
    {
        try {
            (new \app\imaccess\service\RegistrationService())->retryPending(min(20,$limit));
        } catch (\Throwable $e) {
            \think\facade\Log::warning('[IM bridge registration] pending scan unavailable');
        }
        $processed = 0;
        $ids = [];

        try {
            $queue = new QueueService();
            while (count($ids) < $limit) {
                $id = $queue->pop();
                if (!$id) break;
                $ids[$id] = $id;
            }
        } catch (\Throwable $e) {
        }

        $remaining = max(0, $limit - count($ids));
        if ($remaining > 0) {
            $fallback = Db::name('im_callback_event')
                ->where('queue_status', 'in', ['pending', 'failed'])
                ->where('next_retry_at', '<=', time())
                ->order('id', 'asc')
                ->limit($remaining)
                ->column('id');
            foreach ($fallback as $id) $ids[$id] = $id;
        }

        foreach ($ids as $id) {
            if ($this->processOne($id)) $processed++;
        }
        // Bridge delivery is deliberately independent from Tencent's callback response.
        // A Bridge outage leaves a retryable delivery row and never blocks the callback worker.
        try {
            (new BridgeDispatchService())->dispatchPending($limit);
        } catch (\Throwable $e) {
            \think\facade\Log::warning('[IM bridge] pending dispatch scan failed: ' . $e->getMessage());
        }
        (new HeartbeatService())->touch($processed);
        return $processed;
    }

    public function processOne($eventId)
    {
        $claimed = Db::name('im_callback_event')
            ->where('id', $eventId)
            ->where('queue_status', 'in', ['pending', 'failed'])
            ->update(['queue_status' => 'processing']);
        if (!$claimed) return false;

        $event = Db::name('im_callback_event')->where('id', $eventId)->find();
        $startedAt = microtime(true);
        try {
            $payload = json_decode($event['payload_json'], true);
            if (!is_array($payload)) throw new \RuntimeException('Stored callback JSON is invalid');
            $projector = ProjectorRegistry::resolve($event['event_category']);
            if ($projector) $projector->project($event, $payload);
            Db::name('im_callback_event')->where('id', $eventId)->update([
                'handler_status' => 'processed',
                'queue_status' => 'done',
                'processed_at' => time(),
                'duration_ms' => (int) round((microtime(true) - $startedAt) * 1000),
                'error_message' => '',
            ]);
            try {
                (new BridgeDispatchService())->dispatchEvent($eventId);
            } catch (\Throwable $e) {
                \think\facade\Log::warning('[IM bridge] event dispatch failed: ' . $e->getMessage());
            }
            return true;
        } catch (\Throwable $e) {
            $retry = (int) $event['retry_count'] + 1;
            $max = (int) config('im_callback.max_retries');
            $delays = [5, 30, 120, 600, 1800];
            $dead = $retry >= $max;
            Db::name('im_callback_event')->where('id', $eventId)->update([
                'handler_status' => 'failed',
                'queue_status' => $dead ? 'dead' : 'failed',
                'retry_count' => $retry,
                'next_retry_at' => $dead ? 0 : time() + $delays[min($retry - 1, count($delays) - 1)],
                'processed_at' => time(),
                'duration_ms' => (int) round((microtime(true) - $startedAt) * 1000),
                'error_message' => mb_substr($e->getMessage(), 0, 1000, 'UTF-8'),
            ]);
            return false;
        }
    }
}
