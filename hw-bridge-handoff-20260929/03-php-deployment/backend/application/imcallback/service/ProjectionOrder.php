<?php

namespace app\imcallback\service;

class ProjectionOrder
{
    public static function values(array $event, array $payload)
    {
        $value = Payload::first($payload, ['EventTime', 'MsgTime', 'Timestamp']);
        if ($value === '') $value = isset($event['event_time']) ? (string) $event['event_time'] : '';
        $time = is_numeric($value) ? (int) $value : 0;
        if ($time > 0 && $time < 100000000000) $time *= 1000;
        if ($time <= 0) $time = ((int) (isset($event['received_at']) ? $event['received_at'] : time())) * 1000;
        return ['source_event_time' => $time, 'source_event_id' => (int) (isset($event['id']) ? $event['id'] : 0)];
    }

    public static function isStale(array $existing, array $source)
    {
        $oldTime = (int) (isset($existing['source_event_time']) ? $existing['source_event_time'] : 0);
        $oldId = (int) (isset($existing['source_event_id']) ? $existing['source_event_id'] : 0);
        return $oldTime > (int) $source['source_event_time'] ||
            ($oldTime === (int) $source['source_event_time'] && $oldId > (int) $source['source_event_id']);
    }
}
