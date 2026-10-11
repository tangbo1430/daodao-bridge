<?php

namespace app\imcallback\handler;

class RecordOnlyEventHandler implements HandlerInterface
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
        return true;
    }

    public function handle(array $payload)
    {
        return [];
    }
}
