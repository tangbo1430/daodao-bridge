<?php

namespace app\imcallback\projector;

use app\imcallback\service\Payload;
use app\imcallback\service\ProjectionOrder;
use think\Db;

class RelationProjector implements ProjectorInterface
{
    public function project(array $event, array $payload)
    {
        $command = $event['callback_command'];
        $type = strpos($command, 'BlackList') !== false ? 'blacklist' : 'friend';
        $status = (strpos($command, 'Delete') !== false) ? 'deleted' : 'active';
        $pairs = !empty($payload['PairList']) && is_array($payload['PairList'])
            ? $payload['PairList'] : [$payload];
        $now = time();
        $source = ProjectionOrder::values($event, $payload);

        foreach ($pairs as $pair) {
            if (!is_array($pair)) {
                continue;
            }
            $owner = Payload::first($pair, ['From_Account', 'Owner_Account']);
            $peer = Payload::first($pair, ['To_Account', 'Peer_Account']);
            if ($owner === '' || $peer === '') {
                continue;
            }
            $data = [
                'status' => $status,
                'raw_json' => json_encode($pair, JSON_UNESCAPED_UNICODE),
                'updated_at' => $now,
            ] + $source;
            $where = ['owner_account' => $owner, 'peer_account' => $peer, 'relation_type' => $type];
            $existing = Db::name('im_relation_snapshot')->where($where)->find();
            if ($existing) {
                if (ProjectionOrder::isStale($existing, $source)) continue;
                Db::name('im_relation_snapshot')->where('id', $existing['id'])->update($data);
            } else {
                Db::name('im_relation_snapshot')->insert(array_merge($where, $data));
            }
        }
    }
}
