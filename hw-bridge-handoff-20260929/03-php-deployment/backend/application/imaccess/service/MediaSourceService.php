<?php
namespace app\imaccess\service;

/** Resolve only Bridge-approved objects. Never accepts a client-supplied URL. */
class MediaSourceService
{
    public function resolve(array $config, $id, $signature, $head = false)
    {
        $headers = ['X-Bridge-Signature: ' . hash_hmac('sha256', '', $config['secret'])];
        $base = rtrim($config['url'], '/') . '/api/bridge/';
        $query = '?id=' . $id . '&sig=' . $signature;
        $fallback = ['url' => $base . 'media' . $query, 'headers' => $headers, 'resolve' => []];
        // COS preview signatures authorize GET, not HEAD. Bridge answers HEAD
        // from the already validated persisted metadata without downloading.
        if ($head) return $fallback;
        $ch = curl_init($base . 'media-source' . $query);
        curl_setopt_array($ch, [CURLOPT_HTTPHEADER => $headers, CURLOPT_RETURNTRANSFER => true,
            CURLOPT_FOLLOWLOCATION => false, CURLOPT_CONNECTTIMEOUT => 5, CURLOPT_TIMEOUT => 10]);
        $raw = curl_exec($ch);
        $status = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);
        if ($status !== 200 || !is_string($raw) || strlen($raw) > 32768) return $fallback;
        $source = $this->verify($raw, $config['secret'], $id);
        return $source ?: $fallback;
    }

    public function verify($raw, $secret, $id)
    {
        $envelope = json_decode($raw, true);
        if (!is_array($envelope) || !isset($envelope['payload'], $envelope['signature'])
            || !is_string($envelope['payload']) || !is_string($envelope['signature'])
            || !hash_equals(hash_hmac('sha256', "hw-bridge-media-source-v1\n" . $envelope['payload'], $secret), $envelope['signature'])) return null;
        $source = json_decode($envelope['payload'], true);
        if (!is_array($source) || ($source['id'] ?? '') !== $id || ($source['mode'] ?? '') !== 'public'
            || ($source['expires'] ?? 0) < time() || $source['expires'] > time() + 120) return null;
        $url = parse_url($source['url'] ?? '');
        if (!$url || ($url['scheme'] ?? '') !== 'https' || empty($url['host']) || isset($url['user'])
            || isset($url['pass']) || isset($url['fragment']) || ($url['port'] ?? 443) !== 443
            || !preg_match('/^[a-zA-Z0-9.-]+$/D', $url['host'])) return null;
        if (empty($source['addresses']) || !is_array($source['addresses'])) return null;
        $ips = [];
        foreach ($source['addresses'] as $ip) {
            if (!is_string($ip) || !filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_NO_PRIV_RANGE | FILTER_FLAG_NO_RES_RANGE)) return null;
            $ips[] = strpos($ip, ':') !== false ? '[' . $ip . ']' : $ip;
        }
        // Pin authenticated addresses. A second DNS lookup cannot rebind to an
        // internal service. No Bridge HMAC header is sent to COS/CDN.
        return ['url' => $source['url'], 'headers' => [], 'resolve' => [$url['host'] . ':443:' . implode(',', $ips)],
            'name' => (string) ($source['name'] ?? ''), 'mime' => (string) ($source['mime'] ?? '')];
    }
}
