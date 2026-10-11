<?php

$requiredConfig = [
    'account_reg_type', 'customer_link', 'customer_open_status', 'customer_uid',
    'file_upload_type', 'filter_uid', 'group_video_open_status', 'group_voice_open_status',
    'huaxin_account', 'huaxin_password', 'is_maintenance', 'loacl_resource_url',
    'min_withdraw_amount', 'qiniu_access_key', 'qiniu_bucket_name', 'qiniu_cdn_url',
    'qiniu_secret_key', 'show_reg', 'sms_every_day_send_limit', 'sms_send_model',
    'smsbao_account', 'smsbao_password', 'tencent_bucket_name', 'tencent_cdn_url',
    'tencent_secret_id', 'tencent_secret_key', 'tencent_territory', 'tencentyun_im_appid',
    'tencentyun_im_key', 'tencentyun_im_private_key', 'tencentyun_im_public_key',
    'user_withdraw_rate', 'video_open_status', 'voice_open_status', 'web_site',
];
$requiredTables = [
    'admin', 'config', 'invitecode', 'sms', 'system_menu', 'user',
    'im_account_mapping', 'im_sig_audit', 'im_callback_event',
    'im_callback_worker_heartbeat', 'im_callback_admin_action', 'im_group_snapshot',
    'im_group_member_snapshot', 'im_relation_snapshot', 'im_user_profile_snapshot',
    'im_user_state_snapshot',
];

$prefix = getenv('DB_PREFIX') ?: 'yy_';
if (!preg_match('/^[a-zA-Z0-9_]+$/', $prefix)) {
    fwrite(STDERR, "Preflight failed: invalid DB_PREFIX\n");
    exit(1);
}

try {
    $pdo = new PDO(
        'mysql:host=' . (getenv('DB_HOST') ?: 'mysql')
            . ';port=' . (getenv('DB_PORT') ?: '3306')
            . ';dbname=' . getenv('DB_NAME') . ';charset=utf8mb4',
        getenv('DB_USER'),
        getenv('DB_PASSWORD'),
        [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
    );
    $configKeys = $pdo->query('SELECT `key_title` FROM `' . $prefix . 'config`')
        ->fetchAll(PDO::FETCH_COLUMN);
    $missingConfig = array_values(array_diff($requiredConfig, $configKeys));

    $tableNames = $pdo->query(
        'SELECT table_name FROM information_schema.tables WHERE table_schema=' . $pdo->quote(getenv('DB_NAME'))
    )->fetchAll(PDO::FETCH_COLUMN);
    $expectedTables = array_map(static function ($table) use ($prefix) {
        return $prefix . $table;
    }, $requiredTables);
    $missingTables = array_values(array_diff($expectedTables, $tableNames));

    $missingEnvironment = [];
    if (strtolower(trim((string) (getenv('UPLOAD_STORAGE_DRIVER') ?: ''))) === 'cos') {
        foreach (['COS_SECRET_ID', 'COS_SECRET_KEY', 'COS_REGION', 'COS_BUCKET'] as $key) {
            if (getenv($key) === false || trim((string) getenv($key)) === '') {
                $missingEnvironment[] = $key;
            }
        }
    }
    if ((getenv('SMS_SEND_MODEL') ?: '1') === '1') {
        foreach (['SMSBAO_ACCOUNT', 'SMSBAO_PASSWORD'] as $key) {
            if (getenv($key) === false || getenv($key) === '') {
                $missingEnvironment[] = $key;
            }
        }
    }

    if ($missingConfig || $missingTables || $missingEnvironment) {
        fwrite(STDERR, 'Preflight failed: ' . json_encode([
            'missing_config' => $missingConfig,
            'missing_tables' => $missingTables,
            'missing_environment' => $missingEnvironment,
        ], JSON_UNESCAPED_UNICODE) . PHP_EOL);
        exit(1);
    }
    fwrite(STDOUT, "Application preflight passed\n");
} catch (Throwable $exception) {
    fwrite(STDERR, 'Preflight failed: ' . $exception->getMessage() . PHP_EOL);
    exit(1);
}
