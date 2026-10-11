<?php

namespace app\imcallback\service;

class Payload
{
    public static function first(array $payload, array $keys, $default = '')
    {
        foreach ($keys as $key) {
            if (isset($payload[$key]) && !is_array($payload[$key])) {
                return (string) $payload[$key];
            }
        }
        return $default;
    }

    public static function messageSummary(array $payload)
    {
        if (empty($payload['MsgBody']) || !is_array($payload['MsgBody'])) {
            return '';
        }

        $parts = [];
        foreach ($payload['MsgBody'] as $element) {
            $type = isset($element['MsgType']) ? $element['MsgType'] : 'Unknown';
            $content = isset($element['MsgContent']) && is_array($element['MsgContent'])
                ? $element['MsgContent'] : [];
            if ($type === 'TIMTextElem' && isset($content['Text'])) {
                $parts[] = (string) $content['Text'];
            } else {
                $parts[] = '[' . $type . ']';
            }
        }

        return mb_substr(implode(' ', $parts), 0, 200, 'UTF-8');
    }

    public static function accounts(array $value)
    {
        $accounts = [];
        foreach ($value as $item) {
            if (is_string($item) || is_numeric($item)) {
                $accounts[] = (string) $item;
            } elseif (is_array($item)) {
                $account = self::first($item, ['Member_Account', 'To_Account', 'From_Account', 'UserID']);
                if ($account !== '') {
                    $accounts[] = $account;
                }
            }
        }
        return array_values(array_unique($accounts));
    }
}
