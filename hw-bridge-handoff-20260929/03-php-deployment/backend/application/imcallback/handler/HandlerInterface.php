<?php

namespace app\imcallback\handler;

interface HandlerInterface
{
    public function category();

    public function isAsync();

    public function handle(array $payload);
}
