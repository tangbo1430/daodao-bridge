<?php

namespace app\common\service;

class ClientIpResolver
{
    public static function resolve(array $server)
    {
        $remote = trim((string) ($server['REMOTE_ADDR'] ?? ''));
        $internal = trim((string) ($server['HTTP_X_INTERNAL_CLIENT_IP'] ?? ''));

        if (self::isTrustedGateway($remote) && self::isIp($internal)) {
            return $internal;
        }

        return self::isIp($remote) ? $remote : '0.0.0.0';
    }

    private static function isIp($value)
    {
        return $value !== '' && filter_var($value, FILTER_VALIDATE_IP) !== false;
    }

    private static function isTrustedGateway($ip)
    {
        if (!self::isIp($ip)) {
            return false;
        }

        if (filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV4)) {
            $number = ip2long($ip);
            if ($number === false) {
                return false;
            }
            $number = (int) sprintf('%u', $number);
            return self::ipv4InRange($number, '10.0.0.0', 8)
                || self::ipv4InRange($number, '172.16.0.0', 12)
                || self::ipv4InRange($number, '192.168.0.0', 16)
                || self::ipv4InRange($number, '127.0.0.0', 8);
        }

        $binary = inet_pton($ip);
        if ($binary === inet_pton('::1')) {
            return true;
        }
        $first = ord($binary[0]);
        $second = ord($binary[1]);
        return ($first & 0xfe) === 0xfc
            || ($first === 0xfe && ($second & 0xc0) === 0x80);
    }

    private static function ipv4InRange($number, $network, $prefix)
    {
        $networkNumber = (int) sprintf('%u', ip2long($network));
        $mask = (0xffffffff << (32 - $prefix)) & 0xffffffff;
        return ($number & $mask) === ($networkNumber & $mask);
    }
}
