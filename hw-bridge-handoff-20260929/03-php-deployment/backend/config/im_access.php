<?php

return [
    'sdk_app_id' => getenv('IM_SDK_APP_ID') ?: env('im_access.sdk_app_id', ''),
    'secret_key' => getenv('IM_SDK_SECRET_KEY') ?: env('im_access.secret_key', ''),
    'sig_ttl' => (int) (getenv('IM_USER_SIG_TTL') ?: 86400),
    'admin_account' => getenv('IM_ADMIN_ACCOUNT') ?: 'administrator',
    'rest_base_url' => getenv('IM_REST_BASE_URL') ?: 'https://adminapisgp.im.qcloud.com',
];
