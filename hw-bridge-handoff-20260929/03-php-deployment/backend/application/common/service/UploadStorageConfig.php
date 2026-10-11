<?php

namespace app\common\service;

class UploadStorageConfig
{
    private $systemConfig;
    private $environment;

    public function __construct(array $systemConfig, $environment = null)
    {
        $this->systemConfig = $systemConfig;
        $this->environment = is_array($environment) ? $environment : $this->readEnvironment();
    }

    public function driver()
    {
        $driver = strtolower(trim((string) $this->environmentValue('UPLOAD_STORAGE_DRIVER')));
        if (in_array($driver, ['local', 'qiniu', 'cos'], true)) {
            return $driver;
        }

        $legacyType = (int) $this->configValue('file_upload_type', 1);
        return [1 => 'local', 2 => 'qiniu', 3 => 'cos'][$legacyType] ?? 'local';
    }

    public function cos()
    {
        $region = $this->value('COS_REGION', 'tencent_territory', 'ap-guangzhou');
        $bucket = $this->value('COS_BUCKET', 'tencent_bucket_name', '');
        $publicUrl = rtrim($this->value('COS_PUBLIC_URL', 'tencent_cdn_url', ''), '/');
        if ($publicUrl === '' && $bucket !== '' && $region !== '') {
            $publicUrl = 'https://' . $bucket . '.cos.' . $region . '.myqcloud.com';
        }

        return [
            'secret_id' => $this->value('COS_SECRET_ID', 'tencent_secret_id', ''),
            'secret_key' => $this->value('COS_SECRET_KEY', 'tencent_secret_key', ''),
            'region' => $region,
            'bucket' => $bucket,
            'public_url' => $publicUrl,
            'accelerate' => $this->accelerateEnabled($publicUrl),
        ];
    }

    public function missingCosSettings()
    {
        $cos = $this->cos();
        $missing = [];
        foreach (['secret_id', 'secret_key', 'region', 'bucket', 'public_url'] as $key) {
            if ($cos[$key] === '') {
                $missing[] = $key;
            }
        }
        return $missing;
    }

    public function publicUrl($path)
    {
        $path = str_replace('\\', '/', trim((string) $path));
        if ($path === '' || preg_match('#^https?://#i', $path)) {
            return $path;
        }

        if ($this->driver() === 'cos') {
            $baseUrl = $this->cos()['public_url'];
        } elseif ($this->driver() === 'qiniu') {
            $baseUrl = (string) $this->configValue('qiniu_cdn_url', '');
        } else {
            $baseUrl = (string) $this->configValue('loacl_resource_url', '');
        }
        return rtrim($baseUrl, '/') . '/' . ltrim($path, '/');
    }

    private function value($environmentKey, $configKey, $default)
    {
        $environmentValue = trim((string) $this->environmentValue($environmentKey));
        return $environmentValue !== ''
            ? $environmentValue
            : trim((string) $this->configValue($configKey, $default));
    }

    private function environmentValue($key)
    {
        return array_key_exists($key, $this->environment) ? $this->environment[$key] : '';
    }

    private function configValue($key, $default)
    {
        return array_key_exists($key, $this->systemConfig) ? $this->systemConfig[$key] : $default;
    }

    private function readEnvironment()
    {
        $environment = [];
        foreach (['UPLOAD_STORAGE_DRIVER', 'COS_SECRET_ID', 'COS_SECRET_KEY', 'COS_REGION', 'COS_BUCKET', 'COS_PUBLIC_URL', 'COS_ACCELERATE'] as $key) {
            $value = getenv($key);
            $environment[$key] = $value === false ? '' : $value;
        }
        return $environment;
    }

    private function accelerateEnabled($publicUrl)
    {
        $configured = trim((string) $this->environmentValue('COS_ACCELERATE'));
        if ($configured !== '') {
            return filter_var($configured, FILTER_VALIDATE_BOOLEAN);
        }

        return stripos((string) $publicUrl, '.cos.accelerate.myqcloud.com') !== false;
    }
}
