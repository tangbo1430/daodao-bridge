<?php

function localpath_to_netpath($path)
{
    return strpos((string) $path, 'http') === 0 ? $path : 'https://admin.example.test/uploads/' . ltrim($path, '/');
}

require dirname(__DIR__) . '/service/UserProfileViewService.php';

use app\common\service\UserProfileViewService;

$failures = 0;
function profile_check($condition, $message)
{
    global $failures;
    if (!$condition) {
        $failures++;
        fwrite(STDERR, "FAIL: {$message}\n");
    }
}

$service = new UserProfileViewService();
$business = [
    'uid' => 1001,
    'base64_nick_name' => base64_encode('Business User'),
    'head_pic' => 'head_pic/head_pic.png',
];
$fallback = $service->merge($business, null, '1001');
profile_check($fallback['nick_name'] === 'Business User', 'business nickname remains the fallback');
profile_check($fallback['profile_source'] === 'business', 'business source is reported');
profile_check(strpos($fallback['head_pic'], 'admin.example.test/uploads/') !== false, 'legacy avatar is normalized');

$projected = $service->merge($business, [
    'nick_name' => 'Tencent User',
    'avatar_url' => 'https://cos.example.test/avatar.png',
], '1001');
profile_check($projected['nick_name'] === 'Tencent User', 'Tencent nickname has priority');
profile_check($projected['head_pic'] === 'https://cos.example.test/avatar.png', 'Tencent avatar has priority');
profile_check($projected['profile_source'] === 'tencent', 'Tencent source is reported');

fwrite(STDOUT, $failures === 0 ? "User profile view tests passed\n" : "Failures: {$failures}\n");
exit($failures === 0 ? 0 : 1);
