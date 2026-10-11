<?php

require dirname(__DIR__) . '/service/UploadStorageConfig.php';

use app\common\service\UploadStorageConfig;

$failures = 0;

function check_storage($condition, $message)
{
    global $failures;
    if (!$condition) {
        $failures++;
        fwrite(STDERR, "FAIL: {$message}\n");
    }
}

$databaseConfig = [
    'file_upload_type' => '1',
    'loacl_resource_url' => 'https://admin.example.test',
    'tencent_territory' => 'ap-guangzhou',
];

$local = new UploadStorageConfig($databaseConfig, []);
check_storage($local->driver() === 'local', 'legacy local driver remains available');
check_storage(
    $local->publicUrl('/uploads/avatar/a.png') === 'https://admin.example.test/uploads/avatar/a.png',
    'local public URL is normalized'
);

$cos = new UploadStorageConfig($databaseConfig, [
    'UPLOAD_STORAGE_DRIVER' => 'cos',
    'COS_SECRET_ID' => 'id',
    'COS_SECRET_KEY' => 'key',
    'COS_REGION' => 'ap-singapore',
    'COS_BUCKET' => 'images-1234567890',
    'COS_PUBLIC_URL' => '',
]);
check_storage($cos->driver() === 'cos', 'environment overrides legacy database driver');
check_storage($cos->missingCosSettings() === [], 'complete COS settings pass validation');
check_storage($cos->cos()['accelerate'] === false, 'regional COS endpoint remains the default');
check_storage(
    $cos->publicUrl('avatar/20260718/a.png')
        === 'https://images-1234567890.cos.ap-singapore.myqcloud.com/avatar/20260718/a.png',
    'default COS public URL is derived from bucket and region'
);
check_storage(
    $cos->publicUrl('https://cdn.example.test/existing.png') === 'https://cdn.example.test/existing.png',
    'existing absolute URLs remain unchanged'
);

$accelerated = new UploadStorageConfig($databaseConfig, [
    'UPLOAD_STORAGE_DRIVER' => 'cos',
    'COS_SECRET_ID' => 'id',
    'COS_SECRET_KEY' => 'key',
    'COS_REGION' => 'ap-hongkong',
    'COS_BUCKET' => 'images-1234567890',
    'COS_PUBLIC_URL' => 'https://images-1234567890.cos.accelerate.myqcloud.com',
]);
check_storage($accelerated->cos()['accelerate'] === true, 'accelerated public URL enables accelerated uploads');

$accelerationDisabled = new UploadStorageConfig($databaseConfig, [
    'UPLOAD_STORAGE_DRIVER' => 'cos',
    'COS_SECRET_ID' => 'id',
    'COS_SECRET_KEY' => 'key',
    'COS_REGION' => 'ap-hongkong',
    'COS_BUCKET' => 'images-1234567890',
    'COS_PUBLIC_URL' => 'https://images-1234567890.cos.accelerate.myqcloud.com',
    'COS_ACCELERATE' => 'false',
]);
check_storage($accelerationDisabled->cos()['accelerate'] === false, 'explicit setting can disable accelerated uploads');

$incomplete = new UploadStorageConfig($databaseConfig, [
    'UPLOAD_STORAGE_DRIVER' => 'cos',
    'COS_REGION' => 'ap-guangzhou',
]);
check_storage(in_array('secret_id', $incomplete->missingCosSettings(), true), 'missing credentials are detected');

fwrite(STDOUT, $failures === 0 ? "Upload storage config tests passed\n" : "Failures: {$failures}\n");
exit($failures === 0 ? 0 : 1);
