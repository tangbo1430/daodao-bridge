<?php

namespace app\imcallback\service;

use think\Db;
use think\facade\Log;

class BridgeDispatchService
{
    public function dispatchEvent($eventId)
    {
        $event = Db::name('im_callback_event')->where('id', $eventId)->find();
        if (!$event) return false;

        $config = config('im_bridge.');
        if (empty($config['enabled']) || empty($config['url']) || empty($config['secret'])) {
            return false;
        }
        if (!in_array($event['callback_command'], $config['commands'], true)) {
            return false;
        }

        $this->ensureDeliveryRow($event, $config);
        $delivery = Db::name('im_bridge_delivery')->where('event_id', $eventId)->find();
        if (!$delivery || $delivery['status'] === 'done') return true;
        if ((int) $delivery['next_retry_at'] > time()) return false;

        Db::name('im_bridge_delivery')->where('event_id', $eventId)->update([
            'status' => 'processing',
            'updated_at' => time(),
        ]);

        $payload = json_decode($event['payload_json'], true);
        if (!is_array($payload)) $payload = [];
        $payload['CallbackCommand'] = $event['callback_command'];
        $payload['SdkAppid'] = $event['sdk_app_id'];
        if (!empty($event['from_account'])) $payload['From_Account'] = $event['from_account'];
        if (!empty($event['to_account'])) $payload['To_Account'] = $event['to_account'];
        if (!empty($event['group_id'])) $payload['GroupId'] = $event['group_id'];
        if (!empty($event['msg_seq'])) $payload['MsgSeq'] = (int) $event['msg_seq'];
        if (!empty($event['msg_random'])) $payload['MsgRandom'] = (int) $event['msg_random'];

        // /cb/tim receives the raw Tencent callback so the Bridge can retain
        // the official field names and query parameters for replay.
        $body = json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $timestamp = (string) time();
        $nonce = bin2hex(random_bytes(16));
        $signature = hash_hmac('sha256', $timestamp . "\n" . $nonce . "\n" . $body, $config['secret']);

        $ch = curl_init($config['url'] . $config['path']);
        curl_setopt_array($ch, [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_POST => true,
            CURLOPT_POSTFIELDS => $body,
            CURLOPT_HTTPHEADER => [
                'Content-Type: application/json',
                'X-HWB-Timestamp: ' . $timestamp,
                'X-HWB-Nonce: ' . $nonce,
                'X-HWB-Sign: ' . $signature,
                'X-TIM-Query: SdkAppid=' . rawurlencode((string) $event['sdk_app_id']),
                'X-Trace-ID: ' . (string) $event['trace_id'],
            ],
            CURLOPT_CONNECTTIMEOUT => $config['timeout'],
            CURLOPT_TIMEOUT => $config['timeout'],
        ]);
        $response = curl_exec($ch);
        $error = curl_error($ch);
        $status = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        if ($response !== false && $error === '' && $status >= 200 && $status < 300) {
            Db::name('im_bridge_delivery')->where('event_id', $eventId)->update([
                'status' => 'done', 'last_error' => '', 'updated_at' => time(), 'delivered_at' => time(),
            ]);
            return true;
        }

        $attempts = (int) $delivery['attempts'] + 1;
        $dead = $attempts >= (int) $config['max_retries'];
        $delays = [5, 30, 120, 600, 1800];
        $delayIndex = min($attempts - 1, count($delays) - 1);
        Db::name('im_bridge_delivery')->where('event_id', $eventId)->update([
            'status' => $dead ? 'dead' : 'failed',
            'attempts' => $attempts,
            'next_retry_at' => $dead ? 0 : time() + $delays[$delayIndex],
            'last_error' => mb_substr('HTTP ' . $status . ' ' . $error, 0, 1000, 'UTF-8'),
            'updated_at' => time(),
        ]);
        Log::warning('[IM bridge] dispatch failed event_id=' . $eventId . ' status=' . $status . ' error=' . $error);
        return false;
    }

    public function dispatchPending($limit = 100)
    {
        $rows = Db::name('im_bridge_delivery')
            ->where('status', 'in', ['pending', 'failed'])
            ->where('next_retry_at', '<=', time())
            ->order('id', 'asc')->limit($limit)->column('event_id');
        $done = 0;
        foreach ($rows as $eventId) if ($this->dispatchEvent($eventId)) $done++;
        return $done;
    }

    private function ensureDeliveryRow(array $event, array $config)
    {
        $exists = Db::name('im_bridge_delivery')->where('event_id', $event['id'])->find();
        if ($exists) return;
        Db::name('im_bridge_delivery')->insert([
            'event_id' => $event['id'], 'status' => 'pending', 'attempts' => 0,
            'next_retry_at' => time(), 'last_error' => '', 'created_at' => time(), 'updated_at' => time(),
        ]);
    }
}
