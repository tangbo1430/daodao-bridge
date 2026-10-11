<?php

namespace app\imcallback\handler;

use app\imcallback\service\BeforePolicyService;

class AllowBeforeEventHandler implements HandlerInterface
{
    private $category;

    public function __construct($category)
    {
        $this->category = $category;
    }

    public function category()
    {
        return $this->category;
    }

    public function isAsync()
    {
        return false;
    }

    public function handle(array $payload)
    {
        $response = (new BeforePolicyService())->evaluate($payload);

        return [
            'response' => $response,
        ];
    }
}
