<?php

return [
    // Enable only after Bridge/PHP compatibility and the new QA APK are deployed.
    'strict_registration' => filter_var(getenv('IM_BRIDGE_STRICT_REGISTRATION') ?: 'false', FILTER_VALIDATE_BOOLEAN),
    'enabled' => (bool) (getenv('IM_BRIDGE_ENABLED') ?: false),
    'url' => rtrim((string) (getenv('IM_BRIDGE_URL') ?: ''), '/'),
    'path' => (string) (getenv('IM_BRIDGE_PATH') ?: '/cb/tim'),
    'secret' => (string) (getenv('IM_BRIDGE_SHARED_SECRET') ?: ''),
    'timeout' => max(1, (int) (getenv('IM_BRIDGE_TIMEOUT_SECONDS') ?: 5)),
    'max_retries' => max(1, (int) (getenv('IM_BRIDGE_MAX_RETRIES') ?: 5)),
    'test_users' => array_values(array_filter(array_map('trim', explode(',', (string) (getenv('IM_BRIDGE_TEST_USERS') ?: ''))))),
    'peer_id' => (string) (getenv('IM_BRIDGE_PEER_ID') ?: ''),
    'commands' => [
        'Bridge.AccountDeleted',
        'C2C.CallbackAfterSendMsg',
        'Group.CallbackAfterSendMsg',
        'C2C.CallbackAfterMsgWithDraw',
        'Group.CallbackAfterRecallMsg',
        'C2C.CallbackAfterMsgReport',
        'Sns.CallbackFriendAdd',
        'Sns.CallbackFriendDelete',
        'Sns.CallbackBlackListAdd',
        'Sns.CallbackBlackListDelete',
        'Group.CallbackAfterNewMemberJoin',
        'Group.CallbackAfterMemberExit',
        'Group.CallbackAfterGroupInfoChanged',
        'Group.CallbackAfterGroupDestroyed',
        'Group.CallbackAfterChangeGroupOwner',
        'Group.CallbackAfterMemberFieldChanged',
    ],
];
