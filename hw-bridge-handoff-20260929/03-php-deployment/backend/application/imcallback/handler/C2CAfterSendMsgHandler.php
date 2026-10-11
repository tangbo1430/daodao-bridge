<?php

namespace app\imcallback\handler;

class C2CAfterSendMsgHandler implements HandlerInterface
{
    public function category()
    {
        return 'message';
    }

    public function isAsync()
    {
        return true;
    }

    public function handle(array $payload)
    {
        if (empty($payload['From_Account']) || empty($payload['To_Account'])) {
            throw new \InvalidArgumentException('C2C callback accounts are required');
        }

        return [];
    }
}
