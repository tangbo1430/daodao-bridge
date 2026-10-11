<?php

namespace app\imaccess\service;

use think\facade\Log;

class TencentAccountClient
{
    private $config;

    public function __construct(array $config)
    {
        $this->config = $config;
    }

    public function importAccount($imUserId, $nickName = '', $faceUrl = '')
    {
        $body = ['Identifier' => (string) $imUserId];
        if ($nickName !== '') $body['Nick'] = mb_substr($nickName, 0, 20);
        if ($faceUrl !== '') $body['FaceUrl'] = $faceUrl;
        return $this->request('im_open_login_svc', 'account_import', $body);
    }

    public function accountCheck($imUserId)
    {
        return $this->request('im_open_login_svc', 'account_check', [
            'CheckItem' => [['UserID' => (string) $imUserId]],
        ]);
    }

    public function deleteAccount($imUserId)
    {
        return $this->request('im_open_login_svc', 'account_delete', [
            'DeleteItem' => [['UserID' => (string) $imUserId]],
        ]);
    }

    public function setProfile($imUserId, $nickName = '', $faceUrl = '')
    {
        $items = [];
        if ($nickName !== '') {
            $items[] = ['Tag' => 'Tag_Profile_IM_Nick', 'Value' => mb_substr($nickName, 0, 50)];
        }
        if ($faceUrl !== '') {
            $items[] = ['Tag' => 'Tag_Profile_IM_Image', 'Value' => $faceUrl];
        }
        if (empty($items)) {
            return ['ActionStatus' => 'OK', 'ErrorCode' => 0, 'ErrorInfo' => ''];
        }
        return $this->request('profile', 'portrait_set', [
            'From_Account' => (string) $imUserId,
            'ProfileItem' => $items,
        ]);
    }

    private function request($service, $command, array $body)
    {
        $sig = UserSigService::generate(
            $this->config['sdk_app_id'], $this->config['secret_key'],
            $this->config['admin_account'], 86400
        );
        $query = http_build_query([
            'sdkappid' => $this->config['sdk_app_id'],
            'identifier' => $this->config['admin_account'],
            'usersig' => $sig,
            'random' => random_int(10000000, 99999999),
            'contenttype' => 'json',
        ]);
        $url = rtrim($this->config['rest_base_url'], '/') . '/v4/' . $service . '/' . $command . '?' . $query;
        $ch = curl_init($url);
        curl_setopt_array($ch, [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_POST => true,
            CURLOPT_POSTFIELDS => json_encode($body, JSON_UNESCAPED_UNICODE),
            CURLOPT_HTTPHEADER => ['Content-Type: application/json', 'Accept: application/json'],
            CURLOPT_CONNECTTIMEOUT => 3,
            CURLOPT_TIMEOUT => 8,
        ]);
        $caFile = getenv('SSL_CERT_FILE');
        if ($caFile && is_file($caFile)) curl_setopt($ch, CURLOPT_CAINFO, $caFile);
        $raw = curl_exec($ch);
        $error = curl_error($ch);
        $status = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);
        if ($raw === false || $error !== '') {
            Log::error('[IM access] Tencent REST curl error=' . $error);
            throw new \RuntimeException('Tencent IM network request failed');
        }
        $result = json_decode($raw, true);
        if (!is_array($result)) throw new \RuntimeException('Tencent IM returned an invalid response');
        if ($status >= 400 || ($result['ActionStatus'] ?? '') !== 'OK') {
            $code = isset($result['ErrorCode']) ? (string) $result['ErrorCode'] : (string) $status;
            Log::error('[IM access] Tencent REST code=' . $code . ' info=' . (string) ($result['ErrorInfo'] ?? ''));
            throw new \RuntimeException('Tencent IM account operation failed: ' . $code);
        }
        return $result;
    }
}
