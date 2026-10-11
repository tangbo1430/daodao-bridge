<?php

namespace app\imaccess\service;

class UserSigService
{
    public static function generate($sdkAppId, $secretKey, $userId, $ttl)
    {
        self::loadSdk();
        $api = new \Tencent\TLSSigAPIv2((int) $sdkAppId, (string) $secretKey);
        return $api->genUserSig((string) $userId, (int) $ttl);
    }

    private static function loadSdk()
    {
        if (class_exists('Tencent\\TLSSigAPIv2')) return;
        $file = dirname(__DIR__, 3) . '/vendor/tencent/tls-sig-api-v2/src/TLSSigAPIv2.php';
        if (!is_file($file)) throw new \RuntimeException('Tencent IM signing SDK is unavailable');
        require_once $file;
    }
}
