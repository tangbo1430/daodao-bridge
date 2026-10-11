<?php

namespace app\imcallback\service;

use think\Db;

class HeartbeatService
{
    private static $lastTouch = 0;

    public function touch($count = 0)
    {
        $now = time();
        if (self::$lastTouch > 0 && $now - self::$lastTouch < 10) return;
        $data = [
            'host_name' => (string) gethostname(),
            'process_id' => function_exists('getmypid') ? (int) getmypid() : 0,
            'last_seen_at' => $now,
            'last_batch_count' => (int) $count,
            'updated_at' => $now,
        ];
        $exists = Db::name('im_callback_worker_heartbeat')->where('worker_name', 'default')->find();
        if ($exists) Db::name('im_callback_worker_heartbeat')->where('worker_name', 'default')->update($data);
        else Db::name('im_callback_worker_heartbeat')->insert(array_merge(['worker_name' => 'default'], $data));
        self::$lastTouch = $now;
    }
}
