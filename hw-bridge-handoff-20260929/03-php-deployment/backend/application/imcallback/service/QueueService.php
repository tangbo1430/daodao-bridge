<?php

namespace app\imcallback\service;

class QueueService
{
    public function enqueue($eventId)
    {
        $redis = $this->connect();
        return (bool) $redis->rPush(config('im_callback.queue_key'), (string) $eventId);
    }

    public function pop()
    {
        $redis = $this->connect();
        $value = $redis->lPop(config('im_callback.queue_key'));
        return $value === false ? 0 : (int) $value;
    }

    private function connect()
    {
        $redis = new \Redis();
        $redis->connect(getenv('REDIS_HOST') ?: '127.0.0.1', (int) (getenv('REDIS_PORT') ?: 6379), 1.0);
        return $redis;
    }
}
