<?php

namespace app\imcallback\service;

use think\Db;

class OverviewService
{
    public function get($days)
    {
        $days = in_array((int) $days, [1, 7, 30], true) ? (int) $days : 1;
        $since = time() - $days * 86400;
        $base = Db::name('im_callback_event')->where('received_at', '>=', $since);
        $total = (int) (clone $base)->count();
        $success = (int) (clone $base)->where('handler_status', 'in', ['processed', 'ignored'])->count();
        $failed = (int) (clone $base)->where(function ($q) {
            $q->where('queue_status', 'in', ['failed', 'dead'])->whereOr('handler_status', 'failed');
        })->count();
        $dead = (int) (clone $base)->where('queue_status', 'dead')->count();
        $currentDead = (int) Db::name('im_callback_event')->where('queue_status', 'dead')->count();
        $pending = (int) Db::name('im_callback_event')->where('queue_status', 'in', ['pending', 'processing', 'failed'])->count();
        $duplicates = (int) (clone $base)->sum('duplicate_count');
        $lastReceived = (int) Db::name('im_callback_event')->max('received_at');
        $oldestPending = (int) Db::name('im_callback_event')->where('queue_status', 'in', ['pending', 'processing', 'failed'])->min('received_at');
        $waitSeconds = $oldestPending ? max(0, time() - $oldestPending) : 0;

        $durations = Db::name('im_callback_event')->where('received_at', '>=', $since)
            ->where('queue_status', 'none')->where('event_category', '<>', 'unknown')
            ->order('duration_ms', 'asc')->column('duration_ms');
        $p95 = $this->percentile($durations, 0.95);

        $format = $days === 1 ? '%Y-%m-%d %H:00' : '%Y-%m-%d';
        $trendRows = Db::query("SELECT DATE_FORMAT(FROM_UNIXTIME(received_at), ?) bucket, COUNT(*) received, SUM(handler_status IN ('processed','ignored')) success, SUM(queue_status IN ('failed','dead') OR handler_status='failed') failed, SUM(duplicate_count) duplicates FROM yy_im_callback_event WHERE received_at>=? GROUP BY bucket ORDER BY bucket", [$format, $since]);
        $categoryRows = Db::name('im_callback_event')->field('event_category,COUNT(*) total')
            ->where('received_at', '>=', $since)->group('event_category')->select();

        $heartbeat = Db::name('im_callback_worker_heartbeat')->where('worker_name', 'default')->find();
        $heartbeatAge = $heartbeat ? max(0, time() - (int) $heartbeat['last_seen_at']) : 999999;
        $redis = ['available' => false, 'queue_length' => null];
        try {
            $client = new \Redis();
            $client->connect(getenv('REDIS_HOST') ?: '127.0.0.1', (int) (getenv('REDIS_PORT') ?: 6379), 1.0);
            $redis['available'] = $client->ping() !== false;
            $redis['queue_length'] = (int) $client->lLen(config('im_callback.queue_key'));
        } catch (\Throwable $e) {
        }

        $health = 'healthy'; $reasons = [];
        if ($heartbeatAge > 30) { $health = 'critical'; $reasons[] = 'Worker 心跳异常'; }
        if ($currentDead > 0) { $health = 'critical'; $reasons[] = '存在死信事件'; }
        if ($waitSeconds > 300 || $p95 > 800) { $health = 'critical'; $reasons[] = $waitSeconds > 300 ? '事件积压超过5分钟' : '事前回调耗时过高'; }
        if ($health !== 'critical' && (!$redis['available'] || $waitSeconds > 60 || $p95 > 500)) {
            $health = 'warning';
            if (!$redis['available']) $reasons[] = 'Redis 不可用，已降级为数据库补偿';
            if ($waitSeconds > 60) $reasons[] = '事件积压超过1分钟';
            if ($p95 > 500) $reasons[] = '事前回调 P95 超过500ms';
        }

        return [
            'range_days' => $days,
            'metrics' => ['total' => $total, 'success_rate' => $total ? round($success * 100 / $total, 2) : 100,
                'failed' => $failed, 'dead' => $dead, 'duplicates' => $duplicates, 'pending' => $pending,
                'before_p95_ms' => $p95, 'last_received_at' => $lastReceived],
            'health' => ['level' => $health, 'reasons' => $reasons, 'worker_age_seconds' => $heartbeatAge,
                'oldest_pending_seconds' => $waitSeconds, 'redis' => $redis],
            'trend' => $trendRows, 'categories' => $categoryRows, 'generated_at' => time(),
        ];
    }

    private function percentile(array $values, $percentile)
    {
        if (!$values) return 0;
        $index = max(0, (int) ceil(count($values) * $percentile) - 1);
        return (int) $values[$index];
    }
}
