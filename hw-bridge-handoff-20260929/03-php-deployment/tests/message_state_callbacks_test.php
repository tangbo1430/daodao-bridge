<?php
require '/var/www/app/thinkphp/base.php';
\think\Container::get('app')->path('/var/www/app/application/')->initialize();
foreach (['C2C.CallbackAfterMsgWithDraw', 'C2C.CallbackAfterMsgReport', 'Group.CallbackAfterRecallMsg', 'Group.CallbackAfterMemberFieldChanged'] as $command) {
    $handler = \app\imcallback\handler\HandlerRegistry::resolve($command);
    if (!$handler || !$handler->isAsync()) throw new RuntimeException('Missing asynchronous handler: ' . $command);
}
echo "MESSAGE_STATE_CALLBACKS_OK\n";
