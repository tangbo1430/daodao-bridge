<?php

namespace app\imcallback\projector;

use app\imcallback\service\Payload;
use app\imcallback\service\ProjectionOrder;
use think\Db;

class StateProjector implements ProjectorInterface
{
    public function project(array $event, array $payload)
    {
        $source = ProjectionOrder::values($event, $payload);
        $info = !empty($payload['Info']) && is_array($payload['Info']) ? $payload['Info'] : $payload;
        $items = isset($info['To_Account']) || isset($info['From_Account']) || isset($info['UserID'])
            ? [$info]
            : $info;
        foreach ($items as $item) {
            if (!is_array($item)) continue;
            $account = Payload::first($item, ['To_Account', 'From_Account', 'UserID']);
            if ($account === '') continue;
            $eventTime = Payload::first($item, ['EventTime', 'Timestamp']);
            if ($eventTime === '') $eventTime = Payload::first($payload, ['EventTime', 'Timestamp']);
            $data = [
                'state' => Payload::first($item, ['Action', 'State']),
                'reason' => Payload::first($item, ['Reason']),
                'last_event_time' => $eventTime,
                'raw_json' => json_encode($item, JSON_UNESCAPED_UNICODE),
                'updated_at' => time(),
            ] + $source;
            $existing = Db::name('im_user_state_snapshot')->where('account', $account)->find();
            if ($existing) {
                if (ProjectionOrder::isStale($existing, $source)) continue;
                Db::name('im_user_state_snapshot')->where('account', $account)->update($data);
            } else {
                $data['account'] = $account;
                Db::name('im_user_state_snapshot')->insert($data);
            }
        }
    }
}
